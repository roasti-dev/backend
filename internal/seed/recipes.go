package seed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/recipe"

	_ "embed"
)

const seedUserID = "test-user"

//go:embed recipes.json
var recipesData []byte

func seedRecipes(ctx context.Context, recipeService *recipe.Service) error {
	recipes, err := recipeService.ListRecipes(ctx, seedUserID, models.ListRecipesParams{
		Page:  new(int32(1)),
		Limit: new(int32(1)),
	})
	if err != nil {
		return fmt.Errorf("list recipes: %w", err)
	}

	if recipes.Pagination.ItemsCount > 0 {
		return nil
	}

	var seed struct {
		Req []models.CreateRecipeRequest `json:"recipes"`
	}

	dec := json.NewDecoder(bytes.NewReader(recipesData))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&seed); err != nil {
		return fmt.Errorf("decode recipes: %w", err)
	}

	for _, req := range seed.Req {
		if _, err := recipeService.CreateRecipe(ctx, seedUserID, req); err != nil {
			return fmt.Errorf("create recipe: %w", err)
		}
	}

	return nil
}
