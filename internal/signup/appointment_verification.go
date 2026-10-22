package signup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"strings"
)

type SignupRequest struct {
	Email                string `json:"email"`
	Password             string `json:"password"`
	Name                 string `json:"name"`
	AppointmentReference string `json:"appointment_reference"`
}

type SignupResult struct {
	UserID    string `json:"user_id"`
	MessageID string `json:"message_id"`
	State     string `json:"state"`
}

type UserCreator interface {
	CreateUser(context.Context, createUserRequest) (createUserResponse, error)
	DeleteUser(context.Context, string) error
}

type EmailSender interface {
	SendEmail(context.Context, sendEmailRequest, string) (sendEmailResponse, error)
}

type VerificationService struct {
	users     UserCreator
	emails    EmailSender
	publicURL string
}

func NewVerificationService(users UserCreator, emails EmailSender, publicURL string) *VerificationService {
	return &VerificationService{users: users, emails: emails, publicURL: strings.TrimRight(publicURL, "/")}
}

func (s *VerificationService) Start(ctx context.Context, request SignupRequest) (SignupResult, error) {
	if strings.TrimSpace(request.Email) == "" || strings.TrimSpace(request.Password) == "" || strings.TrimSpace(request.Name) == "" {
		return SignupResult{}, fmt.Errorf("email, password, and name are required")
	}
	key := signupKey(request.Email, request.AppointmentReference)
	user, err := s.users.CreateUser(ctx, createUserRequest{
		Email: request.Email, Password: request.Password, Name: request.Name,
		Metadata:       map[string]string{"appointment_reference": request.AppointmentReference},
		IdempotencyKey: key,
	})
	if err != nil {
		return SignupResult{}, err
	}
	link := s.publicURL + "/verify-email?user_id=" + template.URLQueryEscaper(user.UserID)
	email, err := s.emails.SendEmail(ctx, sendEmailRequest{
		To: request.Email, Subject: "Confirm your email to view appointment details",
		HTML: "<p>Confirm your email to continue.</p><p><a href=\"" + template.HTMLEscapeString(link) + "\">Confirm email</a></p>",
	}, key)
	if err != nil {
		if deleteErr := s.users.DeleteUser(ctx, user.UserID); deleteErr != nil {
			return SignupResult{}, fmt.Errorf("send verification email: %w; delete user %s: %v", err, user.UserID, deleteErr)
		}
		return SignupResult{}, err
	}
	return SignupResult{UserID: user.UserID, MessageID: email.MessageID, State: "verification_sent"}, nil
}

func signupKey(email, appointmentReference string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email)) + "\n" + appointmentReference))
	return hex.EncodeToString(sum[:])
}
