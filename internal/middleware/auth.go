package middleware

import (
	"context"
	"fmt"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/nikpivkin/roasti-app-backend/internal/requestctx"
)

func Authenticate(firebaseAuth *auth.Client) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, ai *openapi3filter.AuthenticationInput) error {
		if ai.SecuritySchemeName != "BearerAuth" {
			return nil
		}

		r := ai.RequestValidationInput.Request
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			return fmt.Errorf("missing Authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return fmt.Errorf("invalid Authorization header")
		}

		token, err := firebaseAuth.VerifyIDToken(r.Context(), parts[1])
		if err != nil {
			return fmt.Errorf("invalid token: %w", err)
		}

		*r = *r.WithContext(requestctx.WithUserID(r.Context(), token.UID))
		return nil
	}
}
