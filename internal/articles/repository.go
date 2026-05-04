package articles

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
	"github.com/nikpivkin/roasti-app-backend/internal/x/sqlutil"
)

const (
	articlesTable = "articles"
	blocksTable   = "article_blocks"
	// commentsTable and commentTargetType are used by getCommentsByArticleIDs.
	commentsTable     = "comments"
	commentTargetType = "article"
)

var articleselectColumns = []string{
	"articles.id",
	"articles.author_id",
	"articles.title",
	"articles.created_at",
	"articles.updated_at",
	"users.username",
	"users.name",
	"users.avatar_id",
}

var blockSelectColumns = []string{
	"article_blocks.id",
	"article_blocks.article_id",
	"article_blocks.block_order",
	"article_blocks.type",
	"article_blocks.images",
	"article_blocks.text",
	"article_blocks.recipe_id",
	"CASE WHEN article_blocks.recipe_id IS NULL THEN NULL WHEN recipes.id IS NOT NULL THEN 'available' ELSE 'unavailable' END",
}

var commentColumns = []string{
	"comments.id",
	"comments.target_id",
	"comments.parent_id",
	"comments.text",
	"comments.created_at",
	"comments.updated_at",
	"comments.deleted_at",
	"comments.author_id",
	"users.username",
	"users.name",
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

func (r *Repository) Create(ctx context.Context, article models.Article) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = r.psql.Insert(articlesTable).
		Columns("id", "author_id", "title", "created_at", "updated_at").
		Values(article.Id, article.Author.Id, article.Title, article.CreatedAt, article.UpdatedAt).
		RunWith(tx).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert article: %w", err)
	}

	if len(article.Blocks) > 0 {
		q := r.psql.Insert(blocksTable).
			Columns("id", "article_id", "block_order", "type", "images", "text", "recipe_id")
		for i, block := range article.Blocks {
			var imagesJSON *string
			if block.Images != nil && len(*block.Images) > 0 {
				b, err := json.Marshal(*block.Images)
				if err != nil {
					return fmt.Errorf("marshal block images: %w", err)
				}
				s := string(b)
				imagesJSON = &s
			}
			var recipeID *string
			if block.Recipe != nil {
				recipeID = &block.Recipe.Id
			}
			q = q.Values(id.NewID(), article.Id, i, block.Type, imagesJSON, block.Text, recipeID)
		}
		if _, err := q.RunWith(tx).ExecContext(ctx); err != nil {
			return fmt.Errorf("insert blocks: %w", err)
		}
	}

	return tx.Commit()
}

func (r *Repository) GetArticleByID(ctx context.Context, articleID string) (models.Article, error) {
	row := r.psql.
		Select(articleselectColumns...).
		From(articlesTable).
		Join("users ON users.id = articles.author_id").
		Where(sq.Eq{"articles.id": articleID}).
		Limit(1).
		RunWith(r.runner).
		QueryRowContext(ctx)

	article, err := scanArticle(row)
	if err != nil {
		return models.Article{}, err
	}

	if err := r.enrichArticles(ctx, []*models.Article{&article}); err != nil {
		return models.Article{}, err
	}
	return article, nil
}

func (r *Repository) ListArticles(ctx context.Context, params ListArticlesParams) ([]models.Article, int, error) {
	pag := params.Pagination()
	q := r.psql.
		Select(articleselectColumns...).
		From(articlesTable).
		Join("users ON users.id = articles.author_id").
		OrderBy("articles.created_at DESC, articles.id DESC").
		Limit(uint64(pag.GetLimit())).
		Offset(uint64(pag.Offset()))
	if params.AuthorID != nil {
		q = q.Where(sq.Eq{"articles.author_id": *params.AuthorID})
	} else if len(params.AuthorIDs) > 0 {
		q = q.Where(sq.Eq{"articles.author_id": params.AuthorIDs})
	}
	rows, err := q.RunWith(r.runner).QueryContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		p, err := scanArticle(rows)
		if err != nil {
			return nil, 0, err
		}
		articles = append(articles, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if len(articles) == 0 {
		return articles, 0, nil
	}

	ptrs := make([]*models.Article, len(articles))
	for i := range articles {
		ptrs[i] = &articles[i]
	}
	if err := r.enrichArticles(ctx, ptrs); err != nil {
		return nil, 0, err
	}

	var total int
	countQ := r.psql.Select("COUNT(*)").From(articlesTable)
	if params.AuthorID != nil {
		countQ = countQ.Where(sq.Eq{"author_id": *params.AuthorID})
	} else if len(params.AuthorIDs) > 0 {
		countQ = countQ.Where(sq.Eq{"author_id": params.AuthorIDs})
	}
	if err := countQ.RunWith(r.runner).QueryRowContext(ctx).Scan(&total); err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

func (r *Repository) UpdateArticle(ctx context.Context, articleID, title string, blocks []models.ArticleBlock) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = r.psql.Update(articlesTable).
		Set("title", title).
		Set("updated_at", sq.Expr("datetime('now')")).
		Where(sq.Eq{"id": articleID}).
		RunWith(tx).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("update article: %w", err)
	}

	if _, err := r.psql.Delete(blocksTable).
		Where(sq.Eq{"article_id": articleID}).
		RunWith(tx).
		ExecContext(ctx); err != nil {
		return fmt.Errorf("delete blocks: %w", err)
	}

	if len(blocks) > 0 {
		q := r.psql.Insert(blocksTable).
			Columns("id", "article_id", "block_order", "type", "images", "text", "recipe_id")
		for i, block := range blocks {
			var imagesJSON *string
			if block.Images != nil && len(*block.Images) > 0 {
				b, err := json.Marshal(*block.Images)
				if err != nil {
					return fmt.Errorf("marshal block images: %w", err)
				}
				s := string(b)
				imagesJSON = &s
			}
			var recipeID *string
			if block.Recipe != nil {
				recipeID = &block.Recipe.Id
			}
			q = q.Values(id.NewID(), articleID, i, block.Type, imagesJSON, block.Text, recipeID)
		}
		if _, err := q.RunWith(tx).ExecContext(ctx); err != nil {
			return fmt.Errorf("insert blocks: %w", err)
		}
	}

	return tx.Commit()
}

func (r *Repository) DeleteArticle(ctx context.Context, articleID string) error {
	_, err := r.psql.Delete(articlesTable).
		Where(sq.Eq{"id": articleID}).
		RunWith(r.runner).
		ExecContext(ctx)
	return err
}

func (r *Repository) GetArticlesByIDs(ctx context.Context, ids []string) ([]models.Article, error) {
	rows, err := r.psql.
		Select(articleselectColumns...).
		From(articlesTable).
		Join("users ON users.id = articles.author_id").
		Where(sq.Eq{"articles.id": ids}).
		OrderBy("articles.created_at DESC, articles.id DESC").
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articleList []models.Article
	for rows.Next() {
		p, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		articleList = append(articleList, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(articleList) == 0 {
		return articleList, nil
	}

	ptrs := make([]*models.Article, len(articleList))
	for i := range articleList {
		ptrs[i] = &articleList[i]
	}
	if err := r.enrichArticles(ctx, ptrs); err != nil {
		return nil, err
	}
	return articleList, nil
}

func (r *Repository) enrichArticles(ctx context.Context, articles []*models.Article) error {
	ids := make([]string, len(articles))
	index := make(map[string]*models.Article, len(articles))
	for i, p := range articles {
		ids[i] = p.Id
		index[p.Id] = p
	}

	blocksMap, err := r.getBlocksByArticleIDs(ctx, ids)
	if err != nil {
		return err
	}

	commentsMap, err := r.getCommentsByArticleIDs(ctx, ids)
	if err != nil {
		return err
	}

	for _, p := range articles {
		p.Blocks = blocksMap[p.Id]
		if p.Blocks == nil {
			p.Blocks = []models.ArticleBlock{}
		}
		p.Comments = commentsMap[p.Id]
		if p.Comments == nil {
			p.Comments = []models.Comment{}
		}
	}
	return nil
}

func (r *Repository) getBlocksByArticleIDs(ctx context.Context, articleIDs []string) (map[string][]models.ArticleBlock, error) {
	rows, err := r.psql.
		Select(blockSelectColumns...).
		From(blocksTable).
		LeftJoin("recipes ON recipes.id = article_blocks.recipe_id AND recipes.public = 1").
		Where(sq.Eq{"article_blocks.article_id": articleIDs}).
		OrderBy("article_blocks.block_order ASC").
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocksMap := make(map[string][]models.ArticleBlock)
	for rows.Next() {
		var (
			blockID, articleID string
			blockOrder         int
			block              models.ArticleBlock
			imagesJSON         sql.NullString
			recipeID           sql.NullString
			recipeStatus       sql.NullString
		)
		if err := rows.Scan(
			&blockID, &articleID, &blockOrder,
			&block.Type, &imagesJSON, &block.Text, &recipeID, &recipeStatus,
		); err != nil {
			return nil, err
		}
		if imagesJSON.Valid {
			if err := json.Unmarshal([]byte(imagesJSON.String), &block.Images); err != nil {
				return nil, fmt.Errorf("unmarshal block images: %w", err)
			}
		}
		if recipeID.Valid {
			block.Recipe = &models.ArticleRecipeRef{
				Id:     recipeID.String,
				Status: models.ArticleRecipeRefStatus(recipeStatus.String),
			}
		}
		blocksMap[articleID] = append(blocksMap[articleID], block)
	}
	return blocksMap, rows.Err()
}

func (r *Repository) getCommentsByArticleIDs(ctx context.Context, articleIDs []string) (map[string][]models.Comment, error) {
	rows, err := r.psql.
		Select(commentColumns...).
		From(commentsTable).
		LeftJoin("users ON users.id = comments.author_id").
		Where(sq.Eq{"target_id": articleIDs, "target_type": commentTargetType}).
		OrderBy("comments.created_at ASC").
		RunWith(r.runner).
		QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commentsMap := make(map[string][]models.Comment)
	for rows.Next() {
		var (
			targetID                           string
			comment                            models.Comment
			parentID, deletedAt                sql.NullString
			authorID, username, name, avatarID sql.NullString
		)
		if err := rows.Scan(
			&comment.Id, &targetID, &parentID, &comment.Text,
			&comment.CreatedAt, &comment.UpdatedAt, &deletedAt,
			&authorID, &username, &name, &avatarID,
		); err != nil {
			return nil, err
		}
		if parentID.Valid {
			comment.ParentId = &parentID.String
		}
		if deletedAt.Valid {
			comment.IsDeleted = true
			comment.Text = ""
		} else {
			author := sqlutil.BuildUserPreview(authorID.String, username.String, name, avatarID)
			comment.Author = &author
		}
		commentsMap[targetID] = append(commentsMap[targetID], comment)
	}
	return commentsMap, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanArticle(s scanner) (models.Article, error) {
	var (
		article        models.Article
		authorUsername string
		name, avatarID sql.NullString
	)
	err := s.Scan(
		&article.Id, &article.Author.Id, &article.Title,
		&article.CreatedAt, &article.UpdatedAt,
		&authorUsername, &name, &avatarID,
	)
	article.Author = sqlutil.BuildUserPreview(article.Author.Id, authorUsername, name, avatarID)
	return article, err
}
