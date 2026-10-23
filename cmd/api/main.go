package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/digocelos/financisto/internal/config"
	"github.com/digocelos/financisto/internal/db"
	"github.com/digocelos/financisto/internal/httpapi"
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

	oidcDiscovery := strings.TrimRight(cfg.OIDCIssuerURL, "/") + "/.well-known/openid-configuration"
	mux := http.NewServeMux()
	mux.Handle("/healthz", httpapi.NewHealthHandler(pool))
	mux.Handle("/readyz", httpapi.NewReadinessHandler(httpapi.ReadinessDependencies{
		Database:      pool,
		OIDCDiscovery: oidcDiscovery,
		HTTPClient:    &http.Client{Timeout: 2 * time.Second},
	}))

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("financisto API listening on %s (OIDC issuer %s)", cfg.HTTPAddr, cfg.OIDCIssuerURL)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
