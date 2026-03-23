package recipes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/ids"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
)

type LikeChecker interface {
	IsLiked(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (bool, error)
	GetLikedIDs(ctx context.Context, userID string, targetType models.LikeTargetType, targetIDs []string) (map[string]bool, error)
}

type Service struct {
	logger      *slog.Logger
	repo        *Repository
	uploader    *uploads.Service
	likeChecker LikeChecker
}

func NewService(repo *Repository, uploader *uploads.Service, likeChecker LikeChecker) *Service {
	return &Service{
		logger:      slog.Default(),
		repo:        repo,
		uploader:    uploader,
		likeChecker: likeChecker,
	}
}

func (s *Service) GetRecipeByID(ctx context.Context, userID, recipeID string) (models.Recipe, error) {
	recipe, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Recipe{}, ErrNotFound
		}
		return models.Recipe{}, err
	}

	if !recipe.Public && recipe.AuthorId != userID {
		return models.Recipe{}, ErrForbidden
	}

	recipe.IsLiked, err = s.likeChecker.IsLiked(ctx, userID, recipeID, models.LikeTargetTypeRecipe)
	if err != nil {
		return models.Recipe{}, err
	}

	return recipe, nil
}

func (s *Service) ListRecipes(
	ctx context.Context, userID string, params models.ListRecipesParams,
) (models.GenericPage[models.Recipe], error) {
	page, err := s.repo.ListRecipes(ctx, userID, params)
	if err != nil {
		return models.GenericPage[models.Recipe]{}, err
	}

	ids := make([]string, len(page.Items))
	for i, r := range page.Items {
		ids[i] = r.Id
	}

	likedIDs, err := s.likeChecker.GetLikedIDs(ctx, userID, models.LikeTargetTypeRecipe, ids)
	if err != nil {
		return models.GenericPage[models.Recipe]{}, err
	}

	for i, r := range page.Items {
		page.Items[i].IsLiked = likedIDs[r.Id]
	}

	return page, nil
}

func (s *Service) GetPreviewsByIDs(ctx context.Context, currentUserID, ownerID string, ids []string) ([]models.RecipePreview, error) {
	previews, err := s.repo.GetPreviewsByIDs(ctx, currentUserID, ids)
	if err != nil {
		return nil, fmt.Errorf("get recipe previews: %w", err)
	}

	likedIDs := make([]string, len(previews))
	for i, p := range previews {
		likedIDs[i] = p.Id
	}

	likedMap, err := s.likeChecker.GetLikedIDs(ctx, currentUserID, models.LikeTargetTypeRecipe, likedIDs)
	if err != nil {
		return nil, fmt.Errorf("get liked ids: %w", err)
	}

	for i := range previews {
		previews[i].IsLiked = likedMap[previews[i].Id]
	}

	return previews, nil
}

func (s *Service) CreateRecipe(ctx context.Context, userID string, request models.CreateRecipeRequest) (models.Recipe, error) {
	if err := validateRecipePayload(request); err != nil {
		return models.Recipe{}, err
	}

	recipe := recipePayloadToModel(request)

	recipe.Id = ids.NewID()
	recipe.AuthorId = userID
	if err := s.repo.UpsertRecipe(ctx, recipe); err != nil {
		return recipe, err
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

func (s *Service) CloneRecipe(ctx context.Context, userID, recipeID string) (models.Recipe, error) {
	original, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return models.Recipe{}, err
	}

	if !original.Public {
		return models.Recipe{}, ErrNotFound
	}

	if original.AuthorId == userID {
		return models.Recipe{}, ErrForbidden
	}

	clone := original
	clone.Id = ids.NewID()
	clone.AuthorId = userID
	clone.Title = "Copy of " + original.Title
	clone.Origin = &models.RecipeOrigin{
		RecipeId: recipeID,
	}
	clone.LikesCount = 0
	clone.CreatedAt = time.Now().UTC()
	clone.UpdatedAt = time.Now().UTC()
	clone.Steps = original.Steps

	if err := s.repo.UpsertRecipe(ctx, clone); err != nil {
		return models.Recipe{}, err
	}

	return s.repo.GetRecipeByID(ctx, clone.Id)
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
		if err := s.uploader.Confirm(ctx, *recipe.ImageId); err != nil {
			s.logger.WarnContext(ctx, "failed to confirm recipe image",
				slog.String("recipe_id", recipe.Id),
				slog.String("image_id", *recipe.ImageId),
			)
		}
	}

	for _, step := range recipe.Steps {
		if step.ImageId != nil {
			if err := s.uploader.Confirm(ctx, *step.ImageId); err != nil {
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
