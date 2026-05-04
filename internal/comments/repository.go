package comments

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/x/sqlutil"
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

func (r *Repo) Create(ctx context.Context, comment models.Comment, targetID, targetType string) error {
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
		return fmt.Errorf("insert comment: %w", err)
	}
	return nil
}

func (r *Repo) GetByID(ctx context.Context, commentID string) (models.Comment, error) {
	var (
		avatarID        sql.NullString
		scannedParentID sql.NullString
		comment         models.Comment
		author          models.UserPreview
	)
	comment.Author = &author
	row := r.psql.
		Select("comments.id", "comments.text", "comments.parent_id", "comments.created_at", "comments.updated_at", "users.id", "users.username", "users.avatar_id").
		From(commentsTable).
		Join("users ON users.id = comments.author_id").
		Where(sq.Eq{"comments.id": commentID}).
		Where("comments.deleted_at IS NULL").
		Limit(1).
		RunWith(r.runner).
		QueryRowContext(ctx)
	err := row.Scan(
		&comment.Id, &comment.Text, &scannedParentID, &comment.CreatedAt, &comment.UpdatedAt,
		&author.Id, &author.Username, &avatarID,
	)
	if err != nil {
		return models.Comment{}, fmt.Errorf("fetch comment: %w", err)
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

func (r *Repo) Update(ctx context.Context, commentID, text string) error {
	result, err := r.psql.Update(commentsTable).
		Set("text", text).
		Set("updated_at", sq.Expr("datetime('now')")).
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

func (r *Repo) ListForTarget(ctx context.Context, targetID string, pag models.PaginationParams) ([]models.CommentThread, int, error) {
	var total int
	err := r.psql.Select("COUNT(*)").
		From(commentsTable).
		Where(sq.Eq{"target_id": targetID}).
		Where("parent_id IS NULL").
		RunWith(r.runner).
		QueryRowContext(ctx).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []models.CommentThread{}, 0, nil
	}

	rootRows, err := r.psql.
		Select(
			"comments.id", "comments.text", "comments.created_at", "comments.updated_at", "comments.deleted_at",
			"users.id", "users.username", "users.name", "users.avatar_id").
		From(commentsTable).
		LeftJoin("users ON users.id = comments.author_id").
		Where(sq.Eq{"target_id": targetID}).
		Where("comments.parent_id IS NULL").
		OrderBy("comments.created_at ASC").
		Limit(uint64(pag.GetLimit())).
		Offset(uint64(pag.Offset())).
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer rootRows.Close()

	var roots []models.CommentThread
	var rootIDs []string
	for rootRows.Next() {
		var (
			c                                                    models.CommentThread
			deletedAt                                            sql.NullString
			authorID, authorUsername, authorName, authorAvatarID sql.NullString
		)
		if err := rootRows.Scan(&c.Id, &c.Text, &c.CreatedAt, &c.UpdatedAt, &deletedAt, &authorID, &authorUsername, &authorName, &authorAvatarID); err != nil {
			return nil, 0, err
		}
		if deletedAt.Valid {
			c.IsDeleted = true
			c.Text = ""
		} else {
			author := sqlutil.BuildUserPreview(authorID.String, authorUsername.String, authorName, authorAvatarID)
			c.Author = &author
		}
		c.Replies = []models.Comment{}
		roots = append(roots, c)
		rootIDs = append(rootIDs, c.Id)
	}
	if err := rootRows.Err(); err != nil {
		return nil, 0, err
	}

	// rootIdxMap maps root comment ID → index in roots slice
	rootIdxMap := make(map[string]int, len(roots))
	for i, r := range roots {
		rootIdxMap[r.Id] = i
	}

	placeholders := strings.Repeat("?,", len(rootIDs))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(rootIDs))
	for i, id := range rootIDs {
		args[i] = id
	}
	//nolint:gosec
	replyQuery := fmt.Sprintf(`
		WITH RECURSIVE descendants(id) AS (
			SELECT id FROM comments WHERE parent_id IN (%s)
			UNION ALL
			SELECT c.id FROM comments c
			JOIN descendants d ON c.parent_id = d.id
		)
		SELECT comments.id, comments.text, comments.parent_id, comments.created_at, comments.updated_at, comments.deleted_at,
		       users.id, users.username, users.name, users.avatar_id
		FROM comments
		LEFT JOIN users ON users.id = comments.author_id
		WHERE comments.id IN (SELECT id FROM descendants)
		ORDER BY comments.created_at ASC
	`, placeholders)

	replyRows, err := r.runner.QueryContext(ctx, replyQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer replyRows.Close()

	// commentToRoot maps any descendant comment ID → index in roots slice
	commentToRoot := make(map[string]int)

	for replyRows.Next() {
		var (
			c                                                    models.Comment
			parentID, deletedAt                                  sql.NullString
			authorID, authorUsername, authorName, authorAvatarID sql.NullString
		)
		if err := replyRows.Scan(&c.Id, &c.Text, &parentID, &c.CreatedAt, &c.UpdatedAt, &deletedAt, &authorID, &authorUsername, &authorName, &authorAvatarID); err != nil {
			return nil, 0, err
		}
		if deletedAt.Valid {
			c.IsDeleted = true
			c.Text = ""
		} else {
			author := sqlutil.BuildUserPreview(authorID.String, authorUsername.String, authorName, authorAvatarID)
			c.Author = &author
		}
		if !parentID.Valid {
			continue
		}
		c.ParentId = &parentID.String

		// find the root: parent is either a root comment or a known descendant
		rootIdx, ok := rootIdxMap[*c.ParentId]
		if !ok {
			rootIdx, ok = commentToRoot[*c.ParentId]
		}
		if !ok {
			continue
		}
		commentToRoot[c.Id] = rootIdx
		roots[rootIdx].Replies = append(roots[rootIdx].Replies, c)
	}
	if err := replyRows.Err(); err != nil {
		return nil, 0, err
	}

	return roots, total, nil
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
