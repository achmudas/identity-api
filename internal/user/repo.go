package user

import (
	"context"
)

type Repo interface {
	Get(ctx context.Context, email string) (User, error)
	Create(ctx context.Context, user User) error
}
