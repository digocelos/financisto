package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type healthyDependency struct{}

func (healthyDependency) Ping(context.Context) error { return nil }

func TestReadinessRequiresOIDCDiscovery(t *testing.T) {
	oidc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer oidc.Close()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()
	NewReadinessHandler(ReadinessDependencies{
		Database:      healthyDependency{},
		OIDCDiscovery: oidc.URL,
	}).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
}

func TestReadinessFailsWhenDatabaseIsUnavailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()
	NewReadinessHandler(ReadinessDependencies{}).ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}
