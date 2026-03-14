package uploads

import (
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrNoFile              = apierr.NewApiError(http.StatusBadRequest, "file not found in request")
	ErrFileTooLarge        = apierr.NewApiError(http.StatusRequestEntityTooLarge, "file too large")
	ErrUnsupportedMIMEType = apierr.NewApiError(http.StatusUnsupportedMediaType, "unsupported MIME type")
	ErrNotFound            = apierr.NewApiError(http.StatusNotFound, "image not found")
)
