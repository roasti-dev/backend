package beans

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
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

type Service struct {
	logger   *slog.Logger
	repo     repository
	uploader uploader
}

func NewService(logger *slog.Logger, repo repository, uploader uploader) *Service {
	return &Service{logger: logger, repo: repo, uploader: uploader}
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

func (s *Service) GetBean(ctx context.Context, beanID string) (models.Bean, error) {
	bean, err := s.repo.GetByID(ctx, beanID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Bean{}, ErrNotFound
		}
		return models.Bean{}, err
	}
	return bean, nil
}

func (s *Service) ListBeans(ctx context.Context, params ListBeansParams) (models.GenericPage[models.Bean], error) {
	pag := models.NewPaginationParams(
		ptr.FromPtr(params.Page),
		ptr.FromPtr(params.Limit),
	)
	items, total, err := s.repo.List(ctx, params)
	if err != nil {
		return models.GenericPage[models.Bean]{}, err
	}
	return models.NewPage(items, pag, total), nil
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
