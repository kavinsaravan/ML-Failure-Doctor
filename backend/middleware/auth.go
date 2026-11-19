package middleware

import (
	"context"
	"crashlens/db"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

type identityKey struct{}
type Identity struct {
	Owner string
	Admin bool
}

func RequestIdentity(r *http.Request) Identity {
	if identity, ok := r.Context().Value(identityKey{}).(Identity); ok {
		return identity
	}
	return Identity{Owner: "legacy"}
}
func Authenticate(database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			expected := os.Getenv("CRASHLENS_API_KEY")
			identity := Identity{Owner: "legacy"}
			if expected != "" && subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1 {
				identity.Admin = true
			} else if token != "" {
				owner, err := database.ResolveAPIKey(token)
				if err != nil {
					http.Error(w, "Invalid API key", 401)
					return
				}
				identity.Owner = owner
			} else {
				http.Error(w, "API key required", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey{}, identity)))
		})
	}
}
