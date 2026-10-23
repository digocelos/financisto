package config

import "testing"

func TestLoadReadsOIDCSettingsFromEnvironment(t *testing.T) {
	t.Setenv("OIDC_ISSUER_URL", "http://mock-oauth2-server:8080/default")
	t.Setenv("OIDC_CLIENT_ID", "financisto-local")
	t.Setenv("OIDC_CLIENT_SECRET", "local-secret")
	t.Setenv("OIDC_REDIRECT_URL", "http://localhost:5173/auth/callback")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/financisto")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.OIDCIssuerURL != "http://mock-oauth2-server:8080/default" {
		t.Errorf("OIDCIssuerURL = %q", cfg.OIDCIssuerURL)
	}
	if cfg.OIDCClientID != "financisto-local" {
		t.Errorf("OIDCClientID = %q", cfg.OIDCClientID)
	}
	if cfg.OIDCClientSecret != "local-secret" {
		t.Errorf("OIDCClientSecret = %q", cfg.OIDCClientSecret)
	}
	if cfg.OIDCRedirectURL != "http://localhost:5173/auth/callback" {
		t.Errorf("OIDCRedirectURL = %q", cfg.OIDCRedirectURL)
	}
	if cfg.DatabaseURL == "" {
		t.Error("DatabaseURL should be populated")
	}
}

func TestLoadRejectsMissingRequiredSettings(t *testing.T) {
	for _, key := range []string{"OIDC_ISSUER_URL", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET", "OIDC_REDIRECT_URL", "DATABASE_URL"} {
		t.Run(key, func(t *testing.T) {
			for _, env := range []string{"OIDC_ISSUER_URL", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET", "OIDC_REDIRECT_URL", "DATABASE_URL"} {
				t.Setenv(env, "configured")
			}
			t.Setenv(key, "")

			if _, err := Load(); err == nil {
				t.Errorf("Load() error = nil, want missing %s", key)
			}
		})
	}
}
