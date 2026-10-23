package config

import (
	"fmt"
	"os"
)

// Config contains runtime settings shared by local and production environments.
// OIDC is provider-agnostic: local development points at mock-oauth2-server and
// production points at Google's issuer through environment variables.
type Config struct {
	HTTPAddr         string
	DatabaseURL      string
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:         valueOrDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		OIDCIssuerURL:    os.Getenv("OIDC_ISSUER_URL"),
		OIDCClientID:     os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
	}

	for name, value := range map[string]string{
		"DATABASE_URL":       cfg.DatabaseURL,
		"OIDC_ISSUER_URL":    cfg.OIDCIssuerURL,
		"OIDC_CLIENT_ID":     cfg.OIDCClientID,
		"OIDC_CLIENT_SECRET": cfg.OIDCClientSecret,
		"OIDC_REDIRECT_URL":  cfg.OIDCRedirectURL,
	} {
		if value == "" {
			return Config{}, fmt.Errorf("required environment variable %s is not set", name)
		}
	}

	return cfg, nil
}

func valueOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
