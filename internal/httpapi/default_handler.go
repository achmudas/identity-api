package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	httpapierrors "github.com/achmudas/identity-api/internal/httpapi/errors"
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

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Hello world"))
	if err != nil {
		log.Printf("Failed to write bytes to response: %v", err)
	}

}

func (h *Handler) FindUser(w http.ResponseWriter, r *http.Request) {
	u, err := h.userService.FindUser(r.Context(), r.PathValue("email"))
	if errors.Is(err, user.ErrNotFound) {
		respondError(w, http.StatusNotFound, httpapierrors.APIError{Code: "not_found", Message: "User not found."})
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, httpapierrors.APIError{Code: "bad_error", Message: "Something bad happened when searching for user."})
		return
	}
	respondUser(w, http.StatusOK, u)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	u := &user.User{}
	err := dec.Decode(u)
	if err != nil {
		respondError(w, http.StatusInternalServerError, httpapierrors.APIError{Code: "bad_error", Message: "Something bad happened when creating new user."})
		return
	}

	err = h.userService.CreateUser(r.Context(), *u)
	if err != nil {
		respondError(w, http.StatusInternalServerError, httpapierrors.APIError{Code: "bad_error", Message: "Something bad happened when creating for user."})
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func respondError(w http.ResponseWriter, status int, apiError httpapierrors.APIError) {
	v, err := json.Marshal(httpapierrors.ErrorResponse{Error: apiError})
	if err != nil {
		log.Printf("error when marshaling response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respond(w, status, v)
}

func respondUser(w http.ResponseWriter, status int, user user.User) {
	v, err := json.Marshal(user)
	if err != nil {
		log.Printf("error when marshaling response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respond(w, status, v)
}

func respond(w http.ResponseWriter, status int, v []byte) {
	w.WriteHeader(status)
	_, err := w.Write(v)
	if err != nil {
		log.Printf("error when writing response: %v", err)
		return
	}

}
