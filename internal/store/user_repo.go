package store

import (
	"context"

	"github.com/achmudas/identity-api/internal/user"
)

type PostgresRepo struct {
	db string
}

func NewPostgresRepo(db string) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (_ *PostgresRepo) Get(ctx context.Context, email string) (user.User, error) {
	if email == "" {
		return user.User{}, user.ErrNotFound
	}
	return user.User{Email: email}, nil
}
