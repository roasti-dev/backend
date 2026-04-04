package comments

import (
	"fmt"
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrNotFound    = apierr.NewApiError(http.StatusNotFound, "comment not found")
	ErrForbidden   = apierr.NewApiError(http.StatusForbidden, "not allowed")
	ErrInvalidText = apierr.NewApiError(http.StatusUnprocessableEntity, "comment text cannot be empty")
	ErrTextTooLong = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("comment text must be at most %d characters", textMaxLen))
)
