package handlers

import (
	"context"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/ptr"
	"github.com/nikpivkin/roasti-app-backend/internal/requestctx"
)

func (s *ServerHandler) GetRecipe(ctx context.Context, request GetRecipeRequestObject) (GetRecipeResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	recipe, err := s.recipeService.GetRecipeByID(ctx, userID, request.RecipeId)
	if err != nil {
		return nil, err
	}
	return GetRecipe200JSONResponse(recipe), nil
}

func (s *ServerHandler) ListRecipes(ctx context.Context, request ListRecipesRequestObject) (ListRecipesResponseObject, error) {
	s.logger.DebugContext(ctx, "list recipes request")

	userID := requestctx.GetUserID(ctx)
	params := ptr.GetOr(request.Params.ListRecipes, models.ListRecipesParams{})
	recipePage, err := s.recipeService.ListRecipes(ctx, userID, params)
	if err != nil {
		return nil, err
	}

	s.logger.DebugContext(ctx, "recipes returned",
		slog.Int("count", len(recipePage.Items)),
		slog.Any("pagination", recipePage.Pagination),
	)
	return ListRecipes200JSONResponse{
		Items:      recipePage.Items,
		Pagination: recipePage.Pagination,
	}, nil
}

func (s *ServerHandler) CreateRecipe(ctx context.Context, request CreateRecipeRequestObject) (CreateRecipeResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	created, err := s.recipeService.CreateRecipe(ctx, userID, *request.Body)
	if err != nil {
		return nil, err
	}
	return CreateRecipe201JSONResponse(created), nil
}

func (s *ServerHandler) UpdateRecipe(ctx context.Context, request UpdateRecipeRequestObject) (UpdateRecipeResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	updated, err := s.recipeService.UpdateRecipe(ctx, userID, request.RecipeId, *request.Body)
	if err != nil {
		return nil, err
	}
	return UpdateRecipe200JSONResponse(updated), nil
}

func (s *ServerHandler) DeleteRecipe(ctx context.Context, request DeleteRecipeRequestObject) (DeleteRecipeResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	if err := s.recipeService.DeleteRecipe(ctx, userID, request.RecipeId); err != nil {
		return nil, err
	}
	return DeleteRecipe204Response{}, nil
}

func (s *ServerHandler) ToggleRecipeLike(ctx context.Context, request ToggleRecipeLikeRequestObject) (ToggleRecipeLikeResponseObject, error) {
	userID := requestctx.GetUserID(ctx)

	result, err := s.likeService.Toggle(ctx, userID, request.RecipeId, models.LikeTargetTypeRecipe)
	if err != nil {
		return nil, err
	}

	return ToggleRecipeLike200JSONResponse(models.ToggleLikeResponse{
		Liked:      result.Liked,
		LikesCount: int32(result.LikesCount),
	}), nil
}
