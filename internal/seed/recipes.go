package seed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/recipe"

	_ "embed"
)

func SeedRecipes(ctx context.Context, recipeService *recipe.Service, userID string, filePath string) error {
	recipes, err := recipeService.ListRecipes(ctx, userID, models.ListRecipesParams{
		Page:  new(models.PageParam(1)),
		Limit: new(models.LimitParam(1)),
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

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&seed); err != nil {
		return fmt.Errorf("decode recipes: %w", err)
	}

	for _, req := range seed.Req {
		if _, err := recipeService.CreateRecipe(ctx, userID, req); err != nil {
			return fmt.Errorf("create recipe: %w", err)
		}
	}

	return nil
}
