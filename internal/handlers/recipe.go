package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/requestctx"
)

func (s *ServerHandler) GetApiV1Recipes(ctx context.Context, request GetApiV1RecipesRequestObject) (GetApiV1RecipesResponseObject, error) {
	s.logger.DebugContext(ctx, "list recipes request")

	userID := requestctx.GetUserID(ctx)
	recipes, err := s.recipeService.ListRecipes(ctx, userID, *request.Params.ListRecipes)
	if err != nil {
		return GetApiV1Recipes200JSONResponse{}, err
	}

	s.logger.DebugContext(ctx, "recipes returned",
		"count", len(recipes.Items),
	)
	return GetApiV1Recipes200JSONResponse{
		Items:      recipes.Items,
		Page:       recipes.Page,
		Limit:      recipes.Limit,
		TotalCount: recipes.TotalCount,
	}, nil
}

func (s *ServerHandler) PostApiV1Recipes(ctx context.Context, request PostApiV1RecipesRequestObject) (PostApiV1RecipesResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	created, err := s.recipeService.CreateRecipe(ctx, userID, *request.Body)
	if err != nil {
		return PostApiV1Recipes201JSONResponse{}, err
	}
	return PostApiV1Recipes201JSONResponse(created), nil
}

func (s *ServerHandler) PatchApiV1RecipesRecipeId(ctx context.Context, request PatchApiV1RecipesRecipeIdRequestObject) (PatchApiV1RecipesRecipeIdResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	updated, err := s.recipeService.UpdateRecipe(ctx, userID, request.RecipeId, *request.Body)
	if err != nil {
		return PatchApiV1RecipesRecipeId200JSONResponse{}, err
	}
	return PatchApiV1RecipesRecipeId200JSONResponse(updated), nil
}

func (s *ServerHandler) DeleteApiV1RecipesRecipeId(ctx context.Context, request DeleteApiV1RecipesRecipeIdRequestObject) (DeleteApiV1RecipesRecipeIdResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	if err := s.recipeService.DeleteRecipe(ctx, userID, request.RecipeId); err != nil {
		return DeleteApiV1RecipesRecipeId204Response{}, err
	}
	return DeleteApiV1RecipesRecipeId204Response{}, nil
}
