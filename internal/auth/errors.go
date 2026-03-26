package auth

import (
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrTokenRevoked        = apierr.NewApiError(http.StatusUnauthorized, "token revoked")
	ErrInvalidCredentials  = apierr.NewApiError(http.StatusUnauthorized, "invalid credentials")
	ErrInvalidRefreshToken = apierr.NewApiError(http.StatusUnauthorized, "invalid or expired refresh token")
	ErrUsernameTaken       = apierr.NewApiError(http.StatusConflict, "username already taken")
	ErrEmailTaken          = apierr.NewApiError(http.StatusConflict, "email already taken")
	ErrInvalidEmail        = apierr.NewApiError(http.StatusUnprocessableEntity, "email cannot be empty")
	ErrUsernameRequired    = apierr.NewApiError(http.StatusUnprocessableEntity, "username is required")
	ErrPasswordRequired    = apierr.NewApiError(http.StatusUnprocessableEntity, "password is required")

	ErrIncorrectPassword   = apierr.NewApiError(http.StatusUnauthorized, "current password is incorrect")
	ErrUserDisabled        = apierr.NewApiError(http.StatusForbidden, "user account is disabled")
	ErrUserNotFound        = apierr.NewApiError(http.StatusNotFound, "user not found")
	ErrMissingRefreshToken = apierr.NewApiError(http.StatusBadRequest, "missing refresh token")
	ErrInvalidGrantType    = apierr.NewApiError(http.StatusBadRequest, "invalid grant type")
)
