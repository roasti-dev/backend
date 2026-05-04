package follows

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

const followsTable = "follows"

var followColumns = []string{"id", "follower_id", "target_id", "target_type", "created_at"}

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

func (r *Repository) Create(ctx context.Context, f Follow) error {
	_, err := r.psql.Insert(followsTable).
		Columns(followColumns...).
		Values(f.ID, f.FollowerID, f.TargetID, f.TargetType, time.Now().UTC()).
		Suffix("ON CONFLICT (follower_id, target_id, target_type) DO NOTHING").
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert follow: %w", err)
	}
	return nil
}

func (r *Repository) Exists(ctx context.Context, followerID, targetID, targetType string) (bool, error) {
	var count int
	err := r.psql.Select("COUNT(1)").
		From(followsTable).
		Where(sq.Eq{"follower_id": followerID, "target_id": targetID, "target_type": targetType}).
		QueryRowContext(ctx).
		Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check follow exists: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) Delete(ctx context.Context, followerID, targetID, targetType string) error {
	_, err := r.psql.Delete(followsTable).
		Where(sq.Eq{"follower_id": followerID, "target_id": targetID, "target_type": targetType}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete follow: %w", err)
	}
	return nil
}

func (r *Repository) ListFollowing(ctx context.Context, followerID, targetType string, limit, offset int) ([]string, int, error) {
	q := `
SELECT target_id, COUNT(*) OVER() AS total
FROM follows
WHERE follower_id = ? AND target_type = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, followerID, targetType, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list following: %w", err)
	}
	defer rows.Close()
	var ids []string
	var total int
	for rows.Next() {
		var id string
		if err := rows.Scan(&id, &total); err != nil {
			return nil, 0, fmt.Errorf("scan following: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, total, rows.Err()
}

func (r *Repository) ListFollowers(ctx context.Context, targetID, targetType string, limit, offset int) ([]string, int, error) {
	q := `
SELECT follower_id, COUNT(*) OVER() AS total
FROM follows
WHERE target_id = ? AND target_type = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, targetID, targetType, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list followers: %w", err)
	}
	defer rows.Close()
	var ids []string
	var total int
	for rows.Next() {
		var id string
		if err := rows.Scan(&id, &total); err != nil {
			return nil, 0, fmt.Errorf("scan followers: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, total, rows.Err()
}

type FollowStats struct {
	FollowersCount int
	FollowingCount int
	IsFollowing    bool
	IsFollowed     bool
}

func (r *Repository) GetStats(ctx context.Context, targetUserID, currentUserID string) (FollowStats, error) {
	q := `
SELECT
    (SELECT COUNT(1) FROM follows WHERE target_id = ? AND target_type = 'user') AS followers_count,
    (SELECT COUNT(1) FROM follows WHERE follower_id = ? AND target_type = 'user') AS following_count,
    (SELECT COUNT(1) FROM follows WHERE follower_id = ? AND target_id = ? AND target_type = 'user') AS is_following,
    (SELECT COUNT(1) FROM follows WHERE follower_id = ? AND target_id = ? AND target_type = 'user') AS is_followed`
	var stats FollowStats
	var isFollowing, isFollowed int
	err := r.db.QueryRowContext(ctx, q,
		targetUserID,
		targetUserID,
		currentUserID, targetUserID,
		targetUserID, currentUserID,
	).Scan(&stats.FollowersCount, &stats.FollowingCount, &isFollowing, &isFollowed)
	if err != nil {
		return FollowStats{}, fmt.Errorf("get follow stats: %w", err)
	}
	stats.IsFollowing = isFollowing > 0
	stats.IsFollowed = isFollowed > 0
	return stats, nil
}

type ListFollowingPostsParams struct {
	FollowerID string
	Limit      int
	Offset     int
}

func (r *Repository) ListFollowingUserIDs(ctx context.Context, followerID string) ([]string, error) {
	rows, err := r.psql.Select("target_id").
		From(followsTable).
		Where(sq.Eq{"follower_id": followerID, "target_type": TargetTypeUser}).
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("list following user ids: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan following user id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type ListFollowingParamsRepo struct {
	Page  *int32
	Limit *int32
}

func (r *Repository) GetFollowingPagination(params models.PaginationParams) (int, int) {
	return int(params.GetLimit()), int(params.Offset())
}
