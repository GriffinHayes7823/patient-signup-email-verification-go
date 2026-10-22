package signup

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeUsers struct {
	request       createUserRequest
	deletedUserID string
}

func (f *fakeUsers) CreateUser(_ context.Context, request createUserRequest) (createUserResponse, error) {
	f.request = request
	return createUserResponse{UserID: "patient-42"}, nil
}

func (f *fakeUsers) DeleteUser(_ context.Context, userID string) error {
	f.deletedUserID = userID
	return nil
}

type fakeEmails struct {
	request sendEmailRequest
	key     string
	err     error
}

func (f *fakeEmails) SendEmail(_ context.Context, request sendEmailRequest, key string) (sendEmailResponse, error) {
	f.request, f.key = request, key
	return sendEmailResponse{MessageID: "message-7"}, f.err
}

func TestStartDeletesUserWhenVerificationEmailFails(t *testing.T) {
	users := &fakeUsers{}
	emails := &fakeEmails{err: errors.New("delivery failed")}
	service := NewVerificationService(users, emails, "https://portal.example.test")

	_, err := service.Start(context.Background(), SignupRequest{
		Email: "patient@example.test", Password: "test-password", Name: "Avery",
	})
	if err == nil || err.Error() != "delivery failed" {
		t.Fatalf("unexpected error: %v", err)
	}
	if users.deletedUserID != "patient-42" {
		t.Fatalf("created user was not deleted: %q", users.deletedUserID)
	}
}

func TestStartKeepsAppointmentDetailOutOfVerificationEmail(t *testing.T) {
	tests := []struct {
		name      string
		reference string
	}{
		{name: "scheduled visit", reference: "CARDIOLOGY-2026-09-15"},
		{name: "empty reference", reference: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, emails := &fakeUsers{}, &fakeEmails{}
			service := NewVerificationService(users, emails, "https://portal.example.test")
			result, err := service.Start(context.Background(), SignupRequest{
				Email: "patient@example.test", Password: "test-password", Name: "Avery", AppointmentReference: tt.reference,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.State != "verification_sent" || result.MessageID != "message-7" {
				t.Fatalf("unexpected result: %#v", result)
			}
			if tt.reference != "" && (strings.Contains(emails.request.HTML, tt.reference) || strings.Contains(emails.request.Subject, tt.reference)) {
				t.Fatal("verification notification exposed appointment detail")
			}
			if !strings.Contains(emails.request.HTML, "user_id=patient-42") {
				t.Fatal("verification link did not target the created user")
			}
			if users.request.IdempotencyKey != emails.key {
				t.Fatal("account creation and email did not share a stable operation key")
			}
		})
	}
}
