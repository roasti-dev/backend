package handlers

import (
	"context"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/x/ptr"
	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
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
	params := models.ListRecipesParams{
		Query:         request.Params.Query,
		AuthorId:      request.Params.AuthorId,
		BrewMethod:    request.Params.BrewMethod,
		Difficulty:    request.Params.Difficulty,
		RoastLevel:    request.Params.RoastLevel,
		Limit:         request.Params.Limit,
		Page:          request.Params.Page,
		SortDirection: request.Params.SortDirection,
	}

	if request.Params.SortField != nil {
		params.SortField = new(models.ListRecipesParamsSortField(*request.Params.SortField))
	}
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

	result, err := s.recipeService.ToggleLike(ctx, userID, request.RecipeId)
	if err != nil {
		return nil, err
	}

	return ToggleRecipeLike200JSONResponse(models.ToggleLikeResponse{
		Liked:      result.Liked,
		LikesCount: int32(result.LikesCount),
	}), nil
}

func (s *ServerHandler) CloneRecipe(ctx context.Context, request CloneRecipeRequestObject) (CloneRecipeResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	cloned, err := s.recipeService.CloneRecipe(ctx, userID, request.RecipeId)
	if err != nil {
		return nil, err
	}
	return CloneRecipe201JSONResponse(cloned), nil
}

func (s *ServerHandler) ListRecipeComments(ctx context.Context, request ListRecipeCommentsRequestObject) (ListRecipeCommentsResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	pag := models.NewPaginationParams(ptr.FromPtr(request.Params.Page), ptr.FromPtr(request.Params.Limit))
	page, err := s.recipeService.ListComments(ctx, userID, request.RecipeId, pag)
	if err != nil {
		return nil, err
	}
	return ListRecipeComments200JSONResponse(models.CommentPage(page)), nil
}

func (s *ServerHandler) CreateRecipeComment(ctx context.Context, request CreateRecipeCommentRequestObject) (CreateRecipeCommentResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	comment, err := s.recipeService.CreateComment(ctx, userID, request.RecipeId, request.Body.Text, request.Body.ParentId)
	if err != nil {
		return nil, err
	}
	return CreateRecipeComment201JSONResponse(comment), nil
}

func (s *ServerHandler) UpdateRecipeComment(ctx context.Context, request UpdateRecipeCommentRequestObject) (UpdateRecipeCommentResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	comment, err := s.recipeService.UpdateComment(ctx, userID, request.CommentId, request.Body.Text)
	if err != nil {
		return nil, err
	}
	return UpdateRecipeComment200JSONResponse(comment), nil
}

func (s *ServerHandler) DeleteRecipeComment(ctx context.Context, request DeleteRecipeCommentRequestObject) (DeleteRecipeCommentResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	if err := s.recipeService.DeleteComment(ctx, userID, request.CommentId); err != nil {
		return nil, err
	}
	return DeleteRecipeComment204Response{}, nil
}
