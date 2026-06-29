package user

import (
	"context"
	"errors"
	"log"
)

type Service struct {
	repo Repo
}

var ErrNotFound = errors.New("user not found")

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) FindUser(ctx context.Context, email string) (User, error) {
	u, err := s.repo.Get(ctx, email)
	if err != nil {
		log.Printf("error: %v", err)
		return User{}, err
	}

	log.Printf("user found %v", u)
	return u, nil
}

func (s *Service) CreateUser(ctx context.Context, user User) error {
	err := s.repo.Create(ctx, user)
	if err != nil {
		log.Printf("error: %v", err)
		return err
	}
	return nil
}
