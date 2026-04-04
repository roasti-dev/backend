package recipes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/events"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
)

type RecipeRepository interface {
	UpsertRecipe(ctx context.Context, recipe models.Recipe) error
	GetRecipeByID(ctx context.Context, recipeID string) (models.Recipe, error)
	ListRecipes(ctx context.Context, currentUserID string, params models.ListRecipesParams) (models.GenericPage[models.Recipe], error)
	GetRecipesByIDs(ctx context.Context, currentUserID string, ids []string) ([]models.Recipe, error)
	GetPreviewsByIDs(ctx context.Context, currentUserID string, ids []string) ([]models.RecipePreview, error)
	DeleteRecipe(ctx context.Context, userID, recipeID string) error
}

// Uploader manages uploaded files.
type Uploader interface {
	Confirm(ctx context.Context, fileID string) error
	Copy(ctx context.Context, fileID string) (string, error)
	Delete(ctx context.Context, fileID string) error
}

type LikeChecker interface {
	IsLiked(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (bool, error)
	GetLikedIDs(ctx context.Context, userID string, targetType models.LikeTargetType, targetIDs []string) (map[string]bool, error)
	CountByTarget(ctx context.Context, targetID string, targetType models.LikeTargetType) (int, error)
	CountByTargets(ctx context.Context, targetIDs []string, targetType models.LikeTargetType) (map[string]int, error)
}

type LikeToggler interface {
	Toggle(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (likes.ToggleResult, error)
}

type EventPublisher interface {
	Publish(e events.Event)
}

type CommentService interface {
	Create(ctx context.Context, userID, targetID, targetType, text string, parentID *string) (models.PostComment, error)
	Update(ctx context.Context, userID, commentID, text string) (models.PostComment, error)
	Delete(ctx context.Context, userID, commentID string) error
	List(ctx context.Context, targetID string, pag models.PaginationParams) (models.GenericPage[models.CommentThread], error)
}

type Service struct {
	logger         *slog.Logger
	repo           RecipeRepository
	uploader       Uploader
	likeChecker    LikeChecker
	likeToggler    LikeToggler
	publisher      EventPublisher
	commentService CommentService
}

func NewService(repo RecipeRepository, uploader Uploader, likeChecker LikeChecker, likeToggler LikeToggler, publisher EventPublisher, commentService CommentService) *Service {
	return &Service{
		logger:         slog.Default(),
		repo:           repo,
		uploader:       uploader,
		likeChecker:    likeChecker,
		likeToggler:    likeToggler,
		publisher:      publisher,
		commentService: commentService,
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
		return models.Recipe{}, ErrNotFound
	}

	recipe.IsLiked, err = s.likeChecker.IsLiked(ctx, userID, recipeID, models.LikeTargetTypeRecipe)
	if err != nil {
		return models.Recipe{}, err
	}

	likesCount, err := s.likeChecker.CountByTarget(ctx, recipeID, models.LikeTargetTypeRecipe)
	if err != nil {
		return models.Recipe{}, err
	}
	recipe.LikesCount = int32(likesCount)

	recipe.RedactForUser(userID)

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

	likesCounts, err := s.likeChecker.CountByTargets(ctx, ids, models.LikeTargetTypeRecipe)
	if err != nil {
		return models.GenericPage[models.Recipe]{}, err
	}

	for i, r := range page.Items {
		page.Items[i].IsLiked = likedIDs[r.Id]
		page.Items[i].LikesCount = int32(likesCounts[r.Id])
		page.Items[i].RedactForUser(userID)
	}

	return page, nil
}

func (s *Service) GetRecipesByIDs(ctx context.Context, currentUserID string, ids []string) ([]models.Recipe, error) {
	recipes, err := s.repo.GetRecipesByIDs(ctx, currentUserID, ids)
	if err != nil {
		return nil, fmt.Errorf("get recipe previews: %w", err)
	}

	likedMap, err := s.likeChecker.GetLikedIDs(ctx, currentUserID, models.LikeTargetTypeRecipe, ids)
	if err != nil {
		return nil, fmt.Errorf("get liked ids: %w", err)
	}

	likesCounts, err := s.likeChecker.CountByTargets(ctx, ids, models.LikeTargetTypeRecipe)
	if err != nil {
		return nil, fmt.Errorf("count likes: %w", err)
	}

	for i := range recipes {
		recipes[i].IsLiked = likedMap[recipes[i].Id]
		recipes[i].LikesCount = int32(likesCounts[recipes[i].Id])
		recipes[i].RedactForUser(currentUserID)
	}

	return recipes, nil
}

func (s *Service) GetPreviewsByIDs(ctx context.Context, currentUserID, ownerID string, ids []string) ([]models.RecipePreview, error) {
	previews, err := s.repo.GetPreviewsByIDs(ctx, currentUserID, ids)
	if err != nil {
		return nil, fmt.Errorf("get recipe previews: %w", err)
	}

	previewIDs := make([]string, len(previews))
	for i, p := range previews {
		previewIDs[i] = p.Id
	}

	likedMap, err := s.likeChecker.GetLikedIDs(ctx, currentUserID, models.LikeTargetTypeRecipe, previewIDs)
	if err != nil {
		return nil, fmt.Errorf("get liked ids: %w", err)
	}

	likesCounts, err := s.likeChecker.CountByTargets(ctx, previewIDs, models.LikeTargetTypeRecipe)
	if err != nil {
		return nil, fmt.Errorf("count likes: %w", err)
	}

	for i := range previews {
		previews[i].IsLiked = likedMap[previews[i].Id]
		previews[i].LikesCount = int32(likesCounts[previews[i].Id])
	}

	return previews, nil
}

func normalizeRecipePayload(req *models.RecipePayload) {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	for i := range req.Steps {
		req.Steps[i].Title = strings.TrimSpace(req.Steps[i].Title)
		if req.Steps[i].Description != nil {
			trimmed := strings.TrimSpace(*req.Steps[i].Description)
			req.Steps[i].Description = &trimmed
		}
	}
}

func (s *Service) CreateRecipe(ctx context.Context, userID string, request models.CreateRecipeRequest) (models.Recipe, error) {
	normalizeRecipePayload(&request)
	if err := validateRecipePayload(request); err != nil {
		return models.Recipe{}, err
	}

	recipe := recipePayloadToModel(request)

	recipe.Id = id.NewID()
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
	normalizeRecipePayload(&request)
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
		if errors.Is(err, sql.ErrNoRows) {
			return models.Recipe{}, ErrNotFound
		}
		return models.Recipe{}, err
	}

	if original.AuthorId == userID {
		return models.Recipe{}, ErrForbidden
	}

	if !original.Public {
		return models.Recipe{}, ErrNotFound
	}

	clone := original.CloneFor(userID, id.NewID(), time.Now().UTC())

	if original.ImageId != nil {
		newID, err := s.uploader.Copy(ctx, *original.ImageId)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to copy recipe image",
				slog.String("recipe_id", recipeID),
				slog.String("image_id", *original.ImageId),
			)
		} else {
			clone.ImageId = &newID
		}
	}

	for i, step := range clone.Steps {
		if step.ImageId == nil {
			continue
		}
		newID, err := s.uploader.Copy(ctx, *step.ImageId)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to copy step image",
				slog.Int64("step_order", int64(step.Order)),
				slog.String("image_id", *step.ImageId),
			)
			clone.Steps[i].ImageId = nil
		} else {
			clone.Steps[i].ImageId = &newID
		}
	}

	if err := s.repo.UpsertRecipe(ctx, clone); err != nil {
		return models.Recipe{}, err
	}

	return s.repo.GetRecipeByID(ctx, clone.Id)
}

func (s *Service) ToggleLike(ctx context.Context, userID, recipeID string) (likes.ToggleResult, error) {
	recipe, err := s.GetRecipeByID(ctx, userID, recipeID)
	if err != nil {
		return likes.ToggleResult{}, err
	}

	result, err := s.likeToggler.Toggle(ctx, userID, recipeID, models.LikeTargetTypeRecipe)
	if err != nil {
		return likes.ToggleResult{}, err
	}

	if s.publisher != nil {
		s.publisher.Publish(events.RecipeLikeToggled{
			RecipeID: recipeID,
			OwnerID:  recipe.AuthorId,
			ByUserID: userID,
			Liked:    result.Liked,
		})
	}

	return result, nil
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
		if !recipe.Public {
			return ErrNotFound
		}
		return ErrForbidden
	}

	if err := s.repo.DeleteRecipe(ctx, userID, recipeID); err != nil {
		return err
	}

	s.deleteRecipeImages(ctx, recipe)
	return nil
}

func (s *Service) deleteRecipeImages(ctx context.Context, recipe models.Recipe) {
	if recipe.ImageId != nil {
		if err := s.uploader.Delete(ctx, *recipe.ImageId); err != nil {
			s.logger.WarnContext(ctx, "failed to delete recipe image",
				slog.String("recipe_id", recipe.Id),
				slog.String("image_id", *recipe.ImageId),
			)
		}
	}

	for _, step := range recipe.Steps {
		if step.ImageId != nil {
			if err := s.uploader.Delete(ctx, *step.ImageId); err != nil {
				s.logger.WarnContext(ctx, "failed to delete step image",
					slog.Int64("step_id", step.Id),
					slog.String("image_id", *step.ImageId),
				)
			}
		}
	}
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
		Note:        payload.Note,
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

func (s *Service) CreateComment(ctx context.Context, userID, recipeID, text string, parentID *string) (models.PostComment, error) {
	recipe, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PostComment{}, ErrNotFound
		}
		return models.PostComment{}, err
	}
	if !recipe.Public && recipe.AuthorId != userID {
		return models.PostComment{}, ErrNotFound
	}
	return s.commentService.Create(ctx, userID, recipeID, "recipe", text, parentID)
}

func (s *Service) ListComments(ctx context.Context, userID, recipeID string, pag models.PaginationParams) (models.GenericPage[models.CommentThread], error) {
	recipe, err := s.repo.GetRecipeByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GenericPage[models.CommentThread]{}, ErrNotFound
		}
		return models.GenericPage[models.CommentThread]{}, err
	}
	if !recipe.Public && recipe.AuthorId != userID {
		return models.GenericPage[models.CommentThread]{}, ErrNotFound
	}
	return s.commentService.List(ctx, recipeID, pag)
}

func (s *Service) UpdateComment(ctx context.Context, userID, commentID, text string) (models.PostComment, error) {
	return s.commentService.Update(ctx, userID, commentID, text)
}

func (s *Service) DeleteComment(ctx context.Context, userID, commentID string) error {
	return s.commentService.Delete(ctx, userID, commentID)
}
