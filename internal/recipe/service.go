package recipe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/ids"
	"github.com/nikpivkin/roasti-app-backend/internal/pagination"
)

type ApiError struct {
	Status  int
	Message string
}

func (e *ApiError) Error() string {
	return fmt.Sprintf("%d: %s", e.Status, e.Message)
}

func NewApiError(status int, msg string) *ApiError {
	return &ApiError{
		Status:  status,
		Message: msg,
	}
}

var (
	ErrForbidden          = NewApiError(403, "forbidden")
	ErrNotFound           = NewApiError(404, "recipe not found")
	ErrInvalidTitle       = NewApiError(400, "title cannot be empty")
	ErrInvalidDescription = NewApiError(400, "description cannot be empty")
	ErrInvalidBrewMethod  = NewApiError(400, "invalid brew method")
	ErrInvalidDifficulty  = NewApiError(400, "invalid difficulty")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListRecipes(
	ctx context.Context, userID string, params models.ListRecipesParams,
) (pagination.PaginatedResult[models.Recipe], error) {
	return s.repo.ListRecipes(ctx, userID, params)
}

func ValidateCreateRecipe(req models.CreateRecipeRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return ErrInvalidTitle
	}
	if strings.TrimSpace(req.Description) == "" {
		return ErrInvalidDescription
	}
	if req.BrewMethod == models.BrewMethodNone {
		return ErrInvalidBrewMethod
	}
	if req.Difficulty == models.DifficultyNone {
		return ErrInvalidDifficulty
	}
	return nil
}

func (s *Service) CreateRecipe(ctx context.Context, userID string, request models.CreateRecipeRequest) (models.Recipe, error) {
	if request.Title == "" {
		return models.Recipe{}, ErrInvalidTitle
	}

	recipe := models.Recipe{
		Title:       request.Title,
		Description: request.Description,
		ImageUrl:    request.ImageUrl,
		BrewMethod:  request.BrewMethod,
		Difficulty:  request.Difficulty,
		RoastLevel:  request.RoastLevel,
		Beans:       request.Beans,
		Public:      request.Public != nil && *request.Public,
		Steps:       request.Steps,
	}

	recipe.Id = ids.NewID()
	recipe.AuthorId = userID
	if err := s.repo.CreateRecipe(ctx, recipe); err != nil {
		return recipe, nil
	}
	return recipe, nil
}

func ValidatePatchRecipe(req models.PatchRecipeRequest) error {
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return ErrInvalidTitle
	}

	if req.Description != nil && strings.TrimSpace(*req.Description) == "" {
		return ErrInvalidDescription
	}

	if req.BrewMethod != nil && *req.BrewMethod == models.BrewMethodNone {
		return ErrInvalidBrewMethod
	}

	if req.Difficulty != nil && *req.Difficulty == models.DifficultyNone {
		return ErrInvalidDifficulty
	}
	return nil
}

func (s *Service) UpdateRecipe(
	ctx context.Context, userID, recipeID string, requst models.PatchRecipeRequest,
) (models.Recipe, error) {
	recipe, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Recipe{}, ErrNotFound
		}
		return models.Recipe{}, err
	}

	if err := ValidatePatchRecipe(requst); err != nil {
		return models.Recipe{}, err
	}

	if recipe.AuthorId != userID {
		return models.Recipe{}, ErrForbidden
	}
	return s.repo.UpdateRecipe(ctx, userID, recipeID, requst)
}

func (s *Service) DeleteRecioe(ctx context.Context, userID, recipeID string) error {
	return s.repo.DeleteRecipe(ctx, userID, recipeID)
}
