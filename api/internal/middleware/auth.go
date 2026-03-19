package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/auth"
)

// TokenValidator is the interface the middleware needs for JWT validation.
// Defined here (consumer-side) per project conventions.
type TokenValidator interface {
	Validate(tokenString string) (*auth.Claims, error)
}

// JWTAuth returns middleware that reads the JWT from the "doit_token" cookie,
// validates it, and sets the user ID in the request context.
func JWTAuth(tv TokenValidator, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("doit_token")
			if err != nil {
				writeUnauthorized(w)
				return
			}

			claims, err := tv.Validate(cookie.Value)
			if err != nil {
				logger.Debug().Err(err).Msg("invalid auth token")
				writeUnauthorized(w)
				return
			}

			ctx := auth.WithUserID(r.Context(), claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	// Response write error is intentionally not handled — if the client
	// disconnected, there is nothing useful we can do.
	json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"}) //nolint:errcheck
}
