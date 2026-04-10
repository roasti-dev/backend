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

type likeRepository interface {
	Exists(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (bool, error)
	Create(ctx context.Context, like Like) error
	Delete(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) error
	GetLikedIDs(ctx context.Context, userID string, targetType models.LikeTargetType, targetIDs []string) (map[string]bool, error)
	CountByTargets(ctx context.Context, targetIDs []string, targetType models.LikeTargetType) (map[string]int, error)
	CountByUser(ctx context.Context, userID string, targetType models.LikeTargetType) (int, error)
	ListByUser(ctx context.Context, userID string, targetType models.LikeTargetType, limit, offset int) ([]Like, error)
}

type Service struct {
	repo likeRepository
}

func NewService(repo likeRepository) *Service {
	return &Service{repo: repo}
}

type ToggleResult struct {
	Liked      bool
	LikesCount int
}

func (s *Service) GetInfo(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (Info, error) {
	batch, err := s.GetInfoBatch(ctx, userID, targetType, []string{targetID})
	if err != nil {
		return Info{}, err
	}
	return batch[targetID], nil
}

func (s *Service) GetInfoBatch(ctx context.Context, userID string, targetType models.LikeTargetType, targetIDs []string) (map[string]Info, error) {
	likedIDs, err := s.repo.GetLikedIDs(ctx, userID, targetType, targetIDs)
	if err != nil {
		return nil, err
	}
	counts, err := s.repo.CountByTargets(ctx, targetIDs, targetType)
	if err != nil {
		return nil, err
	}
	result := make(map[string]Info, len(targetIDs))
	for _, id := range targetIDs {
		result[id] = Info{IsLiked: likedIDs[id], Count: counts[id]}
	}
	return result, nil
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

	counts, err := s.repo.CountByTargets(ctx, []string{targetID}, targetType)
	if err != nil {
		return ToggleResult{}, fmt.Errorf("count likes: %w", err)
	}

	return ToggleResult{Liked: liked, LikesCount: counts[targetID]}, nil
}

func (s *Service) CountByUser(ctx context.Context, userID string, targetType models.LikeTargetType) (int, error) {
	return s.repo.CountByUser(ctx, userID, targetType)
}

func (s *Service) ListByUser(ctx context.Context, userID string, targetType models.LikeTargetType, limit, offset int) ([]Like, error) {
	return s.repo.ListByUser(ctx, userID, targetType, limit, offset)
}
