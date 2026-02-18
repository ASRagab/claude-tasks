package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

// Auth middleware enforces Bearer token auth when CLAUDE_TASKS_AUTH_TOKEN is set.
func Auth(next http.Handler) http.Handler {
	token := strings.TrimSpace(os.Getenv("CLAUDE_TASKS_AUTH_TOKEN"))
	if token == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer "+token {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(ErrorResponse{Error: "Unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// CORS middleware allows cross-origin requests from mobile apps.
// If CLAUDE_TASKS_CORS_ORIGIN is set, only that origin is allowed.
func CORS(next http.Handler) http.Handler {
	allowedOrigin := strings.TrimSpace(os.Getenv("CLAUDE_TASKS_CORS_ORIGIN"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Vary", "Origin")
		} else if origin != "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			if allowedOrigin != "" && origin != "" && origin != allowedOrigin {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
