package recipes

import "github.com/nikpivkin/roasti-app-backend/internal/api/models"

const (
	titleMaxLen           = 100
	descriptionMaxLen     = 2000
	stepTitleMaxLen       = 100
	stepDescriptionMaxLen = 1000
	stepsMaxCount         = 30
)

func validateRecipePayload(req models.RecipePayload) error {
	if req.Title == "" {
		return ErrInvalidTitle
	}
	if len(req.Title) > titleMaxLen {
		return ErrTitleTooLong
	}

	if req.Description == "" {
		return ErrInvalidDescription
	}
	if len(req.Description) > descriptionMaxLen {
		return ErrDescriptionTooLong
	}

	if !req.BrewMethod.Valid() {
		return ErrInvalidBrewMethod
	}

	if !req.Difficulty.Valid() {
		return ErrInvalidDifficulty
	}

	if req.RoastLevel != nil && !req.RoastLevel.Valid() {
		return ErrInvalidRoastLevel
	}

	if len(req.Steps) > stepsMaxCount {
		return ErrTooManySteps
	}

	for _, step := range req.Steps {
		if step.Title == "" {
			return ErrInvalidStepTitle
		}
		if len(step.Title) > stepTitleMaxLen {
			return ErrStepTitleTooLong
		}
		if step.Description != nil && *step.Description == "" {
			return ErrInvalidStepDescription
		}
		if step.Description != nil && len(*step.Description) > stepDescriptionMaxLen {
			return ErrStepDescriptionTooLong
		}
	}

	return nil
}
