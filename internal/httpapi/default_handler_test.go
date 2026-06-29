package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	v1 "github.com/achmudas/identity-api/gen/profile/v1"
	"github.com/achmudas/identity-api/internal/user"
)

type MockRepo struct {
}

func NewMockRepo() *MockRepo {
	return &MockRepo{}
}

type MockProfileServiceClient struct {
}

func NewMockProfileServiceClient() *MockProfileServiceClient {
	return &MockProfileServiceClient{}
}

func (m *MockProfileServiceClient) GetProfileData(context.Context, *v1.GetProfileDataRequest) (*v1.GetProfileDataResponse, error) {
	return &v1.GetProfileDataResponse{Profile: &v1.Profile{}}, nil
}

func (m *MockRepo) Get(_ context.Context, email string) (user.User, error) {
	if email == "" {
		return user.User{}, user.ErrNotFound
	}

	if email == "someothererror@email.com" {
		return user.User{}, errors.New("Some other driver related error")
	}

	foundUser := user.User{UserID: 5, Username: "user", Password: "pass", Email: "user@email.com"}
	return foundUser, nil
}

func (m *MockRepo) Create(_ context.Context, newUser user.User) error {

	if newUser.Email == "error@email.com" {
		return errors.New("User wasn't created")
	}
	return nil

}

func TestUserFinding(t *testing.T) {
	req := httptest.NewRequest("GET", "/user/user@email.com", nil)
	req.SetPathValue("email", "user@email.com")

	w := httptest.NewRecorder()

	handler := NewHandler(user.NewService(NewMockRepo()), NewMockProfileServiceClient())

	handler.FindUser(w, req)

	res := w.Result()

	defer res.Body.Close()

	dec := json.NewDecoder(res.Body)
	u := &user.UserDTO{}
	err := dec.Decode(u)

	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if u.ID != 5 {
		t.Errorf("expected to receive user with ID 5 got %d", u.ID)
	}

}
