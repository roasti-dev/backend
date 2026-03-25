package auth

import (
	"fmt"
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrTokenRevoked          = apierr.NewApiError(http.StatusUnauthorized, "token revoked")
	ErrInvalidCredentials    = apierr.NewApiError(http.StatusUnauthorized, "invalid credentials")
	ErrInvalidRefreshToken   = apierr.NewApiError(http.StatusUnauthorized, "invalid or expired refresh token")
	ErrUsernameTaken         = apierr.NewApiError(http.StatusConflict, "username already taken")
	ErrEmailTaken            = apierr.NewApiError(http.StatusConflict, "email already taken")
	ErrInvalidEmail          = apierr.NewApiError(http.StatusUnprocessableEntity, "email cannot be empty")
	ErrUsernameRequired      = apierr.NewApiError(http.StatusUnprocessableEntity, "username is required")
	ErrPasswordRequired      = apierr.NewApiError(http.StatusUnprocessableEntity, "password is required")
	ErrPasswordTooShort      = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("password must be at least %d characters", minPasswordLen))
	ErrPasswordTooLong       = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("password must be at most %d characters", maxPasswordLen))
	ErrUsernameTooShort      = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("username must be at least %d characters", minUsernameLen))
	ErrUsernameTooLong       = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("username must be at most %d characters", maxUsernameLen))
	ErrInvalidUsernameFormat = apierr.NewApiError(http.StatusUnprocessableEntity, "username can only contain letters, numbers and underscores")

	ErrIncorrectPassword   = apierr.NewApiError(http.StatusUnauthorized, "current password is incorrect")
	ErrUserDisabled        = apierr.NewApiError(http.StatusForbidden, "user account is disabled")
	ErrUserNotFound        = apierr.NewApiError(http.StatusNotFound, "user not found")
	ErrMissingRefreshToken = apierr.NewApiError(http.StatusBadRequest, "missing refresh token")
	ErrInvalidGrantType    = apierr.NewApiError(http.StatusBadRequest, "invalid grant type")
)
