package recipe

import (
	"context"
	"errors"
	"fmt"
	"net/http"

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
	BrewMethod *BrewMethod
	Difficulty *Difficulty
	Pagination pagination.Pagination
}

func ParseListRecipesParams(r *http.Request) (ListRecipesParams, error) {
	q := r.URL.Query()

	params := ListRecipesParams{
		Pagination: pagination.FromRequest(r),
		AuthorID:   q.Get("author_id"),
	}

	if s := q.Get("brew_method"); s != "" {
		val, err := ParseBrewMethod(s)
		if err != nil {
			return params, fmt.Errorf("invalid brew_method param: %w", err)
		}
		params.BrewMethod = &val
	}
	if s := q.Get("difficulty"); s != "" {
		val, err := ParseDifficulty(s)
		if err != nil {
			return params, fmt.Errorf("invalid difficulty param: %w", err)
		}
		params.Difficulty = &val
	}
	return params, nil
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
