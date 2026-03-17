package handlers

import (
	"context"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/ptr"
	"github.com/nikpivkin/roasti-app-backend/internal/requestctx"
)

func (s *ServerHandler) GetApiV1RecipesRecipeId(ctx context.Context, request GetApiV1RecipesRecipeIdRequestObject) (GetApiV1RecipesRecipeIdResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	recipe, err := s.recipeService.GetRecipeByID(ctx, userID, request.RecipeId)
	if err != nil {
		return nil, err
	}
	return GetApiV1RecipesRecipeId200JSONResponse(recipe), nil
}

func (s *ServerHandler) GetApiV1Recipes(ctx context.Context, request GetApiV1RecipesRequestObject) (GetApiV1RecipesResponseObject, error) {
	s.logger.DebugContext(ctx, "list recipes request")

	userID := requestctx.GetUserID(ctx)
	recipePage, err := s.recipeService.ListRecipes(ctx, userID, ptr.GetOr(request.Params.ListRecipes, models.ListRecipesParams{}))
	if err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "recipes returned",
		slog.Int("count", len(recipePage.Items)),
		slog.Any("pagination", recipePage.Pagination),
	)
	return GetApiV1Recipes200JSONResponse{
		Items:      recipePage.Items,
		Pagination: recipePage.Pagination,
	}, nil
}

func (s *ServerHandler) PostApiV1Recipes(ctx context.Context, request PostApiV1RecipesRequestObject) (PostApiV1RecipesResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	created, err := s.recipeService.CreateRecipe(ctx, userID, *request.Body)
	if err != nil {
		return nil, err
	}
	return PostApiV1Recipes201JSONResponse(created), nil
}

func (s *ServerHandler) PutApiV1RecipesRecipeId(ctx context.Context, request PutApiV1RecipesRecipeIdRequestObject) (PutApiV1RecipesRecipeIdResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	updated, err := s.recipeService.UpdateRecipe(ctx, userID, request.RecipeId, *request.Body)
	if err != nil {
		return nil, err
	}
	return PutApiV1RecipesRecipeId200JSONResponse(updated), nil
}

func (s *ServerHandler) PatchApiV1RecipesRecipeId(ctx context.Context, request PatchApiV1RecipesRecipeIdRequestObject) (PatchApiV1RecipesRecipeIdResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	updated, err := s.recipeService.PatchRecipe(ctx, userID, request.RecipeId, *request.Body)
	if err != nil {
		return nil, err
	}
	return PatchApiV1RecipesRecipeId200JSONResponse(updated), nil
}

func (s *ServerHandler) DeleteApiV1RecipesRecipeId(ctx context.Context, request DeleteApiV1RecipesRecipeIdRequestObject) (DeleteApiV1RecipesRecipeIdResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	if err := s.recipeService.DeleteRecipe(ctx, userID, request.RecipeId); err != nil {
		return nil, err
	}
	return DeleteApiV1RecipesRecipeId204Response{}, nil
}
