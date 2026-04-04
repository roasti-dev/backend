package comments

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
)

const textMaxLen = 1000

type Repository interface {
	Create(ctx context.Context, comment models.PostComment, targetID, targetType string) (models.PostComment, error)
	GetAuthorID(ctx context.Context, commentID string) (string, error)
	Delete(ctx context.Context, commentID string) error
	ExistsInTarget(ctx context.Context, commentID, targetID string) (bool, error)
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
	text = strings.TrimSpace(text)
	if text == "" {
		return models.PostComment{}, ErrInvalidText
	}
	if len(text) > textMaxLen {
		return models.PostComment{}, ErrTextTooLong
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
	comment := models.PostComment{
		Id:        id.NewID(),
		Author:    &author,
		Text:      text,
		ParentId:  parentID,
		CreatedAt: time.Now().UTC(),
	}
	return s.repo.Create(ctx, comment, targetID, targetType)
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
