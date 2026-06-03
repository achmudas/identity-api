// just main package for testing initial setup
package main

import (
	"log"
	"net/http"

	"github.com/achmudas/identity-api/internal/config"
	"github.com/achmudas/identity-api/internal/httpapi"
	"github.com/achmudas/identity-api/internal/store"
	"github.com/achmudas/identity-api/internal/user"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	var repo user.Repo = store.NewPostgresRepo(cfg.DBUrl)
	service := user.NewService(repo)

	handler := httpapi.NewHandler(service)

	r := chi.NewRouter()
	r.Get("/healthz", handler.Healthz)
	r.Get("/user", handler.User)

	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}
