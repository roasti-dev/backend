package recipe

import (
	"context"
	"errors"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/ids"
	"github.com/nikpivkin/roasti-app-backend/internal/pagination"
)

var ErrEmptyTitle = errors.New("recipe title cannot be empty")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateRecipe(ctx context.Context, userID string, recipe Recipe) (Recipe, error) {
	if recipe.Title == "" {
		return Recipe{}, ErrEmptyTitle
	}
	recipe.ID = ids.NewID()
	recipe.AuthorID = userID
	if err := s.repo.CreateRecipe(ctx, recipe); err != nil {
		return recipe, nil
	}
	return recipe, nil
}

type ListRecipesParams struct {
	AuthorID   string
	BrewMethod *models.BrewMethod
	Difficulty *models.Difficulty
	Pagination pagination.Pagination
}

func (s *Service) ListRecipes(ctx context.Context, userID string, params ListRecipesParams) (pagination.PaginatedResult[Recipe], error) {
	if params.AuthorID == "" {
		params.AuthorID = userID
	} else {
		// TODO: check permissions
	}
	return s.repo.ListRecipes(ctx, params)
}

func (s *Service) DeleteRecioe(ctx context.Context, userID, recipeID string) error {
	return s.repo.DeleteRecipe(ctx, userID, recipeID)
}
