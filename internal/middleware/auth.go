package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey struct{}

var userIDKey contextKey

func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Fields(r.Header.Get("Authorization"))
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeUnauthorized(w, "missing or invalid authorization header")
				return
			}

			claims := &jwt.RegisteredClaims{}

			token, err := jwt.ParseWithClaims(
				parts[1],
				claims,
				func(token *jwt.Token) (any, error) {
					return []byte(jwtSecret), nil
				},
				jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
				jwt.WithExpirationRequired(),
			)
			if err != nil || !token.Valid {
				writeUnauthorized(w, "invalid or expired token")
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				writeUnauthorized(w, "invalid token subject")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
