package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/achmudas/identity-api/gen/profile/v1/profilev1connect"
	"github.com/achmudas/identity-api/internal/auth"
	"github.com/achmudas/identity-api/internal/config"
	"github.com/achmudas/identity-api/internal/httpapi"
	"github.com/achmudas/identity-api/internal/logger"
	"github.com/achmudas/identity-api/internal/user"
	"github.com/achmudas/identity-api/internal/user/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s/postgres?sslmode=disable&search_path=%s", cfg.DBConfig.DBUsername, cfg.DBConfig.DBPassword, cfg.DBConfig.DBUrl, cfg.DBConfig.DBSchema)
	m, err := migrate.New("file://db/migrations", connString)
	if err != nil {
		log.Fatalf("failed to initialize migration %v", err)
	}

	m.Log = &logger.MigrateLogger{}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("failed to run migration: %v", err)
	}

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("failed to initialize connection pool %v", err)
	}

	// #TODO put it under config
	client := profilev1connect.NewProfileServiceClient(http.DefaultClient, "http://localhost:8085")

	var repo user.Repo = store.NewPostgresRepo(pool)
	service := user.NewService(repo, client)

	handler := httpapi.NewHandler(service)

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
	}))

	var authenticator auth.Authenticator = auth.NewKeycloak(&cfg.KeycloakConfig)

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", handler.Healthz)
	r.Get("/auth/authenticate", authenticator.AuthenticateRedirect)
	r.Get("/auth/callback", authenticator.CallbackAuthenticate)

	r.Group(func(r chi.Router) {
		r.Use(authenticator.AuthClaims)
		r.Use(auth.Middleware)
		r.Get("/user/{email}", handler.FindUser)
		r.Post("/user", handler.CreateUser)
		r.Get("/", handler.Home)
	})

	srv := &http.Server{Addr: ":" + cfg.AppConfig.Port, Handler: r}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped")
}
