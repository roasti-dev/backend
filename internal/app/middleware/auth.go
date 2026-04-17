package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
)

type tokenRetriever func(r *http.Request) string

func tokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

func tokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("access_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// OptionalAuth sets the user ID in the context when a valid token is present,
// without rejecting unauthenticated requests. Skips if user ID is already set.
func OptionalAuth(firebaseAuth *auth.Client) func(http.Handler) http.Handler {
	retrievers := []tokenRetriever{tokenFromHeader, tokenFromCookie}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requestctx.GetUserID(r.Context()) == "" {
				for _, retrieve := range retrievers {
					if token := retrieve(r); token != "" {
						if t, err := firebaseAuth.VerifyIDToken(r.Context(), token); err == nil {
							r = r.WithContext(requestctx.WithUserID(r.Context(), t.UID))
						}
						break
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Authenticate(firebaseAuth *auth.Client) openapi3filter.AuthenticationFunc {
	retrievers := []tokenRetriever{tokenFromHeader, tokenFromCookie}
	return func(ctx context.Context, ai *openapi3filter.AuthenticationInput) error {
		switch ai.SecuritySchemeName {
		case "BearerAuth", "AccessTokenCookie":
		default:
			return nil
		}

		r := ai.RequestValidationInput.Request
		token := ""
		for _, retrieve := range retrievers {
			if t := retrieve(r); t != "" {
				token = t
				break
			}
		}

		t, err := firebaseAuth.VerifyIDToken(r.Context(), token)
		if err != nil {
			return fmt.Errorf("invalid token: %w", err)
		}

		*r = *r.WithContext(requestctx.WithUserID(r.Context(), t.UID))
		return nil
	}
}
