package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const userIDCtxKey ctxKey = "user_id"

func RequireAuth(secret, supabaseURL string, db *sql.DB) func(http.Handler) http.Handler {
	jwks := newJWKSCache(supabaseURL)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""
			if c, err := r.Cookie("ss_token"); err == nil {
				tokenStr = c.Value
			} else if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
				tokenStr = strings.TrimPrefix(h, "Bearer ")
			}
			if tokenStr == "" {
				http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				switch t.Method.Alg() {
				case "HS256":
					return []byte(secret), nil
				case "ES256":
					kid, _ := t.Header["kid"].(string)
					if kid == "" {
						return nil, fmt.Errorf("ES256 token missing kid header")
					}
					return jwks.publicKey(kid)
				default:
					return nil, fmt.Errorf("unsupported signing algorithm %q", t.Method.Alg())
				}
			}, jwt.WithValidMethods([]string{"HS256", "ES256"}))
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

			var isBlocked bool
			err = db.QueryRowContext(r.Context(),
				`SELECT is_blocked FROM profiles WHERE id = $1`, sub,
			).Scan(&isBlocked)
			if err != nil {
				http.Error(w, `{"error":"account not found"}`, http.StatusForbidden)
				return
			}
			if isBlocked {
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
