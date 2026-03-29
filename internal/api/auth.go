package api

import (
	"net/http"
	"strings"

	"github.com/getflaggy/flaggy/internal/models"
)

// RequireMasterKey returns a middleware that requires the master key for admin routes.
// If masterKey is empty, auth is disabled (dev mode).
func RequireMasterKey(masterKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if masterKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			token := extractBearer(r)
			if token != masterKey {
				respondError(w, http.StatusUnauthorized, "invalid or missing master key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAPIKey returns a middleware that validates API keys for client routes.
// If masterKey is set and matches, it also passes (admin can do everything).
func RequireAPIKey(s apiKeyValidator, masterKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				respondError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			// Master key bypasses API key validation
			if masterKey != "" && token == masterKey {
				next.ServeHTTP(w, r)
				return
			}

			hashedKey := models.HashKey(token)
			apiKey, err := s.ValidateAPIKey(hashedKey)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "auth error")
				return
			}
			if apiKey == nil {
				respondError(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// apiKeyValidator is the subset of Store needed by the auth middleware.
type apiKeyValidator interface {
	ValidateAPIKey(hashedKey string) (*models.APIKey, error)
}

func extractBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return auth
}
