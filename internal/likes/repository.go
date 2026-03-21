package likes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

const likesTable = "likes"

var likeColumns = []string{"id", "user_id", "target_id", "target_type", "created_at"}

type Repository struct {
	db   *sql.DB
	psql sq.StatementBuilderType
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db:   db,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(db),
	}
}

func (r *Repository) WithTx(tx *sql.Tx) *Repository {
	return &Repository{
		db:   r.db,
		psql: r.psql.RunWith(tx),
	}
}

func (r *Repository) Create(ctx context.Context, like Like) error {
	_, err := r.psql.Insert(likesTable).
		Columns(likeColumns...).
		Values(
			like.ID,
			like.UserID,
			like.TargetID,
			like.TargetType,
			time.Now().UTC(),
		).
		Suffix("ON CONFLICT (user_id, target_id, target_type) DO NOTHING").
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert like: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) error {
	_, err := r.psql.Delete(likesTable).
		Where(sq.Eq{
			"user_id":     userID,
			"target_id":   targetID,
			"target_type": targetType,
		}).
		ExecContext(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTargetNotFound
		}
		return fmt.Errorf("delete like: %w", err)
	}
	return nil
}

func (r *Repository) Exists(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (bool, error) {
	var exists bool
	err := r.psql.Select("COUNT(*) > 0").
		From(likesTable).
		Where(sq.Eq{
			"user_id":     userID,
			"target_id":   targetID,
			"target_type": targetType,
		}).
		QueryRowContext(ctx).
		Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check like exists: %w", err)
	}
	return exists, nil
}

func (r *Repository) GetLikedIDs(ctx context.Context, userID string, targetType models.LikeTargetType, targetIDs []string) (map[string]bool, error) {
	rows, err := r.psql.Select("target_id").
		From(likesTable).
		Where(sq.Eq{
			"user_id":     userID,
			"target_type": targetType,
			"target_id":   targetIDs,
		}).
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get liked ids: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool, len(targetIDs))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan liked id: %w", err)
		}
		result[id] = true
	}
	return result, nil
}
