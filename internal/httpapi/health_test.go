package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzReturnsOKWithoutDependencies(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	NewHealthHandler(nil).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Body.String(); got != "{\"status\":\"ok\"}\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestCORSAllowsConfiguredFrontendOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	res := httptest.NewRecorder()

	NewCORSHandler("http://localhost:5173", NewHealthHandler(nil)).ServeHTTP(res, req)

	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestCORSAllowsFamilyContextHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/v1/families/family-1/members", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Headers", "X-Family-ID")
	res := httptest.NewRecorder()

	NewCORSHandler("http://localhost:5173", NewHealthHandler(nil)).ServeHTTP(res, req)

	if got := res.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization, X-Family-ID" {
		t.Fatalf("allowed headers = %q", got)
	}
}
