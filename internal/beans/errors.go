package beans

import (
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrNotFound  = apierr.NewApiError(http.StatusNotFound, "bean not found")
	ErrForbidden = apierr.NewApiError(http.StatusForbidden, "forbidden")
)
