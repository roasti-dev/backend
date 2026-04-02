package recipes

import (
	"fmt"
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrForbidden                 = apierr.NewApiError(http.StatusForbidden, "forbidden")
	ErrNotFound                  = apierr.NewApiError(http.StatusNotFound, "recipe not found")
	ErrInvalidTitle              = apierr.NewApiError(http.StatusUnprocessableEntity, "title cannot be empty")
	ErrTitleTooLong              = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("title must be at most %d characters", titleMaxLen))
	ErrInvalidDescription        = apierr.NewApiError(http.StatusUnprocessableEntity, "description cannot be empty")
	ErrDescriptionTooLong        = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("description must be at most %d characters", descriptionMaxLen))
	ErrInvalidBrewMethod         = apierr.NewApiError(http.StatusUnprocessableEntity, "invalid brew method")
	ErrInvalidDifficulty         = apierr.NewApiError(http.StatusUnprocessableEntity, "invalid difficulty")
	ErrInvalidRoastLevel         = apierr.NewApiError(http.StatusUnprocessableEntity, "invalid roast level")
	ErrTooManySteps              = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("recipe must have at most %d steps", stepsMaxCount))
	ErrInvalidStepTitle          = apierr.NewApiError(http.StatusUnprocessableEntity, "step title cannot be empty")
	ErrStepTitleTooLong          = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("step title must be at most %d characters", stepTitleMaxLen))
	ErrInvalidStepDescription    = apierr.NewApiError(http.StatusUnprocessableEntity, "step description cannot be empty")
	ErrStepDescriptionTooLong    = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("step description must be at most %d characters", stepDescriptionMaxLen))
)
