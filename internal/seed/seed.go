package seed

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
)

type Services struct {
	RecipeService *recipe.Service
}

func Run(ctx context.Context, s Services) error {
	if err := seedRecipes(ctx, s.RecipeService); err != nil {
		return err
	}

	return nil
}
