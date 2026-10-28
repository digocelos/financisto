package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/digocelos/financisto/internal/accounts"
	"github.com/digocelos/financisto/internal/auth"
	"github.com/digocelos/financisto/internal/cards"
	"github.com/digocelos/financisto/internal/config"
	"github.com/digocelos/financisto/internal/dashboard"
	"github.com/digocelos/financisto/internal/db"
	"github.com/digocelos/financisto/internal/httpapi"
	"github.com/digocelos/financisto/internal/ledger"
	"github.com/digocelos/financisto/internal/goals"
	"github.com/digocelos/financisto/internal/planning"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create database pool: %v", err)
	}
	defer pool.Close()

	if err := db.ApplyMigrations(ctx, pool); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}
	authService, err := auth.New(ctx, pool, auth.Config{
		IssuerURL: cfg.OIDCIssuerURL, InternalIssuerURL: cfg.OIDCInternalIssuerURL, AuthorizationURL: cfg.OIDCAuthorizationURL, TokenURL: cfg.OIDCTokenURL, ClientID: cfg.OIDCClientID, ClientSecret: cfg.OIDCClientSecret,
		RedirectURL: cfg.OIDCRedirectURL, FrontendURL: cfg.FrontendURL, InvitationURLBase: cfg.InvitationURLBase,
		SMTPHost: cfg.SMTPHost, SMTPPort: cfg.SMTPPort, SMTPUsername: cfg.SMTPUsername, SMTPPassword: cfg.SMTPPassword, SMTPFrom: cfg.SMTPFrom,
		SecureCookies: cfg.SecureCookies,
	})
	if err != nil {
		log.Fatalf("initialize oidc: %v", err)
	}

	oidcIssuerForAPI := cfg.OIDCIssuerURL
	if cfg.OIDCInternalIssuerURL != "" {
		oidcIssuerForAPI = cfg.OIDCInternalIssuerURL
	}
	oidcDiscovery := strings.TrimRight(oidcIssuerForAPI, "/") + "/.well-known/openid-configuration"
	mux := http.NewServeMux()
	mux.Handle("/healthz", httpapi.NewCORSHandler(cfg.CORSAllowedOrigin, httpapi.NewHealthHandler(pool)))
	mux.Handle("/readyz", httpapi.NewCORSHandler(cfg.CORSAllowedOrigin, httpapi.NewReadinessHandler(httpapi.ReadinessDependencies{
		Database:      pool,
		OIDCDiscovery: oidcDiscovery,
		HTTPClient:    &http.Client{Timeout: 2 * time.Second},
	})))
	apiRoutes := http.NewServeMux()
	authService.MountRoutes(apiRoutes)
	accounts.NewHandler(pool, authService).MountRoutes(apiRoutes)
	cards.NewHandler(pool, authService).MountRoutes(apiRoutes)
	ledger.NewHandler(pool, authService).MountRoutes(apiRoutes)
	dashboard.NewHandler(pool, authService).MountRoutes(apiRoutes)
	planning.NewHandler(pool, authService).MountRoutes(apiRoutes)
	goals.NewHandler(pool, authService).MountRoutes(apiRoutes)
	mux.Handle("/auth/", httpapi.NewCORSHandler(cfg.CORSAllowedOrigin, apiRoutes))
	mux.Handle("/v1/", httpapi.NewCORSHandler(cfg.CORSAllowedOrigin, apiRoutes))

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("financisto API listening on %s (OIDC issuer %s)", cfg.HTTPAddr, cfg.OIDCIssuerURL)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
