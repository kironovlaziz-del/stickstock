package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const userIDCtxKey ctxKey = "user_id"

// RequireAuth validates a Bearer JWT, rejects blocked users (see
// handlers/admin.go), and injects the authenticated user's ID into the
// request context. Routes that don't need auth (health check, public
// share links) should not be wrapped with this.
func RequireAuth(secret string, db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":"missing bearer token"}`, http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{"HS256"}))
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
				return
			}
			sub, _ := claims["sub"].(string)
			if sub == "" {
				http.Error(w, `{"error":"token missing subject"}`, http.StatusUnauthorized)
				return
			}

			// Checked on every request rather than once at login, since a
			// user blocked mid-session should lose access immediately, not
			// just stop being able to log in again.
			var isBlocked bool
			if err := db.QueryRowContext(r.Context(),
				`SELECT is_blocked FROM profiles WHERE id = $1`, sub,
			).Scan(&isBlocked); err == nil && isBlocked {
				http.Error(w, `{"error":"account blocked"}`, http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), userIDCtxKey, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDCtxKey).(string)
	return v, ok
}

// RequireAdmin further restricts an already-RequireAuth-wrapped route to
// callers whose profile has is_admin = true. Chain it after RequireAuth
// so UserIDFromContext is already populated: auth(admin(handler)).
func RequireAdmin(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			var isAdmin bool
			if err := db.QueryRowContext(r.Context(),
				`SELECT is_admin FROM profiles WHERE id = $1`, userID,
			).Scan(&isAdmin); err != nil || !isAdmin {
				http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
