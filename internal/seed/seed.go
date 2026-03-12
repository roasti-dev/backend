package seed

import (
	"context"
	"fmt"

	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
)

type Services struct {
	RecipeService *recipe.Service
}

func Run(ctx context.Context, s Services) error {
	if err := seedRecipes(ctx, s.RecipeService); err != nil {
		return fmt.Errorf("seed recipes: %w", err)
	}

	return nil
}
