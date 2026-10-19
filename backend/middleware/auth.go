package middleware

import (
	"net/http"
	"os"
	"strings"
)

// RequireAPIKey middleware checks for a valid API key on write/delete operations
func RequireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get API key from environment
		expectedKey := os.Getenv("CRASHLENS_API_KEY")

		// If no API key is configured, allow all requests (development mode)
		if expectedKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Support both "Bearer <token>" and direct API key
		var providedKey string
		if strings.HasPrefix(authHeader, "Bearer ") {
			providedKey = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			providedKey = authHeader
		}

		// Validate API key
		if providedKey != expectedKey {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
