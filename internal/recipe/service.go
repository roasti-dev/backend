package recipe

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/ids"
	"github.com/nikpivkin/roasti-app-backend/internal/pagination"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
)

type Service struct {
	logger   *slog.Logger
	repo     *Repository
	uploader *uploads.Service
}

func NewService(repo *Repository, uploader *uploads.Service) *Service {
	return &Service{
		logger:   slog.Default(),
		repo:     repo,
		uploader: uploader,
	}
}

func (s *Service) ListRecipes(
	ctx context.Context, userID string, params models.ListRecipesParams,
) (pagination.Page[models.Recipe], error) {
	return s.repo.ListRecipes(ctx, userID, params)
}

func (s *Service) CreateRecipe(ctx context.Context, userID string, request models.CreateRecipeRequest) (models.Recipe, error) {
	if err := validateRecipePayload(request); err != nil {
		return models.Recipe{}, err
	}

	recipe := recipePayloadToModel(request)

	recipe.Id = ids.NewID()
	recipe.AuthorId = userID
	if err := s.repo.UpsertRecipe(ctx, recipe); err != nil {
		return recipe, nil
	}

	created, err := s.repo.GetRecipeByID(ctx, recipe.Id)
	if err != nil {
		return models.Recipe{}, err
	}
	s.confirmRecipeImages(ctx, created)
	return created, nil
}

func (s *Service) UpdateRecipe(
	ctx context.Context, userID, recipeID string, request models.UpdateRecipeRequest,
) (models.Recipe, error) {
	if err := validateRecipePayload(request); err != nil {
		return models.Recipe{}, err
	}

	existing, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Recipe{}, ErrNotFound
		}
		return models.Recipe{}, err
	}

	if existing.AuthorId != userID {
		return models.Recipe{}, ErrForbidden
	}

	recipe := recipePayloadToModel(request)
	recipe.Id = recipeID
	recipe.AuthorId = existing.AuthorId
	recipe.CreatedAt = existing.CreatedAt

	if err := s.repo.UpsertRecipe(ctx, recipe); err != nil {
		return models.Recipe{}, err
	}

	updated, err := s.repo.GetRecipeByID(ctx, recipe.Id)
	if err != nil {
		return models.Recipe{}, err
	}

	s.confirmRecipeImages(ctx, updated)
	return updated, nil
}

func (s *Service) PatchRecipe(
	ctx context.Context, userID, recipeID string, requst models.PatchRecipeRequest,
) (models.Recipe, error) {
	if err := validatePatchRecipe(requst); err != nil {
		return models.Recipe{}, err
	}

	recipe, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Recipe{}, ErrNotFound
		}
		return models.Recipe{}, err
	}

	if recipe.AuthorId != userID {
		return models.Recipe{}, ErrForbidden
	}
	return s.repo.PatchRecipe(ctx, userID, recipeID, requst)
}

func (s *Service) DeleteRecipe(ctx context.Context, userID, recipeID string) error {
	recipe, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if recipe.AuthorId != userID {
		return ErrForbidden
	}

	return s.repo.DeleteRecipe(ctx, userID, recipeID)
}

func (s *Service) confirmRecipeImages(ctx context.Context, recipe models.Recipe) {
	if recipe.ImageId != nil {
		if err := s.uploader.Confirm(*recipe.ImageId); err != nil {
			s.logger.WarnContext(ctx, "failed to confirm recipe image",
				slog.String("recipe_id", recipe.Id),
				slog.String("image_id", *recipe.ImageId),
			)
		}
	}

	for _, step := range recipe.Steps {
		if step.ImageId != nil {
			if err := s.uploader.Confirm(*step.ImageId); err != nil {
				s.logger.WarnContext(ctx, "failed to confirm step image",
					slog.Int64("step_id", step.Id),
					slog.String("image_id", *step.ImageId),
				)
			}
		}
	}
}

func recipePayloadToModel(payload models.RecipePayload) models.Recipe {
	return models.Recipe{
		Title:       payload.Title,
		Description: payload.Description,
		ImageId:     payload.ImageId,
		BrewMethod:  payload.BrewMethod,
		Difficulty:  payload.Difficulty,
		RoastLevel:  payload.RoastLevel,
		Beans:       payload.Beans,
		Public:      payload.Public != nil && *payload.Public,
		Steps:       mapSlice(payload.Steps, brewStepPayloadToModel),
	}
}

func brewStepPayloadToModel(payload models.BrewStepPayload) models.BrewStep {
	return models.BrewStep{
		Title:           payload.Title,
		Description:     payload.Description,
		ImageId:         payload.ImageId,
		Order:           payload.Order,
		DurationSeconds: payload.DurationSeconds,
	}
}

func mapSlice[T, U any](slice []T, f func(T) U) []U {
	result := make([]U, len(slice))
	for i, v := range slice {
		result[i] = f(v)
	}
	return result
}
