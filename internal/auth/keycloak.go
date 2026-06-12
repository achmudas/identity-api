package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/achmudas/identity-api/internal/config"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type Authenticator interface {
	AuthenticateRedirect(w http.ResponseWriter, r *http.Request)
	CallbackAuthenticate(w http.ResponseWriter, r *http.Request)
	AuthClaims(next http.Handler) http.Handler
}

type cachedClaims struct {
	roles    []string
	cachedAt time.Time
}

type Keycloak struct {
	oauth2Conf    *oauth2.Config
	tokenVerifier *oidc.IDTokenVerifier
	provider      *oidc.Provider
	sessions      map[string]*oauth2.Token
	mu            sync.Mutex
	claimsCache   map[string]cachedClaims
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
		sessions:      make(map[string]*oauth2.Token),
		mu:            sync.Mutex{},
		claimsCache:   make(map[string]cachedClaims),
	}
}

func (k *Keycloak) AuthenticateRedirect(w http.ResponseWriter, r *http.Request) {
	state := uuid.NewString()
	pkce := oauth2.GenerateVerifier()
	http.SetCookie(w, &http.Cookie{Name: "state", Value: state, SameSite: http.SameSiteLaxMode, HttpOnly: true})
	http.SetCookie(w, &http.Cookie{Name: "pkce", Value: pkce, SameSite: http.SameSiteLaxMode, HttpOnly: true})
	http.Redirect(w, r, k.oauth2Conf.AuthCodeURL(state, oauth2.S256ChallengeOption(pkce)), http.StatusFound)
}

func (k *Keycloak) CallbackAuthenticate(w http.ResponseWriter, r *http.Request) {
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

	sessionID := uuid.NewString()
	k.mu.Lock()
	k.sessions[sessionID] = oauth2Token
	k.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: sessionID, HttpOnly: true, SameSite: http.SameSiteLaxMode, Path: "/"})
	http.Redirect(w, r, "http://localhost:8080", http.StatusFound)
}

func (k *Keycloak) AuthClaims(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			log.Printf("Failed to retrieve session id %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		sessionID := cookie.Value

		k.mu.Lock()
		token, ok := k.sessions[sessionID]
		k.mu.Unlock()

		if !ok {
			log.Printf("No session ID in cookies")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		k.mu.Lock()
		cached, ok := k.claimsCache[sessionID]
		k.mu.Unlock()

		if ok && time.Since(cached.cachedAt) < 2*time.Minute {
			ctx := context.WithValue(r.Context(), RolesKey, cached.roles)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		tokenSource := k.oauth2Conf.TokenSource(r.Context(), token)
		_, err = k.provider.UserInfo(r.Context(), tokenSource)
		if err != nil {
			log.Printf("Failed to verify token %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		accessVerifier := k.provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
		accessToken, err := accessVerifier.Verify(r.Context(), token.AccessToken)

		if err != nil {
			log.Printf("Failed to verify token %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var claims struct {
			RealmAccess struct {
				Roles []string `json:"roles"`
			} `json:"realm_access"`
			ResourceAccess map[string]struct {
				Roles []string `json:"roles"`
			} `json:"resource_access"`
			Email string `json:"email"`
		}
		if err := accessToken.Claims(&claims); err != nil {
			log.Printf("Failed to retrieve claims from access token %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		k.claimsCache[sessionID] = cachedClaims{roles: claims.ResourceAccess["bestclient"].Roles, cachedAt: time.Now()}
		ctx := context.WithValue(r.Context(), RolesKey, claims.ResourceAccess["bestclient"].Roles)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
