package recipes

import (
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrForbidden              = apierr.NewApiError(http.StatusForbidden, "forbidden")
	ErrNotFound               = apierr.NewApiError(http.StatusNotFound, "recipe not found")
	ErrInvalidTitle           = apierr.NewApiError(http.StatusUnprocessableEntity, "title cannot be empty")
	ErrInvalidDescription     = apierr.NewApiError(http.StatusUnprocessableEntity, "description cannot be empty")
	ErrInvalidBrewMethod      = apierr.NewApiError(http.StatusUnprocessableEntity, "invalid brew method")
	ErrInvalidDifficulty      = apierr.NewApiError(http.StatusUnprocessableEntity, "invalid difficulty")
	ErrInvalidRoastLevel      = apierr.NewApiError(http.StatusUnprocessableEntity, "invalid roast level")
	ErrInvalidStepTitle       = apierr.NewApiError(http.StatusUnprocessableEntity, "step title cannot be empty")
	ErrInvalidStepDescription = apierr.NewApiError(http.StatusUnprocessableEntity, "step description cannot be empty")
)
