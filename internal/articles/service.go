package articles

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/events"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
	"github.com/nikpivkin/roasti-app-backend/internal/x/ptr"
)

type articleRepository interface {
	Create(ctx context.Context, article models.Article) error
	GetArticleByID(ctx context.Context, articleID string) (models.Article, error)
	GetArticlesByIDs(ctx context.Context, ids []string) ([]models.Article, error)
	UpdateArticle(ctx context.Context, articleID, title string, blocks []models.ArticleBlock) error
	DeleteArticle(ctx context.Context, articleID string) error
	ListArticles(ctx context.Context, params ListArticlesParams) ([]models.Article, int, error)
}

type commentService interface {
	Create(ctx context.Context, userID, targetID, targetType, text string, parentID *string) (models.Comment, error)
	List(ctx context.Context, targetID string, pag models.PaginationParams) (models.GenericPage[models.CommentThread], error)
}

type likeEnricher interface {
	EnrichOne(ctx context.Context, userID string, item likes.Likeable) error
	EnrichMany(ctx context.Context, userID string, items []likes.Likeable) error
}

type likeToggler interface {
	Toggle(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (likes.ToggleResult, error)
}

type uploader interface {
	Confirm(ctx context.Context, fileID string) error
}

type eventPublisher interface {
	Publish(e events.Event)
}

type ListArticlesParams struct {
	AuthorID  *string
	AuthorIDs []string // if set, filter to articles from these author IDs
	Limit     *int32
	Page      *int32
}

func (p ListArticlesParams) Pagination() models.PaginationParams {
	return models.NewPaginationParams(
		ptr.GetOr(p.Page, models.DefaultPage),
		ptr.GetOr(p.Limit, models.DefaultLimit),
	)
}

type Service struct {
	logger         *slog.Logger
	repo           articleRepository
	uploader       uploader
	likeEnricher   likeEnricher
	likeToggler    likeToggler
	publisher      eventPublisher
	commentService commentService
}

func NewService(repo articleRepository, uploader uploader, likeEnricher likeEnricher, likeToggler likeToggler, publisher eventPublisher, commentService commentService) *Service {
	return &Service{
		logger:         slog.Default(),
		repo:           repo,
		uploader:       uploader,
		likeEnricher:   likeEnricher,
		likeToggler:    likeToggler,
		publisher:      publisher,
		commentService: commentService,
	}
}

func (s *Service) CreateComment(ctx context.Context, userID, articleID, text string, parentID *string) (models.Comment, error) {
	article, err := s.repo.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Comment{}, ErrNotFound
		}
		return models.Comment{}, err
	}
	created, err := s.commentService.Create(ctx, userID, articleID, "article", text, parentID)
	if err != nil {
		return models.Comment{}, err
	}
	if s.publisher != nil {
		s.publisher.Publish(events.ArticleCommentCreated{
			ArticleID: articleID,
			OwnerID:   article.Author.Id,
			ByUserID:  userID,
			CommentID: created.Id,
		})
	}
	return created, nil
}

func (s *Service) ToggleLike(ctx context.Context, userID, articleID string) (likes.ToggleResult, error) {
	article, err := s.repo.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return likes.ToggleResult{}, ErrNotFound
		}
		return likes.ToggleResult{}, err
	}
	result, err := s.likeToggler.Toggle(ctx, userID, articleID, models.LikeTargetTypeArticle)
	if err != nil {
		return likes.ToggleResult{}, err
	}
	if s.publisher != nil {
		s.publisher.Publish(events.ArticleLikeToggled{
			ArticleID: articleID,
			OwnerID:   article.Author.Id,
			ByUserID:  userID,
			Liked:     result.Liked,
		})
	}
	return result, nil
}

func normalizeArticlePayload(req *models.ArticlePayload) {
	req.Title = strings.TrimSpace(req.Title)
}

func (s *Service) CreateArticle(ctx context.Context, userID string, req models.CreateArticleRequest) (models.Article, error) {
	normalizeArticlePayload(&req)
	if err := validateArticlePayload(req); err != nil {
		return models.Article{}, err
	}
	now := time.Now().UTC()
	article := models.Article{
		Id:        id.NewID(),
		Title:     req.Title,
		Blocks:    blockPayloadsToModel(req.Blocks),
		Author:    models.UserPreview{Id: userID},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, article); err != nil {
		return models.Article{}, err
	}

	created, err := s.repo.GetArticleByID(ctx, article.Id)
	if err != nil {
		return models.Article{}, err
	}
	s.confirmArticleImages(ctx, created)
	return created, nil
}

func (s *Service) ListArticles(ctx context.Context, userID string, params ListArticlesParams) (models.GenericPage[models.Article], error) {
	pag := params.Pagination()

	articleList, total, err := s.repo.ListArticles(ctx, params)
	if err != nil {
		return models.GenericPage[models.Article]{}, err
	}

	if len(articleList) == 0 {
		return models.NewPage(articleList, pag, 0), nil
	}

	likeables := make([]likes.Likeable, len(articleList))
	for i := range articleList {
		likeables[i] = &articleList[i]
	}
	if err := s.likeEnricher.EnrichMany(ctx, userID, likeables); err != nil {
		return models.GenericPage[models.Article]{}, err
	}

	return models.NewPage(articleList, pag, total), nil
}

func (s *Service) GetArticle(ctx context.Context, userID, articleID string) (models.Article, error) {
	article, err := s.repo.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Article{}, ErrNotFound
		}
		return models.Article{}, err
	}

	if err := s.likeEnricher.EnrichOne(ctx, userID, &article); err != nil {
		return models.Article{}, err
	}
	return article, nil
}

func (s *Service) UpdateArticle(ctx context.Context, userID, articleID string, req models.UpdateArticleRequest) (models.Article, error) {
	normalizeArticlePayload(&req)
	if err := validateArticlePayload(req); err != nil {
		return models.Article{}, err
	}
	article, err := s.repo.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Article{}, ErrNotFound
		}
		return models.Article{}, err
	}
	if article.Author.Id != userID {
		return models.Article{}, ErrForbidden
	}
	if err := s.repo.UpdateArticle(ctx, articleID, req.Title, blockPayloadsToModel(req.Blocks)); err != nil {
		return models.Article{}, err
	}
	updated, err := s.GetArticle(ctx, userID, articleID)
	if err != nil {
		return models.Article{}, err
	}
	s.confirmArticleImages(ctx, updated)
	return updated, nil
}

func (s *Service) DeleteArticle(ctx context.Context, userID, articleID string) error {
	article, err := s.repo.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if article.Author.Id != userID {
		return ErrForbidden
	}
	return s.repo.DeleteArticle(ctx, articleID)
}

func (s *Service) ListComments(ctx context.Context, articleID string, pag models.PaginationParams) (models.GenericPage[models.CommentThread], error) {
	_, err := s.repo.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GenericPage[models.CommentThread]{}, ErrNotFound
		}
		return models.GenericPage[models.CommentThread]{}, err
	}
	return s.commentService.List(ctx, articleID, pag)
}

func (s *Service) GetArticlesByIDs(ctx context.Context, currentUserID string, ids []string) ([]models.Article, error) {
	articleList, err := s.repo.GetArticlesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(articleList) == 0 {
		return articleList, nil
	}

	likeables := make([]likes.Likeable, len(articleList))
	for i := range articleList {
		likeables[i] = &articleList[i]
	}
	if err := s.likeEnricher.EnrichMany(ctx, currentUserID, likeables); err != nil {
		return nil, err
	}
	return articleList, nil
}

func (s *Service) confirmArticleImages(ctx context.Context, article models.Article) {
	for _, block := range article.Blocks {
		if block.Images == nil {
			continue
		}
		for _, imageID := range *block.Images {
			if err := s.uploader.Confirm(ctx, imageID); err != nil {
				s.logger.WarnContext(ctx, "failed to confirm article block image",
					slog.String("article_id", article.Id),
					slog.String("image_id", imageID),
				)
			}
		}
	}
}

func blockPayloadsToModel(payloads []models.ArticleBlockPayload) []models.ArticleBlock {
	blocks := make([]models.ArticleBlock, len(payloads))
	for i, p := range payloads {
		var recipe *models.ArticleRecipeRef
		if p.RecipeId != nil {
			recipe = &models.ArticleRecipeRef{Id: *p.RecipeId}
		}
		blocks[i] = models.ArticleBlock{
			Type:   p.Type,
			Images: p.Images,
			Text:   p.Text,
			Recipe: recipe,
		}
	}
	return blocks
}
