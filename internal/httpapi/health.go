package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Dependency interface {
	Ping(context.Context) error
}

type ReadinessDependencies struct {
	Database      Dependency
	OIDCDiscovery string
	HTTPClient    *http.Client
}

func NewCORSHandler(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewHealthHandler(_ Dependency) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
}

func NewReadinessHandler(deps ReadinessDependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if deps.Database == nil || deps.Database.Ping(ctx) != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "dependency": "database"})
			return
		}

		client := deps.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, deps.OIDCDiscovery, nil)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "dependency": "oidc"})
			return
		}
		res, err := client.Do(req)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "dependency": "oidc"})
			return
		}
		res.Body.Close()
		if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "dependency": "oidc"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
