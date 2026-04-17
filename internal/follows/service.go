package follows

import (
	"context"
	"fmt"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/events"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
	"github.com/nikpivkin/roasti-app-backend/internal/x/ptr"
)

type followRepository interface {
	Create(ctx context.Context, f Follow) error
	Exists(ctx context.Context, followerID, targetID, targetType string) (bool, error)
	Delete(ctx context.Context, followerID, targetID, targetType string) error
	GetStats(ctx context.Context, targetUserID, currentUserID string) (FollowStats, error)
	ListFollowing(ctx context.Context, followerID, targetType string, limit, offset int) ([]string, int, error)
	ListFollowers(ctx context.Context, targetID, targetType string, limit, offset int) ([]string, int, error)
	ListFollowingUserIDs(ctx context.Context, followerID string) ([]string, error)
}

type userLookup interface {
	GetPreviewsByIDs(ctx context.Context, ids []string) ([]models.UserPreview, error)
	ExistsByID(ctx context.Context, userID string) (bool, error)
}

type eventPublisher interface {
	Publish(e events.Event)
}

type ListParams struct {
	Limit *int32
	Page  *int32
}

func (p ListParams) Pagination() models.PaginationParams {
	return models.NewPaginationParams(
		ptr.GetOr(p.Page, models.DefaultPage),
		ptr.GetOr(p.Limit, models.DefaultLimit),
	)
}

type Service struct {
	repo      followRepository
	users     userLookup
	publisher eventPublisher
}

func NewService(repo followRepository, users userLookup, publisher eventPublisher) *Service {
	return &Service{repo: repo, users: users, publisher: publisher}
}

func (s *Service) Follow(ctx context.Context, followerID, targetID string) error {
	if followerID == targetID {
		return ErrCannotFollowSelf
	}
	exists, err := s.users.ExistsByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("check user exists: %w", err)
	}
	if !exists {
		return ErrNotFound
	}

	alreadyFollowing, err := s.repo.Exists(ctx, followerID, targetID, TargetTypeUser)
	if err != nil {
		return fmt.Errorf("check follow exists: %w", err)
	}

	if err := s.repo.Create(ctx, Follow{
		ID:         id.NewID(),
		FollowerID: followerID,
		TargetID:   targetID,
		TargetType: TargetTypeUser,
	}); err != nil {
		return err
	}

	if !alreadyFollowing {
		s.publisher.Publish(events.UserFollowed{
			FollowerID: followerID,
			TargetID:   targetID,
		})
	}
	return nil
}

func (s *Service) Unfollow(ctx context.Context, followerID, targetID string) error {
	if followerID == targetID {
		return ErrCannotFollowSelf
	}
	return s.repo.Delete(ctx, followerID, targetID, TargetTypeUser)
}

func (s *Service) GetStats(ctx context.Context, targetUserID, currentUserID string) (FollowStats, error) {
	return s.repo.GetStats(ctx, targetUserID, currentUserID)
}

func (s *Service) ListFollowing(ctx context.Context, userID string, params ListParams) (models.GenericPage[models.UserPreview], error) {
	pag := params.Pagination()
	ids, total, err := s.repo.ListFollowing(ctx, userID, TargetTypeUser, int(pag.GetLimit()), int(pag.Offset()))
	if err != nil {
		return models.GenericPage[models.UserPreview]{}, fmt.Errorf("list following: %w", err)
	}
	previews, err := s.users.GetPreviewsByIDs(ctx, ids)
	if err != nil {
		return models.GenericPage[models.UserPreview]{}, err
	}
	return models.NewPage(previews, pag, total), nil
}

func (s *Service) ListFollowers(ctx context.Context, userID string, params ListParams) (models.GenericPage[models.UserPreview], error) {
	pag := params.Pagination()
	ids, total, err := s.repo.ListFollowers(ctx, userID, TargetTypeUser, int(pag.GetLimit()), int(pag.Offset()))
	if err != nil {
		return models.GenericPage[models.UserPreview]{}, fmt.Errorf("list followers: %w", err)
	}
	previews, err := s.users.GetPreviewsByIDs(ctx, ids)
	if err != nil {
		return models.GenericPage[models.UserPreview]{}, err
	}
	return models.NewPage(previews, pag, total), nil
}

func (s *Service) ListFollowingUserIDs(ctx context.Context, followerID string) ([]string, error) {
	return s.repo.ListFollowingUserIDs(ctx, followerID)
}
