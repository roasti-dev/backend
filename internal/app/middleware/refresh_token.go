package middleware

import (
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
)

func RefreshToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("refresh_token")
		if err == nil && cookie.Value != "" {
			ctx := requestctx.WithRefreshToken(r.Context(), cookie.Value)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
