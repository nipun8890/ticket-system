package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"ticketsystem/internal/auth"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const UserEmailKey contextKey = "user_email"

// RequireAuth validates the Authorization: Bearer <token> header and injects
// the authenticated user's ID/email into the request context. Requests
// without a valid, unexpired JWT are rejected with 401.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			token, err := auth.ExtractBearerToken(header)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "malformed Authorization header, expected: Bearer <token>")
				return
			}

			claims, err := auth.ParseToken(secret, token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
