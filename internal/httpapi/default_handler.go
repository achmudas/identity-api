package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	profilev1 "github.com/achmudas/identity-api/gen/profile/v1"
	"github.com/achmudas/identity-api/gen/profile/v1/profilev1connect"
	httpapierrors "github.com/achmudas/identity-api/internal/httpapi/errors"
	"github.com/achmudas/identity-api/internal/user"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc/metadata"
)

type Handler struct {
	userService   *user.Service
	profileClient profilev1connect.ProfileServiceClient
}

func NewHandler(userService *user.Service, client profilev1connect.ProfileServiceClient) *Handler {
	return &Handler{userService: userService, profileClient: client}
}

func (h *Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Home(w http.ResponseWriter, _ *http.Request) {
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

	dto := user.UserDTO{ID: u.UserID, Email: u.Email, Username: u.Username}

	ctx := metadata.AppendToOutgoingContext(r.Context(), "x-request-id", middleware.GetReqID(r.Context()))
	resp, err := h.profileClient.GetProfileData(ctx, &profilev1.GetProfileDataRequest{UserId: u.UserID})
	if err != nil {
		log.Printf("error when retrieving profile information: %v", err)
		// #TODO could be extended with the errors from grpc service
		respondError(w, http.StatusInternalServerError, httpapierrors.APIError{Code: "bad_error", Message: "Failed to retrieve profile information."})
		return
	}

	profile := resp.GetProfile()
	if profile != nil {
		dto.AvatarLink = profile.AvatarLink
		dto.Address = profile.Address
	}

	respondUser(w, http.StatusOK, dto)
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

func respondUser(w http.ResponseWriter, status int, user user.UserDTO) {
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
