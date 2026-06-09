package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/achmudas/identity-api/internal/config"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type Keycloak struct {
	oauth2Conf    *oauth2.Config
	tokenVerifier *oidc.IDTokenVerifier
	provider      *oidc.Provider
}

func NewKeycloak(keycloakConf *config.KeycloakConfig) *Keycloak {
	prov, err := oidc.NewProvider(context.Background(), fmt.Sprintf("%s/realms/%s", keycloakConf.KeycloakURL, keycloakConf.KeycloakRealm))
	if err != nil {
		log.Fatalf("Failed to initialize provider: %v", err)
	}

	return &Keycloak{oauth2Conf: &oauth2.Config{
		ClientID:     keycloakConf.KeycloakClientID,
		ClientSecret: keycloakConf.KeycloakClientSecret,
		RedirectURL:  keycloakConf.KeycloakRedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		Endpoint:     prov.Endpoint()},
		provider:      prov,
		tokenVerifier: prov.Verifier(&oidc.Config{ClientID: keycloakConf.KeycloakClientID}),
	}
}

func (k *Keycloak) AuthenticateRedirect(w http.ResponseWriter, r *http.Request) {
	state := uuid.NewString()
	pkce := oauth2.GenerateVerifier()
	http.SetCookie(w, &http.Cookie{Name: "state", Value: state, SameSite: http.SameSiteLaxMode, HttpOnly: true})
	http.SetCookie(w, &http.Cookie{Name: "pkce", Value: pkce, SameSite: http.SameSiteLaxMode, HttpOnly: true})
	http.Redirect(w, r, k.oauth2Conf.AuthCodeURL(state, oauth2.S256ChallengeOption(pkce)), http.StatusFound)
}

func (k *Keycloak) CallbackAuthorize(w http.ResponseWriter, r *http.Request) {
	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token. The HTTP Client returned by
	// conf.Client will refresh the token as necessary.
	state, err := r.Cookie("state")
	if err != nil {
		log.Printf("failed to retrieve state from cookie: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if r.URL.Query().Get("state") != state.Value {
		log.Println("states do not match")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		log.Println("no code returned from Keycloak")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	pkceVerifier, err := r.Cookie("pkce")
	if err != nil {
		log.Printf("failed to retrieve pkce verifier from cookie: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	oauth2Token, err := k.oauth2Conf.Exchange(r.Context(), code, oauth2.VerifierOption(pkceVerifier.Value))
	if err != nil {
		log.Printf("error retrieving oauth2Token from Keycloak: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Printf("missing token")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	idToken, err := k.tokenVerifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		log.Printf("error when validating token: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var claims struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
		Subject  string `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		log.Printf("error when extracting claims from token: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "http://localhost:8080", http.StatusFound)

	// client := k.oauth2Conf.Client(r.Context(), tok)
	// // #TODO not sure what to do here
	// client.Get("http://localhost:8080/")
}
