package user

import (
	"context"
	"errors"
	"log"

	profilev1 "github.com/achmudas/identity-api/gen/profile/v1"
	"github.com/achmudas/identity-api/gen/profile/v1/profilev1connect"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc/metadata"
)

type Service struct {
	repo          Repo
	profileClient profilev1connect.ProfileServiceClient
}

var ErrNotFound = errors.New("user not found")

func NewService(repo Repo, client profilev1connect.ProfileServiceClient) *Service {
	return &Service{repo: repo, profileClient: client}
}

func (s *Service) FindUser(ctx context.Context, email string) (UserDTO, error) {
	u, err := s.repo.Get(ctx, email)
	if err != nil {
		log.Printf("error: %v", err)
		return UserDTO{}, err
	}

	dto := UserDTO{ID: u.UserID, Email: u.Email, Username: u.Username, Password: u.Password}

	ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", middleware.GetReqID(ctx))
	resp, err := s.profileClient.GetProfileData(ctx, &profilev1.GetProfileDataRequest{UserId: u.UserID})
	if err != nil {
		log.Printf("error when retrieving profile information: %v", err)
		return dto, err
	}

	profile := resp.GetProfile()
	if profile != nil {
		dto.AvatarLink = profile.AvatarLink
		dto.Address = profile.Address
	}

	log.Printf("user found %v", u)
	return dto, nil
}

func (s *Service) CreateUser(ctx context.Context, user User) error {
	err := s.repo.Create(ctx, user)
	if err != nil {
		log.Printf("error: %v", err)
		return err
	}
	return nil
}
