package users

import (
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrNotFound      = apierr.NewApiError(http.StatusNotFound, "user not found")
	ErrUsernameTaken = apierr.NewApiError(http.StatusConflict, "username already taken")
)
