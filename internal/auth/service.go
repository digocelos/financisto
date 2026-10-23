package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/digocelos/financisto/internal/httpapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

const (
	stateCookieName   = "financisto_oauth_state"
	sessionCookieName = "financisto_session"
	sessionDuration   = 24 * time.Hour
)

type Config struct {
	IssuerURL         string
	InternalIssuerURL string
	AuthorizationURL  string
	TokenURL          string
	ClientID          string
	ClientSecret      string
	RedirectURL       string
	FrontendURL       string
	SecureCookies     bool
}

type Service struct {
	db            *pgxpool.Pool
	oauth         *oauth2.Config
	verifier      *oidc.IDTokenVerifier
	httpClient    *http.Client
	frontendURL   string
	secureCookies bool
}

type claims struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Nonce         string `json:"nonce"`
}

type contextKey string

const userIDKey contextKey = "user_id"

func New(ctx context.Context, db *pgxpool.Pool, cfg Config) (*Service, error) {
	providerContext := ctx
	var providerHTTPClient *http.Client
	if cfg.InternalIssuerURL != "" && cfg.InternalIssuerURL != cfg.IssuerURL {
		providerHTTPClient = &http.Client{Transport: rewriteHostTransport{
			publicHost:   hostOf(cfg.IssuerURL),
			internalHost: hostOf(cfg.InternalIssuerURL),
			base:         http.DefaultTransport,
		}}
		providerContext = oidc.ClientContext(ctx, providerHTTPClient)
	}
	provider, err := oidc.NewProvider(providerContext, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}
	endpoint := provider.Endpoint()
	if cfg.AuthorizationURL != "" {
		endpoint.AuthURL = cfg.AuthorizationURL
	}
	if cfg.TokenURL != "" {
		endpoint.TokenURL = cfg.TokenURL
	}
	return &Service{
		db: db, httpClient: providerHTTPClient,
		oauth: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     endpoint,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier:      provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		frontendURL:   cfg.FrontendURL,
		secureCookies: cfg.SecureCookies,
	}, nil
}

func (s *Service) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/login", s.login)
	mux.HandleFunc("GET /auth/callback", s.callback)
	mux.HandleFunc("GET /auth/logout", s.logout)
	mux.Handle("GET /v1/me", s.RequireUser(http.HandlerFunc(s.me)))
	return mux
}

func (s *Service) login(w http.ResponseWriter, r *http.Request) {
	state, err := NewOAuthState(strings.TrimSpace(r.URL.Query().Get("family_name")))
	if err != nil {
		http.Error(w, "could not start login", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: stateCookieName, Value: state, Path: "/", MaxAge: 600,
		HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, s.oauth.AuthCodeURL(state), http.StatusFound)
}

func (s *Service) callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || !secureEqual(stateCookie.Value, r.URL.Query().Get("state")) {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}
	state, err := ParseOAuthState(stateCookie.Value)
	if err != nil {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	exchangeContext := r.Context()
	if s.httpClient != nil {
		exchangeContext = context.WithValue(exchangeContext, oauth2.HTTPClient, s.httpClient)
	}
	token, err := s.oauth.Exchange(exchangeContext, r.URL.Query().Get("code"), oauth2.SetAuthURLParam("nonce", state.Nonce))
	if err != nil {
		http.Error(w, "could not exchange authorization code", http.StatusUnauthorized)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		http.Error(w, "provider did not return an id token", http.StatusUnauthorized)
		return
	}
	idToken, err := s.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, "invalid id token", http.StatusUnauthorized)
		return
	}
	var identity claims
	if err := idToken.Claims(&identity); err != nil || identity.Subject == "" || identity.Email == "" || !identity.EmailVerified || !secureEqual(identity.Nonce, state.Nonce) {
		http.Error(w, "invalid identity claims", http.StatusUnauthorized)
		return
	}

	userID, err := s.upsertUserAndOnboard(r.Context(), identity, state.FamilyName)
	if err != nil {
		http.Error(w, "could not complete onboarding", http.StatusInternalServerError)
		return
	}
	sessionToken, err := randomToken(32)
	if err != nil {
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}
	if _, err := s.db.Exec(r.Context(), `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`, hashToken(sessionToken), userID, time.Now().UTC().Add(sessionDuration)); err != nil {
		http.Error(w, "could not persist session", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: sessionToken, Path: "/", MaxAge: int(sessionDuration.Seconds()), HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: stateCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, s.frontendURL, http.StatusFound)
}

func (s *Service) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_, _ = s.db.Exec(r.Context(), "DELETE FROM sessions WHERE token_hash = $1", hashToken(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, s.frontendURL, http.StatusFound)
}

func (s *Service) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			httpapi.WriteProblem(w, http.StatusUnauthorized, "authentication required")
			return
		}
		var userID uuid.UUID
		err = s.db.QueryRow(r.Context(), `SELECT user_id FROM sessions JOIN users ON users.id = sessions.user_id WHERE token_hash = $1 AND expires_at > NOW() AND users.blocked_at IS NULL`, hashToken(cookie.Value)).Scan(&userID)
		if err != nil {
			httpapi.WriteProblem(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, userID)))
	})
}

func (s *Service) me(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(uuid.UUID)
	var response meResponse
	if err := s.db.QueryRow(r.Context(), `SELECT id, email, name, COALESCE(avatar_url, '') FROM users WHERE id = $1`, userID).Scan(&response.User.ID, &response.User.Email, &response.User.Name, &response.User.AvatarURL); err != nil {
		httpapi.WriteProblem(w, http.StatusUnauthorized, "authentication required")
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT f.id, f.name FROM families f JOIN family_members m ON m.family_id = f.id WHERE m.user_id = $1 AND f.blocked_at IS NULL AND m.archived_at IS NULL ORDER BY f.created_at`, userID)
	if err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load families")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var family familySummary
		if err := rows.Scan(&family.ID, &family.Name); err != nil {
			httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load families")
			return
		}
		response.Families = append(response.Families, family)
	}
	if err := rows.Err(); err != nil {
		httpapi.WriteProblem(w, http.StatusInternalServerError, "could not load families")
		return
	}
	if len(response.Families) > 0 {
		response.ActiveFamilyID = response.Families[0].ID
	}
	if requested := r.Header.Get("X-Family-ID"); requested != "" {
		familyID, err := uuid.Parse(requested)
		if err != nil || !containsFamily(response.Families, familyID) {
			httpapi.WriteProblem(w, http.StatusForbidden, "family is not available to this user")
			return
		}
		response.ActiveFamilyID = familyID
	}
	httpapi.WriteJSON(w, http.StatusOK, response)
}

type meResponse struct {
	User           userSummary     `json:"user"`
	Families       []familySummary `json:"families"`
	ActiveFamilyID uuid.UUID       `json:"activeFamilyId"`
}

type userSummary struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatarUrl,omitempty"`
}

type familySummary struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (s *Service) upsertUserAndOnboard(ctx context.Context, identity claims, familyName string) (uuid.UUID, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	userID := uuid.New()
	if err := tx.QueryRow(ctx, `INSERT INTO users (id, provider_subject, email, name, avatar_url) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (provider_subject) DO UPDATE SET email = EXCLUDED.email, name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url, updated_at = NOW() RETURNING id`, userID, identity.Subject, identity.Email, identity.Name, nullableString(identity.Picture)).Scan(&userID); err != nil {
		return uuid.Nil, err
	}
	var hasFamily bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM family_members WHERE user_id = $1 AND archived_at IS NULL)`, userID).Scan(&hasFamily); err != nil {
		return uuid.Nil, err
	}
	if !hasFamily {
		familyID := uuid.New()
		memberID := uuid.New()
		if strings.TrimSpace(familyName) == "" {
			familyName = "Minha família"
		}
		if _, err := tx.Exec(ctx, `INSERT INTO families (id, name) VALUES ($1, $2)`, familyID, familyName); err != nil {
			return uuid.Nil, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO family_members (id, family_id, user_id, name, email) VALUES ($1, $2, $3, $4, $5)`, memberID, familyID, userID, identity.Name, identity.Email); err != nil {
			return uuid.Nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func hashToken(value string) []byte { hash := sha256.Sum256([]byte(value)); return hash[:] }

func secureEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func containsFamily(families []familySummary, id uuid.UUID) bool {
	for _, family := range families {
		if family.ID == id {
			return true
		}
	}
	return false
}

type rewriteHostTransport struct {
	publicHost   string
	internalHost string
	base         http.RoundTripper
}

func (t rewriteHostTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host != t.publicHost {
		return t.base.RoundTrip(request)
	}
	clone := request.Clone(request.Context())
	clone.URL.Host = t.internalHost
	clone.Host = t.publicHost
	return t.base.RoundTrip(clone)
}

func hostOf(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}
