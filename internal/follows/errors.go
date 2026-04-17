package follows

import (
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrNotFound         = apierr.NewApiError(http.StatusNotFound, "user not found")
	ErrCannotFollowSelf = apierr.NewApiError(http.StatusBadRequest, "cannot follow yourself")
)
