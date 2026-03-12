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

func (s *Service) CreateRecipe(ctx context.Context, userID string, recipe models.Recipe) (models.Recipe, error) {
	if recipe.Title == "" {
		return models.Recipe{}, ErrInvalidTitle
	}
	recipe.Id = ids.NewID()
	recipe.AuthorId = userID
	if err := s.repo.CreateRecipe(ctx, recipe); err != nil {
		return recipe, nil
	}
	return recipe, nil
}

type ListRecipesParams struct {
	AuthorID   *string
	BrewMethod *models.BrewMethod
	Difficulty *models.Difficulty
	Pagination pagination.Pagination
}

func (s *Service) ListRecipes(
	ctx context.Context, userID string, params ListRecipesParams,
) (pagination.PaginatedResult[models.Recipe], error) {
	return s.repo.ListRecipes(ctx, params, userID)
}

type UpdateRecipeParams struct {
	Title       *string            `json:"title,omitempty"`
	Description *string            `json:"description,omitempty"`
	ImageURL    *string            `json:"image_url,omitempty"`
	BrewMethod  *models.BrewMethod `json:"brew_method,omitempty"`
	Difficulty  *models.Difficulty `json:"difficulty,omitempty"`
	RoastLevel  *models.RoastLevel `json:"roast_level,omitempty"`
	Beans       *string            `json:"beans,omitempty"`
	Public      *bool              `json:"public,omitempty"`
}

func (p UpdateRecipeParams) Validate() error {
	if p.Title != nil && strings.TrimSpace(*p.Title) == "" {
		return ErrInvalidTitle
	}

	if p.Description != nil && strings.TrimSpace(*p.Description) == "" {
		return ErrInvalidDescription
	}

	if p.BrewMethod != nil && *p.BrewMethod == models.BrewMethodNone {
		return ErrInvalidBrewMethod
	}

	if p.Difficulty != nil && *p.Difficulty == models.DifficultyNone {
		return ErrInvalidDifficulty
	}
	return nil
}

func (s *Service) UpdateRecipe(
	ctx context.Context, userID, recipeID string, params UpdateRecipeParams,
) (models.Recipe, error) {
	recipe, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Recipe{}, ErrNotFound
		}
		return models.Recipe{}, err
	}

	if err := params.Validate(); err != nil {
		return models.Recipe{}, err
	}

	if recipe.AuthorId != userID {
		return models.Recipe{}, ErrForbidden
	}
	return s.repo.UpdateRecipe(ctx, userID, recipeID, params)
}

func (s *Service) DeleteRecioe(ctx context.Context, userID, recipeID string) error {
	return s.repo.DeleteRecipe(ctx, userID, recipeID)
}
