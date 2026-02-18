package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ASRagab/claude-tasks/internal/testutil"
)

func TestAuthMiddlewareRejectsUnauthorizedRequests(t *testing.T) {
	t.Setenv("CLAUDE_TASKS_AUTH_TOKEN", "topsecret")
	srv := newTestServer(t)

	rr := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/v1/tasks", nil)
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}

	resp := testutil.DecodeJSON[ErrorResponse](t, rr)
	if resp.Error == "" {
		t.Fatalf("expected auth error message")
	}
}

func TestAuthMiddlewareAllowsAuthorizedRequests(t *testing.T) {
	t.Setenv("CLAUDE_TASKS_AUTH_TOKEN", "topsecret")
	srv := newTestServer(t)

	rr := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/v1/tasks", nil)
	req.Header.Set("Authorization", "Bearer topsecret")
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("CLAUDE_TASKS_CORS_ORIGIN", "https://app.example.com")
	srv := newTestServer(t)

	rr := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/v1/tasks", nil)
	req.Header.Set("Origin", "https://app.example.com")
	srv.Router().ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("expected allowed origin header to match configured origin, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddlewareRejectsMismatchedPreflightOrigin(t *testing.T) {
	t.Setenv("CLAUDE_TASKS_CORS_ORIGIN", "https://app.example.com")
	srv := newTestServer(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/tasks", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestCORSMiddlewareRejectsMismatchedOriginOnSimpleRequest(t *testing.T) {
	t.Setenv("CLAUDE_TASKS_CORS_ORIGIN", "https://app.example.com")
	srv := newTestServer(t)

	rr := httptest.NewRecorder()
	req := testutil.JSONRequest(t, http.MethodGet, "/api/v1/tasks", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	srv.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}


