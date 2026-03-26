package likes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
)

var ErrTargetNotFound = apierr.NewApiError(http.StatusNotFound, "target not found")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type ToggleResult struct {
	Liked      bool
	LikesCount int
}

func (s *Service) IsLiked(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (bool, error) {
	return s.repo.Exists(ctx, userID, targetID, targetType)
}

func (s *Service) Toggle(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (ToggleResult, error) {
	exists, err := s.repo.Exists(ctx, userID, targetID, targetType)
	if err != nil {
		return ToggleResult{}, fmt.Errorf("check like: %w", err)
	}

	var liked bool
	if !exists {
		like := Like{
			ID:         id.NewID(),
			UserID:     userID,
			TargetID:   targetID,
			TargetType: targetType,
		}
		if err := s.repo.Create(ctx, like); err != nil {
			return ToggleResult{}, fmt.Errorf("create like: %w", err)
		}
		liked = true
	} else {
		if err := s.repo.Delete(ctx, userID, targetID, targetType); err != nil {
			return ToggleResult{}, fmt.Errorf("delete like: %w", err)
		}
		liked = false
	}

	count, err := s.repo.CountByTarget(ctx, targetID, targetType)
	if err != nil {
		return ToggleResult{}, fmt.Errorf("count likes: %w", err)
	}

	return ToggleResult{Liked: liked, LikesCount: count}, nil
}

func (s *Service) GetLikedIDs(ctx context.Context, userID string, targetType models.LikeTargetType, targetIDs []string) (map[string]bool, error) {
	return s.repo.GetLikedIDs(ctx, userID, targetType, targetIDs)
}

func (s *Service) CountByTarget(ctx context.Context, targetID string, targetType models.LikeTargetType) (int, error) {
	return s.repo.CountByTarget(ctx, targetID, targetType)
}

func (s *Service) CountByTargets(ctx context.Context, targetIDs []string, targetType models.LikeTargetType) (map[string]int, error) {
	return s.repo.CountByTargets(ctx, targetIDs, targetType)
}
