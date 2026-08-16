package middleware

import (
	"context"
	"net/http"
	"strings"

	"stickstock/backend/internal/i18n"
)

type ctxKey string

const localeCtxKey ctxKey = "locale"

// Locale resolves the request language from, in priority order:
// 1. ?lang= query param, 2. Accept-Language header, 3. default locale.
// It always injects one of i18n.SupportedLocales into the context.
func Locale(defaultLocale string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			locale := defaultLocale

			if q := r.URL.Query().Get("lang"); q != "" && i18n.IsSupported(q) {
				locale = q
			} else if h := r.Header.Get("Accept-Language"); h != "" {
				if primary := strings.Split(strings.Split(h, ",")[0], "-")[0]; i18n.IsSupported(primary) {
					locale = primary
				}
			}

			ctx := context.WithValue(r.Context(), localeCtxKey, locale)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func LocaleFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(localeCtxKey).(string); ok {
		return v
	}
	return "en"
}
