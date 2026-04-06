package notifications

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

const notificationsTable = "notifications"

type Repository struct {
	runner sq.StdSqlCtx
	psql   sq.StatementBuilderType
}

func NewRepository(runner sq.StdSqlCtx) *Repository {
	return &Repository{
		runner: runner,
		psql:   sq.StatementBuilder.PlaceholderFormat(sq.Question),
	}
}

func (r *Repository) Create(ctx context.Context, n Notification) error {
	_, err := r.psql.Insert(notificationsTable).
		Columns("id", "user_id", "type", "actor_id", "entity_id", "created_at").
		Values(n.ID, n.UserID, n.Type, n.ActorID, n.EntityID, time.Now().UTC()).
		RunWith(r.runner).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}

func (r *Repository) List(ctx context.Context, userID string, pag models.PaginationParams) ([]models.Notification, int, error) {
	var total int
	err := r.psql.Select("COUNT(*)").
		From(notificationsTable).
		Where(sq.Eq{"user_id": userID}).
		RunWith(r.runner).
		QueryRowContext(ctx).
		Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	rows, err := r.psql.
		Select(
			"n.id", "n.type", "n.entity_id", "n.read_at", "n.created_at",
			"u.id", "u.username", "u.avatar_id",
		).
		From(notificationsTable + " n").
		Join("users u ON u.id = n.actor_id").
		Where(sq.Eq{"n.user_id": userID}).
		OrderBy("n.created_at DESC").
		Limit(uint64(pag.GetLimit())).
		Offset(uint64(pag.Offset())).
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var result []models.Notification
	for rows.Next() {
		var n models.Notification
		var actor models.UserPreview
		var readAt sql.NullTime
		var avatarID sql.NullString
		var notifType string

		if err := rows.Scan(
			&n.Id, &notifType, &n.EntityId, &readAt, &n.CreatedAt,
			&actor.Id, &actor.Username, &avatarID,
		); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}

		n.Type = models.NotificationType(notifType)
		if readAt.Valid {
			n.ReadAt = &readAt.Time
		}
		if avatarID.Valid {
			actor.AvatarId = &avatarID.String
		}
		n.Actor = actor
		result = append(result, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}
	return result, total, nil
}

func (r *Repository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.psql.Select("COUNT(*)").
		From(notificationsTable).
		Where(sq.Eq{"user_id": userID}).
		Where("read_at IS NULL").
		RunWith(r.runner).
		QueryRowContext(ctx).
		Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.psql.Update(notificationsTable).
		Set("read_at", sq.Expr("datetime('now')")).
		Where(sq.Eq{"user_id": userID}).
		Where("read_at IS NULL").
		RunWith(r.runner).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}
