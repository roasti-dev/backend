package posts

import (
	"context"
	"fmt"
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
)

type PostRepository interface {
	Create(ctx context.Context, post models.Post) error
	GetPostByID(ctx context.Context, postID string) (models.Post, error)
	GetPostsByIDs(ctx context.Context, ids []string) ([]models.Post, error)
	DeletePost(ctx context.Context, postID string) error
	ListPosts(ctx context.Context, pag models.PaginationParams) ([]models.Post, int, error)
}

type LikeChecker interface {
	GetLikedIDs(ctx context.Context, userID string, targetType models.LikeTargetType, targetIDs []string) (map[string]bool, error)
	CountByTargets(ctx context.Context, targetIDs []string, targetType models.LikeTargetType) (map[string]int, error)
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
	repo        PostRepository
	likeChecker LikeChecker
}

func NewService(repo PostRepository, likeChecker LikeChecker) *Service {
	return &Service{repo: repo, likeChecker: likeChecker}
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

func (s *Service) DeletePost(ctx context.Context, userID, postID string) error {
	post, err := s.repo.GetPostByID(ctx, postID)
	if err != nil {
		return err
	}
	if post.Author.Id != userID {
		return ErrForbidden
	}
	return s.repo.DeletePost(ctx, postID)
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
		blocks[i] = models.PostBlock{
			Type:     p.Type,
			Images:   p.Images,
			Text:     p.Text,
			RecipeId: p.RecipeId,
		}
	}
	return blocks
}
