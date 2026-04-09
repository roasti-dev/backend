package beans

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/events"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
	"github.com/nikpivkin/roasti-app-backend/internal/x/ptr"
)

type repository interface {
	Create(ctx context.Context, beanID, authorID string, req models.BeanPayload) error
	GetByID(ctx context.Context, beanID string) (models.Bean, error)
	List(ctx context.Context, params ListBeansParams) ([]models.Bean, int, error)
	Update(ctx context.Context, beanID string, req models.BeanPayload) error
	SoftDelete(ctx context.Context, beanID string) error
}

type uploader interface {
	Confirm(ctx context.Context, fileID string) error
}

type likeEnricher interface {
	EnrichOne(ctx context.Context, userID string, item likes.Likeable) error
	EnrichMany(ctx context.Context, userID string, items []likes.Likeable) error
}

type likeToggler interface {
	Toggle(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (likes.ToggleResult, error)
}

type commentService interface {
	Create(ctx context.Context, userID, targetID, targetType, text string, parentID *string) (models.PostComment, error)
	List(ctx context.Context, targetID string, pag models.PaginationParams) (models.GenericPage[models.CommentThread], error)
}

type eventPublisher interface {
	Publish(e events.Event)
}

type Service struct {
	logger         *slog.Logger
	repo           repository
	uploader       uploader
	likeEnricher   likeEnricher
	likeToggler    likeToggler
	publisher      eventPublisher
	commentService commentService
}

func NewService(logger *slog.Logger, repo repository, uploader uploader, enricher likeEnricher, likeToggler likeToggler, publisher eventPublisher, commentService commentService) *Service {
	return &Service{logger: logger, repo: repo, uploader: uploader, likeEnricher: enricher, likeToggler: likeToggler, publisher: publisher, commentService: commentService}
}

func (s *Service) CreateBean(ctx context.Context, userID string, req models.CreateBeanRequest) (models.Bean, error) {
	beanID := id.NewID()
	if err := s.repo.Create(ctx, beanID, userID, req); err != nil {
		return models.Bean{}, err
	}

	created, err := s.repo.GetByID(ctx, beanID)
	if err != nil {
		return models.Bean{}, err
	}

	s.confirmBeanImage(ctx, beanID, req.ImageId)
	return created, nil
}

func (s *Service) GetBean(ctx context.Context, userID, beanID string) (models.Bean, error) {
	bean, err := s.repo.GetByID(ctx, beanID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Bean{}, ErrNotFound
		}
		return models.Bean{}, err
	}

	if err := s.likeEnricher.EnrichOne(ctx, userID, &bean); err != nil {
		return models.Bean{}, err
	}

	return bean, nil
}

func (s *Service) ListBeans(ctx context.Context, userID string, params ListBeansParams) (models.GenericPage[models.Bean], error) {
	pag := models.NewPaginationParams(
		ptr.FromPtr(params.Page),
		ptr.FromPtr(params.Limit),
	)
	items, total, err := s.repo.List(ctx, params)
	if err != nil {
		return models.GenericPage[models.Bean]{}, err
	}

	if len(items) > 0 {
		likeables := make([]likes.Likeable, len(items))
		for i := range items {
			likeables[i] = &items[i]
		}
		if err := s.likeEnricher.EnrichMany(ctx, userID, likeables); err != nil {
			return models.GenericPage[models.Bean]{}, err
		}
	}

	return models.NewPage(items, pag, total), nil
}

func (s *Service) ToggleLike(ctx context.Context, userID, beanID string) (likes.ToggleResult, error) {
	bean, err := s.GetBean(ctx, userID, beanID)
	if err != nil {
		return likes.ToggleResult{}, err
	}

	result, err := s.likeToggler.Toggle(ctx, userID, beanID, models.LikeTargetTypeBean)
	if err != nil {
		return likes.ToggleResult{}, err
	}

	if s.publisher != nil {
		s.publisher.Publish(events.BeanLikeToggled{
			BeanID:   beanID,
			OwnerID:  bean.Author.Id,
			ByUserID: userID,
			Liked:    result.Liked,
		})
	}

	return result, nil
}

func (s *Service) UpdateBean(ctx context.Context, userID, beanID string, req models.UpdateBeanRequest) (models.Bean, error) {
	existing, err := s.repo.GetByID(ctx, beanID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Bean{}, ErrNotFound
		}
		return models.Bean{}, err
	}
	if existing.Author.Id != userID {
		return models.Bean{}, ErrForbidden
	}

	if err := s.repo.Update(ctx, beanID, req); err != nil {
		return models.Bean{}, err
	}

	updated, err := s.repo.GetByID(ctx, beanID)
	if err != nil {
		return models.Bean{}, err
	}

	s.confirmBeanImage(ctx, beanID, req.ImageId)
	return updated, nil
}

func (s *Service) DeleteBean(ctx context.Context, userID, beanID string) error {
	existing, err := s.repo.GetByID(ctx, beanID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // idempotent
		}
		return err
	}
	if existing.Author.Id != userID {
		return ErrForbidden
	}
	return s.repo.SoftDelete(ctx, beanID)
}

func (s *Service) CreateComment(ctx context.Context, userID, beanID, text string, parentID *string) (models.PostComment, error) {
	bean, err := s.repo.GetByID(ctx, beanID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PostComment{}, ErrNotFound
		}
		return models.PostComment{}, err
	}
	created, err := s.commentService.Create(ctx, userID, beanID, "bean", text, parentID)
	if err != nil {
		return models.PostComment{}, err
	}
	if s.publisher != nil {
		s.publisher.Publish(events.BeanCommentCreated{
			BeanID:    beanID,
			OwnerID:   bean.Author.Id,
			ByUserID:  userID,
			CommentID: created.Id,
		})
	}
	return created, nil
}

func (s *Service) ListComments(ctx context.Context, beanID string, pag models.PaginationParams) (models.GenericPage[models.CommentThread], error) {
	if _, err := s.repo.GetByID(ctx, beanID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GenericPage[models.CommentThread]{}, ErrNotFound
		}
		return models.GenericPage[models.CommentThread]{}, err
	}
	return s.commentService.List(ctx, beanID, pag)
}

func (s *Service) confirmBeanImage(ctx context.Context, beanID string, imageID *string) {
	if imageID == nil {
		return
	}
	if err := s.uploader.Confirm(ctx, *imageID); err != nil {
		s.logger.WarnContext(ctx, "failed to confirm bean image",
			slog.String("bean_id", beanID),
			slog.String("image_id", *imageID),
		)
	}
}
