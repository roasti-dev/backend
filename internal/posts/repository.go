package posts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
)

const (
	postsTable        = "posts"
	blocksTable       = "post_blocks"
	commentsTable     = "comments"
	commentTargetType = "post"
)

var postSelectColumns = []string{
	"posts.id",
	"posts.author_id",
	"posts.title",
	"posts.created_at",
	"posts.updated_at",
	"users.username",
	"users.avatar_id",
}

var blockColumns = []string{
	"id",
	"post_id",
	"block_order",
	"type",
	"images",
	"text",
	"recipe_id",
}

var commentColumns = []string{
	"comments.id",
	"comments.target_id",
	"comments.author_id",
	"comments.text",
	"comments.created_at",
	"users.username",
	"users.avatar_id",
}

type Repository struct {
	db     *sql.DB
	runner sq.StdSqlCtx
	psql   sq.StatementBuilderType
}

func NewRepository(db *sql.DB, runner sq.StdSqlCtx) *Repository {
	return &Repository{
		db:     db,
		runner: runner,
		psql:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(runner),
	}
}

func (r *Repository) Create(ctx context.Context, post models.Post) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = r.psql.Insert(postsTable).
		Columns("id", "author_id", "title", "created_at", "updated_at").
		Values(post.Id, post.Author.Id, post.Title, post.CreatedAt, post.UpdatedAt).
		RunWith(tx).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert post: %w", err)
	}

	if len(post.Blocks) > 0 {
		q := r.psql.Insert(blocksTable).
			Columns("id", "post_id", "block_order", "type", "images", "text", "recipe_id")
		for i, block := range post.Blocks {
			var imagesJSON *string
			if block.Images != nil && len(*block.Images) > 0 {
				b, err := json.Marshal(*block.Images)
				if err != nil {
					return fmt.Errorf("marshal block images: %w", err)
				}
				s := string(b)
				imagesJSON = &s
			}
			q = q.Values(id.NewID(), post.Id, i, block.Type, imagesJSON, block.Text, block.RecipeId)
		}
		if _, err := q.RunWith(tx).ExecContext(ctx); err != nil {
			return fmt.Errorf("insert blocks: %w", err)
		}
	}

	return tx.Commit()
}

func (r *Repository) GetPostByID(ctx context.Context, postID string) (models.Post, error) {
	row := r.psql.
		Select(postSelectColumns...).
		From(postsTable).
		Join("users ON users.id = posts.author_id").
		Where(sq.Eq{"posts.id": postID}).
		Limit(1).
		RunWith(r.runner).
		QueryRowContext(ctx)

	post, err := scanPost(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Post{}, ErrNotFound
		}
		return models.Post{}, err
	}

	if err := r.enrichPosts(ctx, []*models.Post{&post}); err != nil {
		return models.Post{}, err
	}
	return post, nil
}

func (r *Repository) ListPosts(ctx context.Context, pag models.PaginationParams) ([]models.Post, int, error) {
	rows, err := r.psql.
		Select(postSelectColumns...).
		From(postsTable).
		Join("users ON users.id = posts.author_id").
		OrderBy("posts.created_at DESC, posts.id DESC").
		Limit(uint64(pag.GetLimit())).
		Offset(uint64(pag.Offset())).
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if len(posts) == 0 {
		return posts, 0, nil
	}

	ptrs := make([]*models.Post, len(posts))
	for i := range posts {
		ptrs[i] = &posts[i]
	}
	if err := r.enrichPosts(ctx, ptrs); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.psql.Select("COUNT(*)").From(postsTable).
		RunWith(r.runner).QueryRowContext(ctx).Scan(&total); err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

func (r *Repository) DeletePost(ctx context.Context, postID string) error {
	_, err := r.psql.Delete(postsTable).
		Where(sq.Eq{"id": postID}).
		RunWith(r.runner).
		ExecContext(ctx)
	return err
}

func (r *Repository) GetPostsByIDs(ctx context.Context, ids []string) ([]models.Post, error) {
	rows, err := r.psql.
		Select(postSelectColumns...).
		From(postsTable).
		Join("users ON users.id = posts.author_id").
		Where(sq.Eq{"posts.id": ids}).
		OrderBy("posts.created_at DESC, posts.id DESC").
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var postList []models.Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		postList = append(postList, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(postList) == 0 {
		return postList, nil
	}

	ptrs := make([]*models.Post, len(postList))
	for i := range postList {
		ptrs[i] = &postList[i]
	}
	if err := r.enrichPosts(ctx, ptrs); err != nil {
		return nil, err
	}
	return postList, nil
}

func (r *Repository) enrichPosts(ctx context.Context, posts []*models.Post) error {
	ids := make([]string, len(posts))
	index := make(map[string]*models.Post, len(posts))
	for i, p := range posts {
		ids[i] = p.Id
		index[p.Id] = p
	}

	blocksMap, err := r.getBlocksByPostIDs(ctx, ids)
	if err != nil {
		return err
	}

	commentsMap, err := r.getCommentsByPostIDs(ctx, ids)
	if err != nil {
		return err
	}

	for _, p := range posts {
		p.Blocks = blocksMap[p.Id]
		if p.Blocks == nil {
			p.Blocks = []models.PostBlock{}
		}
		p.Comments = commentsMap[p.Id]
		if p.Comments == nil {
			p.Comments = []models.PostComment{}
		}
	}
	return nil
}

func (r *Repository) getBlocksByPostIDs(ctx context.Context, postIDs []string) (map[string][]models.PostBlock, error) {
	rows, err := r.psql.
		Select(blockColumns...).
		From(blocksTable).
		Where(sq.Eq{"post_id": postIDs}).
		OrderBy("block_order ASC").
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocksMap := make(map[string][]models.PostBlock)
	for rows.Next() {
		var (
			blockID, postID string
			blockOrder      int
			block           models.PostBlock
			imagesJSON      sql.NullString
		)
		if err := rows.Scan(
			&blockID, &postID, &blockOrder,
			&block.Type, &imagesJSON, &block.Text, &block.RecipeId,
		); err != nil {
			return nil, err
		}
		if imagesJSON.Valid {
			if err := json.Unmarshal([]byte(imagesJSON.String), &block.Images); err != nil {
				return nil, fmt.Errorf("unmarshal block images: %w", err)
			}
		}
		blocksMap[postID] = append(blocksMap[postID], block)
	}
	return blocksMap, rows.Err()
}

func (r *Repository) getCommentsByPostIDs(ctx context.Context, postIDs []string) (map[string][]models.PostComment, error) {
	rows, err := r.psql.
		Select(commentColumns...).
		From(commentsTable).
		Join("users ON users.id = comments.author_id").
		Where(sq.Eq{"target_id": postIDs, "target_type": commentTargetType}).
		OrderBy("comments.created_at ASC").
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commentsMap := make(map[string][]models.PostComment)
	for rows.Next() {
		var (
			targetID string
			comment  models.PostComment
			avatarID sql.NullString
		)
		if err := rows.Scan(
			&comment.Id, &targetID, &comment.Author.Id,
			&comment.Text, &comment.CreatedAt,
			&comment.Author.Username, &avatarID,
		); err != nil {
			return nil, err
		}
		if avatarID.Valid {
			comment.Author.AvatarId = &avatarID.String
		}
		commentsMap[targetID] = append(commentsMap[targetID], comment)
	}
	return commentsMap, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPost(s scanner) (models.Post, error) {
	var (
		post     models.Post
		avatarID sql.NullString
	)
	err := s.Scan(
		&post.Id, &post.Author.Id, &post.Title,
		&post.CreatedAt, &post.UpdatedAt,
		&post.Author.Username, &avatarID,
	)
	if avatarID.Valid {
		post.Author.AvatarId = &avatarID.String
	}
	return post, err
}
