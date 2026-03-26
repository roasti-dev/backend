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
