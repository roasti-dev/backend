package comments

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

const commentsTable = "comments"

type Repo struct {
	runner sq.StdSqlCtx
	psql   sq.StatementBuilderType
}

func NewRepository(runner sq.StdSqlCtx) *Repo {
	return &Repo{
		runner: runner,
		psql:   sq.StatementBuilder.PlaceholderFormat(sq.Question),
	}
}

func (r *Repo) Create(ctx context.Context, comment models.PostComment, targetID, targetType string) (models.PostComment, error) {
	var parentID interface{}
	if comment.ParentId != nil {
		parentID = *comment.ParentId
	}
	_, err := r.psql.Insert(commentsTable).
		Columns("id", "target_id", "target_type", "author_id", "text", "parent_id", "created_at", "updated_at").
		Values(comment.Id, targetID, targetType, comment.Author.Id, comment.Text, parentID, comment.CreatedAt, comment.CreatedAt).
		RunWith(r.runner).
		ExecContext(ctx)
	if err != nil {
		return models.PostComment{}, fmt.Errorf("insert comment: %w", err)
	}

	row := r.psql.
		Select("comments.id", "comments.text", "comments.parent_id", "comments.created_at", "comments.updated_at", "users.id", "users.username", "users.avatar_id").
		From(commentsTable).
		Join("users ON users.id = comments.author_id").
		Where(sq.Eq{"comments.id": comment.Id}).
		Limit(1).
		RunWith(r.runner).
		QueryRowContext(ctx)

	var (
		avatarID        sql.NullString
		scannedParentID sql.NullString
	)
	err = row.Scan(
		&comment.Id, &comment.Text, &scannedParentID, &comment.CreatedAt, &comment.UpdatedAt,
		&comment.Author.Id, &comment.Author.Username, &avatarID,
	)
	if err != nil {
		return models.PostComment{}, fmt.Errorf("fetch comment: %w", err)
	}
	if avatarID.Valid {
		comment.Author.AvatarId = &avatarID.String
	}
	if scannedParentID.Valid {
		comment.ParentId = &scannedParentID.String
	}
	return comment, nil
}

func (r *Repo) GetAuthorID(ctx context.Context, commentID string) (string, error) {
	var authorID string
	err := r.psql.Select("author_id").
		From(commentsTable).
		Where(sq.Eq{"id": commentID}).
		Where("deleted_at IS NULL").
		Limit(1).
		RunWith(r.runner).
		QueryRowContext(ctx).Scan(&authorID)
	return authorID, err
}

func (r *Repo) Delete(ctx context.Context, commentID string) error {
	result, err := r.psql.Update(commentsTable).
		Set("deleted_at", sq.Expr("datetime('now')")).
		Where(sq.Eq{"id": commentID}).
		Where("deleted_at IS NULL").
		RunWith(r.runner).
		ExecContext(ctx)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) ExistsInTarget(ctx context.Context, commentID, targetID string) (bool, error) {
	var exists bool
	err := r.psql.Select("COUNT(*) > 0").
		From(commentsTable).
		Where(sq.Eq{"id": commentID, "target_id": targetID}).
		Where("deleted_at IS NULL").
		RunWith(r.runner).
		QueryRowContext(ctx).Scan(&exists)
	return exists, err
}
