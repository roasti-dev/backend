package users

import (
	"fmt"
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrNotFound              = apierr.NewApiError(http.StatusNotFound, "user not found")
	ErrUsernameTaken         = apierr.NewApiError(http.StatusConflict, "username already taken")
	ErrUsernameTooShort      = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("username must be at least %d characters", usernameMinLength))
	ErrUsernameTooLong       = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("username must be at most %d characters", usernameMaxLength))
	ErrInvalidUsernameFormat = apierr.NewApiError(http.StatusUnprocessableEntity, "username can only contain letters, numbers and underscores")
)
