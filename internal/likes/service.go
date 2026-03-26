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

type LikeRepository interface {
	Exists(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (bool, error)
	Create(ctx context.Context, like Like) error
	Delete(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) error
	CountByTarget(ctx context.Context, targetID string, targetType models.LikeTargetType) (int, error)
	CountByUser(ctx context.Context, userID string, targetType models.LikeTargetType) (int, error)
	GetLikedIDs(ctx context.Context, userID string, targetType models.LikeTargetType, targetIDs []string) (map[string]bool, error)
	CountByTargets(ctx context.Context, targetIDs []string, targetType models.LikeTargetType) (map[string]int, error)
	ListByUser(ctx context.Context, userID string, targetType models.LikeTargetType, limit, offset int) ([]Like, error)
}

type Service struct {
	repo LikeRepository
}

func NewService(repo LikeRepository) *Service {
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

func (s *Service) CountByUser(ctx context.Context, userID string, targetType models.LikeTargetType) (int, error) {
	return s.repo.CountByUser(ctx, userID, targetType)
}

func (s *Service) ListByUser(ctx context.Context, userID string, targetType models.LikeTargetType, limit, offset int) ([]Like, error) {
	return s.repo.ListByUser(ctx, userID, targetType, limit, offset)
}
