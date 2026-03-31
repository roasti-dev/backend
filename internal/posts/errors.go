package posts

import (
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrNotFound  = apierr.NewApiError(http.StatusNotFound, "post not found")
	ErrForbidden = apierr.NewApiError(http.StatusForbidden, "not allowed")
)
