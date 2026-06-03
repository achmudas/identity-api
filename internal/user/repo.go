package user

import (
	"context"
)

type Repo interface {
	Get(ctx context.Context, email string) (User, error)
}
