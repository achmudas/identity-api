package httpapi

import (
	"errors"
	"net/http"

	"github.com/achmudas/identity-api/internal/user"
)

type Handler struct {
	userService *user.Service
}

func NewHandler(userService *user.Service) *Handler {
	return &Handler{userService}
}

func (h *Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) User(w http.ResponseWriter, r *http.Request) {
	_, err := h.userService.FindUser(r.Context(), "email@domain.com")
	if errors.Is(err, user.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
