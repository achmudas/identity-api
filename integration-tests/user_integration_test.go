package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"

	"github.com/achmudas/identity-api/internal/logger"
	"github.com/achmudas/identity-api/internal/store"
	"github.com/achmudas/identity-api/internal/user"
	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func TestUserCreation(t *testing.T) {
	tests := []struct {
		name  string
		u     user.User
		email string
	}{
		{"user1 created", user.User{Username: "user1", Password: "pass", Email: "user1@email.com"}, "user1@email.com"},
		{"user2 created", user.User{Username: "user2", Password: "pass", Email: ""}, ""},
		{"user3 created", user.User{Username: "user3", Password: "pass", Email: "user3@email.com"}, "user3@email.com"},
	}

	ctx := context.Background()
	postgres, err := testcontainers.Run(
		ctx, "postgres:latest",
		testcontainers.WithExposedPorts("5432/tcp"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
		),
		testcontainers.WithEnv(map[string]string{"POSTGRES_PASSWORD": "pass"}),
	)

	service := createUserService(t, postgres)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = service.CreateUser(ctx, tt.u)
			if err != nil {
				t.Error(err)
			}
			usr, err := service.FindUser(ctx, tt.u.Email)

			if !errors.Is(err, user.ErrNotFound) && usr.Email != tt.email {
				t.Errorf("User create find failed, user created: %s; want %s", tt.u.Email, tt.email)
			}
		})
	}

	testcontainers.CleanupContainer(t, postgres)
	require.NoError(t, err)
}

func createUserService(t *testing.T, postgres *testcontainers.DockerContainer) *user.Service {
	host, err := postgres.Endpoint(context.Background(), "")
	if err != nil {
		t.Error(err)
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s?sslmode=disable", "postgres", "pass", host)
	m, err := migrate.New("file://../db/migrations", connString)
	if err != nil {
		log.Fatalf("failed to initialize migration %v", err)
	}

	m.Log = &logger.MigrateLogger{}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("failed to run migration: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatalf("failed to initialize connection pool %v", err)
	}

	var repo user.Repo = store.NewPostgresRepo(pool)
	return user.NewService(repo)
}
