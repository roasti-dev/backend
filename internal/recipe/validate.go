package recipe

import (
	"strings"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

func validateRecipePayload(req models.RecipePayload) error {
	if strings.TrimSpace(req.Title) == "" {
		return ErrInvalidTitle
	}

	if strings.TrimSpace(req.Description) == "" {
		return ErrInvalidDescription
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

	for _, step := range req.Steps {
		if strings.TrimSpace(step.Title) == "" {
			return ErrInvalidStepTitle
		}
		if step.Description != nil && strings.TrimSpace(*step.Description) == "" {
			return ErrInvalidStepDescription
		}
	}

	return nil
}
