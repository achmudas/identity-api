package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	v1 "github.com/achmudas/identity-api/gen/profile/v1"
)

type MockRepo struct {
}

type MockProfileServiceClient struct {
}

func NewMockProfileServiceClient() *MockProfileServiceClient {
	return &MockProfileServiceClient{}
}

func NewMockRepo() *MockRepo {
	return &MockRepo{}
}

func (m *MockProfileServiceClient) GetProfileData(context.Context, *v1.GetProfileDataRequest) (*v1.GetProfileDataResponse, error) {
	return &v1.GetProfileDataResponse{Profile: &v1.Profile{}}, nil
}

func (m *MockRepo) Get(_ context.Context, email string) (User, error) {
	if email == "" {
		return User{}, ErrNotFound
	}

	if email == "someothererror@email.com" {
		return User{}, errors.New("Some other driver related error")
	}

	foundUser := User{UserID: 5, Username: "user", Password: "pass", Email: "user@email.com"}
	return foundUser, nil
}

func (m *MockRepo) Create(_ context.Context, newUser User) error {

	if newUser.Email == "error@email.com" {
		return errors.New("User wasn't created")
	}
	return nil

}

func TestUserOtherError(t *testing.T) {
	service := NewService(NewMockRepo(), NewMockProfileServiceClient())
	_, err := service.FindUser(context.Background(), "someothererror@email.com")

	if err == nil {
		t.Error("expected error, got nil")
	} else if !strings.Contains(err.Error(), "Some other driver related error") {
		t.Errorf(`service.FindUser %q, want match for %#q`, err.Error(), "Some other driver related error")
	}

}

func TestUserNotFoundBecauseEmailEmpty(t *testing.T) {
	service := NewService(NewMockRepo(), NewMockProfileServiceClient())
	_, err := service.FindUser(context.Background(), "")

	if err == nil {
		t.Error("expected error, got nil")
	} else if !strings.Contains(err.Error(), "user not found") {
		t.Errorf(`service.FindUser %q, want match for %#q`, err.Error(), "user not found")
	}
}
