package store

import (
	"context"
	"errors"

	"github.com/achmudas/identity-api/internal/user"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (p *PostgresRepo) Get(ctx context.Context, email string) (user.User, error) {
	if email == "" {
		return user.User{}, user.ErrNotFound
	}

	queries := user.New(p.pool)

	foundUser, err := queries.FindUser(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}

	if err != nil {
		return user.User{}, err
	}

	return foundUser, nil
}

func (p *PostgresRepo) Create(ctx context.Context, newUser user.User) error {
	queries := user.New(p.pool)

	_, err := queries.CreateUser(ctx, user.CreateUserParams{Username: newUser.Username, Password: newUser.Password, Email: newUser.Email})
	if err != nil {
		return err
	}

	return nil
}
