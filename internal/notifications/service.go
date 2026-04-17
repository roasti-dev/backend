package notifications

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/events"
	"github.com/nikpivkin/roasti-app-backend/internal/log"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
)

type repository interface {
	Create(ctx context.Context, n Notification) error
	List(ctx context.Context, userID string, pag models.PaginationParams) ([]models.Notification, int, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkAllRead(ctx context.Context, userID string) error
}

type Service struct {
	logger *slog.Logger
	repo   repository
}

func NewService(repo repository) *Service {
	return &Service{
		logger: slog.Default(),
		repo:   repo,
	}
}

func (s *Service) HandleEvent(e events.Event) {
	ctx := context.Background()

	var n Notification
	switch ev := e.(type) {
	case events.RecipeLikeToggled:
		if !ev.Liked || ev.OwnerID == ev.ByUserID {
			return
		}
		n = Notification{ID: id.NewID(), UserID: ev.OwnerID, Type: TypeLikeRecipe, ActorID: ev.ByUserID, EntityID: ev.RecipeID}
	case events.PostLikeToggled:
		if !ev.Liked || ev.OwnerID == ev.ByUserID {
			return
		}
		n = Notification{ID: id.NewID(), UserID: ev.OwnerID, Type: TypeLikePost, ActorID: ev.ByUserID, EntityID: ev.PostID}
	case events.BeanLikeToggled:
		if !ev.Liked || ev.OwnerID == ev.ByUserID {
			return
		}
		n = Notification{ID: id.NewID(), UserID: ev.OwnerID, Type: TypeLikeBean, ActorID: ev.ByUserID, EntityID: ev.BeanID}
	case events.RecipeCommentCreated:
		if ev.OwnerID == ev.ByUserID {
			return
		}
		n = Notification{ID: id.NewID(), UserID: ev.OwnerID, Type: TypeCommentRecipe, ActorID: ev.ByUserID, EntityID: ev.RecipeID}
	case events.PostCommentCreated:
		if ev.OwnerID == ev.ByUserID {
			return
		}
		n = Notification{ID: id.NewID(), UserID: ev.OwnerID, Type: TypeCommentPost, ActorID: ev.ByUserID, EntityID: ev.PostID}
	case events.BeanCommentCreated:
		if ev.OwnerID == ev.ByUserID {
			return
		}
		n = Notification{ID: id.NewID(), UserID: ev.OwnerID, Type: TypeCommentBean, ActorID: ev.ByUserID, EntityID: ev.BeanID}
	case events.UserFollowed:
		n = Notification{ID: id.NewID(), UserID: ev.TargetID, Type: TypeFollow, ActorID: ev.FollowerID, EntityID: ev.FollowerID}
	default:
		return
	}

	if err := s.repo.Create(ctx, n); err != nil {
		s.logger.ErrorContext(ctx, "create notification", log.Err(err))
	}
}

func (s *Service) ListNotifications(ctx context.Context, userID string, pag models.PaginationParams) (models.GenericPage[models.Notification], error) {
	items, total, err := s.repo.List(ctx, userID, pag)
	if err != nil {
		return models.GenericPage[models.Notification]{}, fmt.Errorf("list notifications: %w", err)
	}
	return models.NewPage(items, pag, total), nil
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
