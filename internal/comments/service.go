package comments

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
)

const textMaxLen = 1000

type Repository interface {
	Create(ctx context.Context, comment models.PostComment, targetID, targetType string) error
	GetByID(ctx context.Context, commentID string) (models.PostComment, error)
	Update(ctx context.Context, commentID, text string) error
	GetAuthorID(ctx context.Context, commentID string) (string, error)
	Delete(ctx context.Context, commentID string) error
	ExistsInTarget(ctx context.Context, commentID, targetID string) (bool, error)
	ListForTarget(ctx context.Context, targetID string, pag models.PaginationParams) ([]models.CommentThread, int, error)
}

type Service struct {
	logger *slog.Logger
	repo   Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		logger: slog.Default(),
		repo:   repo,
	}
}

func (s *Service) Create(ctx context.Context, userID, targetID, targetType, text string, parentID *string) (models.PostComment, error) {
	text = normalizeComment(text)
	if err := validateComment(text); err != nil {
		return models.PostComment{}, err
	}
	if parentID != nil {
		exists, err := s.repo.ExistsInTarget(ctx, *parentID, targetID)
		if err != nil {
			return models.PostComment{}, err
		}
		if !exists {
			return models.PostComment{}, ErrNotFound
		}
	}

	author := models.UserPreview{Id: userID}
	now := time.Now().UTC()
	comment := models.PostComment{
		Id:        id.NewID(),
		Author:    &author,
		Text:      text,
		ParentId:  parentID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, comment, targetID, targetType); err != nil {
		return models.PostComment{}, err
	}
	return s.repo.GetByID(ctx, comment.Id)
}

func (s *Service) Update(ctx context.Context, userID, commentID, text string) (models.PostComment, error) {
	authorID, err := s.repo.GetAuthorID(ctx, commentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PostComment{}, ErrNotFound
		}
		return models.PostComment{}, err
	}
	if authorID != userID {
		return models.PostComment{}, ErrForbidden
	}
	text = normalizeComment(text)
	if err := validateComment(text); err != nil {
		return models.PostComment{}, err
	}
	if err := s.repo.Update(ctx, commentID, text); err != nil {
		return models.PostComment{}, err
	}
	return s.repo.GetByID(ctx, commentID)
}

func (s *Service) List(ctx context.Context, targetID string, pag models.PaginationParams) (models.GenericPage[models.CommentThread], error) {
	items, total, err := s.repo.ListForTarget(ctx, targetID, pag)
	if err != nil {
		return models.GenericPage[models.CommentThread]{}, err
	}
	return models.NewPage(items, pag, total), nil
}

func (s *Service) Delete(ctx context.Context, userID, commentID string) error {
	authorID, err := s.repo.GetAuthorID(ctx, commentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if authorID != userID {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, commentID)
}
