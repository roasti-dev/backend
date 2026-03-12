package seed

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/pagination"
	"github.com/nikpivkin/roasti-app-backend/internal/recipe"

	_ "embed"
)

const seedUserID = "test-user"

//go:embed recipes.json
var recipesData []byte

func seedRecipes(ctx context.Context, recipeService *recipe.Service) error {
	recipes, err := recipeService.ListRecipes(ctx, seedUserID, recipe.ListRecipesParams{
		Pagination: pagination.New(1, 1),
	})
	if err != nil {
		return err
	}

	if recipes.TotalCount > 0 {
		return nil
	}

	var seed struct {
		Recipes []models.Recipe `yaml:"recipes"`
	}

	dec := json.NewDecoder(bytes.NewReader(recipesData))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&seed); err != nil {
		return err
	}

	for _, recipe := range seed.Recipes {
		if _, err := recipeService.CreateRecipe(ctx, seedUserID, recipe); err != nil {
			return err
		}
	}

	return nil
}
