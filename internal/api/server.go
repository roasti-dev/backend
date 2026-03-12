package api

import (
	"context"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/pagination"
	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
	"github.com/nikpivkin/roasti-app-backend/internal/requestctx"
)

var _ StrictServerInterface = (*ServerHandler)(nil)

type ServerHandler struct {
	logger        *slog.Logger
	recipeService *recipe.Service
}

func NewServerHandler(recipeService *recipe.Service) *ServerHandler {
	return &ServerHandler{
		logger:        slog.Default(),
		recipeService: recipeService,
	}
}

func (s *ServerHandler) GetApiV1Recipes(ctx context.Context, request GetApiV1RecipesRequestObject) (GetApiV1RecipesResponseObject, error) {
	s.logger.DebugContext(ctx, "list recipes request")

	userID := requestctx.GetUserID(ctx)

	params := recipe.ListRecipesParams{
		Pagination: pagination.New(
			getOrDefault(request.Params.Page, pagination.DefaultPage),
			getOrDefault(request.Params.Limit, pagination.DefaultLimit),
		),
		AuthorID:   request.Params.AuthorId,
		BrewMethod: request.Params.BrewMethod,
		Difficulty: request.Params.Difficulty,
	}

	recipes, err := s.recipeService.ListRecipes(ctx, userID, params)
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

	recipe := models.Recipe{
		Title:       request.Body.Title,
		Description: request.Body.Description,
		ImageUrl:    request.Body.ImageUrl,
		BrewMethod:  request.Body.BrewMethod,
		Difficulty:  request.Body.Difficulty,
		RoastLevel:  request.Body.RoastLevel,
		Beans:       request.Body.Beans,
		Public:      false,
		Steps:       request.Body.Steps,
	}

	if request.Body.Public != nil {
		recipe.Public = *request.Body.Public
	}

	created, err := s.recipeService.CreateRecipe(ctx, userID, recipe)
	if err != nil {
		return PostApiV1Recipes201JSONResponse{}, err
	}
	return PostApiV1Recipes201JSONResponse(created), nil
}

func (s *ServerHandler) PatchApiV1RecipesRecipeId(ctx context.Context, request PatchApiV1RecipesRecipeIdRequestObject) (PatchApiV1RecipesRecipeIdResponseObject, error) {
	userID := requestctx.GetUserID(ctx)

	params := recipe.UpdateRecipeParams{
		Title:       request.Body.Title,
		Description: request.Body.Description,
		ImageURL:    request.Body.ImageUrl,
		BrewMethod:  request.Body.BrewMethod,
		Difficulty:  request.Body.Difficulty,
		RoastLevel:  request.Body.RoastLevel,
		Beans:       request.Body.Beans,
		Public:      request.Body.Public,
	}

	updated, err := s.recipeService.UpdateRecipe(ctx, userID, request.RecipeId, params)
	if err != nil {
		return PatchApiV1RecipesRecipeId200JSONResponse{}, err
	}
	return PatchApiV1RecipesRecipeId200JSONResponse(updated), nil
}

func (s *ServerHandler) DeleteApiV1RecipesRecipeId(ctx context.Context, request DeleteApiV1RecipesRecipeIdRequestObject) (DeleteApiV1RecipesRecipeIdResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	if err := s.recipeService.DeleteRecioe(ctx, userID, request.RecipeId); err != nil {
		return DeleteApiV1RecipesRecipeId204Response{}, err
	}
	return DeleteApiV1RecipesRecipeId204Response{}, nil
}

func (s *ServerHandler) GetHealth(ctx context.Context, request GetHealthRequestObject) (GetHealthResponseObject, error) {
	return GetHealth200TextResponse("OK"), nil
}

func getOrDefault[T any](ptr *T, def T) T {
	if ptr != nil {
		return *ptr
	}
	return def
}
