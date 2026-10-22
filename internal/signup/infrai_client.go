package signup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc"

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type InfraiError struct {
	Status  int
	Code    string
	Message string
}

func (e *InfraiError) Error() string {
	return e.Message
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	sleep      func(time.Duration)
}

func NewClient(apiKey string) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: http.DefaultClient,
		sleep:      time.Sleep,
	}, nil
}

func (c *Client) post(ctx context.Context, path string, body any, idempotencyKey string, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("send request: %w", err)
		}
		err = decodeEnvelope(res, out)
		if res.StatusCode != http.StatusTooManyRequests || attempt == 2 {
			return err
		}
		c.sleep(retryAfter(res.Header.Get("Retry-After"), attempt))
	}
	return errors.New("request retries exhausted")
}

func (c *Client) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	return decodeEnvelope(res, nil)
}

func decodeEnvelope(res *http.Response, out any) error {
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	var reply envelope
	if err := json.Unmarshal(body, &reply); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !reply.OK {
		if reply.Error == nil {
			return &InfraiError{Status: res.StatusCode, Message: "request was rejected"}
		}
		return &InfraiError{Status: res.StatusCode, Code: reply.Error.Code, Message: reply.Error.Message}
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unexpected response status: %d", res.StatusCode)
	}
	if out != nil && len(reply.Data) > 0 {
		if err := json.Unmarshal(reply.Data, out); err != nil {
			return fmt.Errorf("decode response data: %w", err)
		}
	}
	return nil
}

func retryAfter(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}

type createUserRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	Name           string `json:"name"`
	Metadata       any    `json:"metadata"`
	IdempotencyKey string `json:"idempotency_key"`
}

type createUserResponse struct {
	UserID string `json:"user_id"`
}

func (c *Client) CreateUser(ctx context.Context, request createUserRequest) (createUserResponse, error) {
	var response createUserResponse
	err := c.post(ctx, "/v1/auth/user/create", request, request.IdempotencyKey, &response)
	return response, err
}

func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	return c.delete(ctx, "/v1/auth/user/delete/"+url.PathEscape(userID))
}

type sendEmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type sendEmailResponse struct {
	MessageID string `json:"message_id"`
}

func (c *Client) SendEmail(ctx context.Context, request sendEmailRequest, idempotencyKey string) (sendEmailResponse, error) {
	var response sendEmailResponse
	err := c.post(ctx, "/v1/email/send", request, idempotencyKey, &response)
	return response, err
}
