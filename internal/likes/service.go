package likes

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/ids"
)

var ErrTargetNotFound = apierr.NewApiError(http.StatusNotFound, "target not found")

type counterUpdater interface {
	IncrementLikes(ctx context.Context, tx *sql.Tx, targetID string) (int, error)
	DecrementLikes(ctx context.Context, tx *sql.Tx, targetID string) (int, error)
}

type txBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type Service struct {
	db      txBeginner
	repo    *Repository
	counter counterUpdater
}

func NewService(db txBeginner, repo *Repository, counter counterUpdater) *Service {
	return &Service{
		db:      db,
		repo:    repo,
		counter: counter,
	}
}

type ToggleResult struct {
	Liked      bool
	LikesCount int
}

func (s *Service) IsLiked(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (bool, error) {
	return s.repo.Exists(ctx, userID, targetID, targetType)
}

func (s *Service) Toggle(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (ToggleResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ToggleResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	txRepo := s.repo.WithTx(tx)

	exists, err := txRepo.Exists(ctx, userID, targetID, targetType)
	if err != nil {
		return ToggleResult{}, fmt.Errorf("check like: %w", err)
	}

	like := Like{
		UserID:     userID,
		TargetID:   targetID,
		TargetType: targetType,
	}

	var likesCount int
	var liked bool
	if !exists {
		like.ID = ids.NewID()
		if err := txRepo.Create(ctx, like); err != nil {
			return ToggleResult{}, fmt.Errorf("create like: %w", err)
		}
		likesCount, err = s.counter.IncrementLikes(ctx, tx, targetID)
		if err != nil {
			return ToggleResult{}, fmt.Errorf("increment likes: %w", err)
		}
		liked = true
	} else {
		if err := txRepo.Delete(ctx, userID, targetID, targetType); err != nil {
			return ToggleResult{}, fmt.Errorf("delete like: %w", err)
		}
		likesCount, err = s.counter.DecrementLikes(ctx, tx, targetID)
		if err != nil {
			return ToggleResult{}, fmt.Errorf("decrement likes: %w", err)
		}
		liked = false
	}

	if err := tx.Commit(); err != nil {
		return ToggleResult{}, fmt.Errorf("commit tx: %w", err)
	}

	return ToggleResult{Liked: liked, LikesCount: likesCount}, nil
}

func (s *Service) GetLikedIDs(ctx context.Context, userID string, targetType models.LikeTargetType, targetIDs []string) (map[string]bool, error) {
	return s.repo.GetLikedIDs(ctx, userID, targetType, targetIDs)
}
