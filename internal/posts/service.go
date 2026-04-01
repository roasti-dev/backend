package posts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
)

type PostRepository interface {
	Create(ctx context.Context, post models.Post) error
	GetPostByID(ctx context.Context, postID string) (models.Post, error)
	GetPostsByIDs(ctx context.Context, ids []string) ([]models.Post, error)
	UpdatePost(ctx context.Context, postID, title string, blocks []models.PostBlock) error
	DeletePost(ctx context.Context, postID string) error
	CreateComment(ctx context.Context, comment models.PostComment, postID string) (models.PostComment, error)
	GetCommentAuthorID(ctx context.Context, commentID string) (string, error)
	DeleteComment(ctx context.Context, commentID string) error
	ListPosts(ctx context.Context, pag models.PaginationParams) ([]models.Post, int, error)
}

type LikeChecker interface {
	GetLikedIDs(ctx context.Context, userID string, targetType models.LikeTargetType, targetIDs []string) (map[string]bool, error)
	CountByTargets(ctx context.Context, targetIDs []string, targetType models.LikeTargetType) (map[string]int, error)
}

type LikeToggler interface {
	Toggle(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (likes.ToggleResult, error)
}

type ListPostsParams struct {
	Limit *int32
	Page  *int32
}

func (p ListPostsParams) Pagination() models.PaginationParams {
	page := int32(models.DefaultPage)
	limit := int32(models.DefaultLimit)
	if p.Page != nil {
		page = *p.Page
	}
	if p.Limit != nil {
		limit = *p.Limit
	}
	return models.NewPaginationParams(page, limit)
}

type Service struct {
	repo         PostRepository
	likeChecker  LikeChecker
	likeToggler  LikeToggler
}

func NewService(repo PostRepository, likeChecker LikeChecker, likeToggler LikeToggler) *Service {
	return &Service{repo: repo, likeChecker: likeChecker, likeToggler: likeToggler}
}

func (s *Service) CreateComment(ctx context.Context, userID, postID, text string) (models.PostComment, error) {
	if _, err := s.repo.GetPostByID(ctx, postID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PostComment{}, ErrNotFound
		}
		return models.PostComment{}, err
	}
	comment := models.PostComment{
		Id:        id.NewID(),
		Author:    models.UserPreview{Id: userID},
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}
	return s.repo.CreateComment(ctx, comment, postID)
}

func (s *Service) ToggleLike(ctx context.Context, userID, postID string) (likes.ToggleResult, error) {
	if _, err := s.repo.GetPostByID(ctx, postID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return likes.ToggleResult{}, ErrNotFound
		}
		return likes.ToggleResult{}, err
	}
	return s.likeToggler.Toggle(ctx, userID, postID, models.LikeTargetTypePost)
}

func (s *Service) CreatePost(ctx context.Context, userID string, req models.CreatePostRequest) (models.Post, error) {
	now := time.Now().UTC()
	post := models.Post{
		Id:        id.NewID(),
		Title:     req.Title,
		Blocks:    blockPayloadsToModel(req.Blocks),
		Author:    models.UserPreview{Id: userID},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, post); err != nil {
		return models.Post{}, err
	}

	created, err := s.repo.GetPostByID(ctx, post.Id)
	if err != nil {
		return models.Post{}, err
	}
	return created, nil
}

func (s *Service) ListPosts(ctx context.Context, userID string, params ListPostsParams) (models.GenericPage[models.Post], error) {
	pag := params.Pagination()

	postList, total, err := s.repo.ListPosts(ctx, pag)
	if err != nil {
		return models.GenericPage[models.Post]{}, err
	}

	if len(postList) == 0 {
		return models.NewPage(postList, pag, 0), nil
	}

	ids := make([]string, len(postList))
	for i, p := range postList {
		ids[i] = p.Id
	}

	likedIDs, err := s.likeChecker.GetLikedIDs(ctx, userID, models.LikeTargetTypePost, ids)
	if err != nil {
		return models.GenericPage[models.Post]{}, fmt.Errorf("get liked ids: %w", err)
	}

	likesCounts, err := s.likeChecker.CountByTargets(ctx, ids, models.LikeTargetTypePost)
	if err != nil {
		return models.GenericPage[models.Post]{}, fmt.Errorf("count likes: %w", err)
	}

	for i, p := range postList {
		postList[i].IsLiked = likedIDs[p.Id]
		postList[i].LikesCount = int32(likesCounts[p.Id])
	}

	return models.NewPage(postList, pag, total), nil
}

func (s *Service) GetPost(ctx context.Context, userID, postID string) (models.Post, error) {
	post, err := s.repo.GetPostByID(ctx, postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Post{}, ErrNotFound
		}
		return models.Post{}, err
	}

	likedIDs, err := s.likeChecker.GetLikedIDs(ctx, userID, models.LikeTargetTypePost, []string{postID})
	if err != nil {
		return models.Post{}, fmt.Errorf("get liked ids: %w", err)
	}

	likesCounts, err := s.likeChecker.CountByTargets(ctx, []string{postID}, models.LikeTargetTypePost)
	if err != nil {
		return models.Post{}, fmt.Errorf("count likes: %w", err)
	}

	post.IsLiked = likedIDs[postID]
	post.LikesCount = int32(likesCounts[postID])
	return post, nil
}

func (s *Service) UpdatePost(ctx context.Context, userID, postID string, req models.UpdatePostRequest) (models.Post, error) {
	post, err := s.repo.GetPostByID(ctx, postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Post{}, ErrNotFound
		}
		return models.Post{}, err
	}
	if post.Author.Id != userID {
		return models.Post{}, ErrForbidden
	}
	if err := s.repo.UpdatePost(ctx, postID, req.Title, blockPayloadsToModel(req.Blocks)); err != nil {
		return models.Post{}, err
	}
	return s.GetPost(ctx, userID, postID)
}

func (s *Service) DeletePost(ctx context.Context, userID, postID string) error {
	post, err := s.repo.GetPostByID(ctx, postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if post.Author.Id != userID {
		return nil
	}
	return s.repo.DeletePost(ctx, postID)
}

func (s *Service) DeleteComment(ctx context.Context, userID, commentID string) error {
	authorID, err := s.repo.GetCommentAuthorID(ctx, commentID)
	if err != nil {
		return err
	}
	if authorID != userID {
		return ErrForbidden
	}
	return s.repo.DeleteComment(ctx, commentID)
}

func (s *Service) GetPostsByIDs(ctx context.Context, currentUserID string, ids []string) ([]models.Post, error) {
	postList, err := s.repo.GetPostsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(postList) == 0 {
		return postList, nil
	}

	postIDs := make([]string, len(postList))
	for i, p := range postList {
		postIDs[i] = p.Id
	}

	likedIDs, err := s.likeChecker.GetLikedIDs(ctx, currentUserID, models.LikeTargetTypePost, postIDs)
	if err != nil {
		return nil, fmt.Errorf("get liked ids: %w", err)
	}

	likesCounts, err := s.likeChecker.CountByTargets(ctx, postIDs, models.LikeTargetTypePost)
	if err != nil {
		return nil, fmt.Errorf("count likes: %w", err)
	}

	for i, p := range postList {
		postList[i].IsLiked = likedIDs[p.Id]
		postList[i].LikesCount = int32(likesCounts[p.Id])
	}
	return postList, nil
}

func blockPayloadsToModel(payloads []models.PostBlockPayload) []models.PostBlock {
	blocks := make([]models.PostBlock, len(payloads))
	for i, p := range payloads {
		var recipe *models.PostRecipeRef
		if p.RecipeId != nil {
			recipe = &models.PostRecipeRef{Id: *p.RecipeId}
		}
		blocks[i] = models.PostBlock{
			Type:   p.Type,
			Images: p.Images,
			Text:   p.Text,
			Recipe: recipe,
		}
	}
	return blocks
}
