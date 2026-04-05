package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/posts"
	"github.com/nikpivkin/roasti-app-backend/internal/x/ptr"
	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
)

func (s *ServerHandler) ListPostComments(ctx context.Context, request ListPostCommentsRequestObject) (ListPostCommentsResponseObject, error) {
	pag := models.NewPaginationParams(ptr.FromPtr(request.Params.Page), ptr.FromPtr(request.Params.Limit))
	page, err := s.postService.ListComments(ctx, request.PostId, pag)
	if err != nil {
		return nil, err
	}
	return ListPostComments200JSONResponse(models.CommentPage(page)), nil
}

func (s *ServerHandler) ListPosts(ctx context.Context, request ListPostsRequestObject) (ListPostsResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	page, err := s.postService.ListPosts(ctx, userID, posts.ListPostsParams{
		Limit: request.Params.Limit,
		Page:  request.Params.Page,
	})
	if err != nil {
		return nil, err
	}
	return ListPosts200JSONResponse(models.PostPage(page)), nil
}

func (s *ServerHandler) GetPost(ctx context.Context, request GetPostRequestObject) (GetPostResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	post, err := s.postService.GetPost(ctx, userID, request.PostId)
	if err != nil {
		return nil, err
	}
	return GetPost200JSONResponse(post), nil
}

func (s *ServerHandler) TogglePostLike(ctx context.Context, request TogglePostLikeRequestObject) (TogglePostLikeResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	result, err := s.postService.ToggleLike(ctx, userID, request.PostId)
	if err != nil {
		return nil, err
	}
	return TogglePostLike200JSONResponse(models.ToggleLikeResponse{
		Liked:      result.Liked,
		LikesCount: int32(result.LikesCount),
	}), nil
}

func (s *ServerHandler) UpdatePost(ctx context.Context, request UpdatePostRequestObject) (UpdatePostResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	post, err := s.postService.UpdatePost(ctx, userID, request.PostId, *request.Body)
	if err != nil {
		return nil, err
	}
	return UpdatePost200JSONResponse(post), nil
}

func (s *ServerHandler) DeletePost(ctx context.Context, request DeletePostRequestObject) (DeletePostResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	if err := s.postService.DeletePost(ctx, userID, request.PostId); err != nil {
		return nil, err
	}
	return DeletePost204Response{}, nil
}

func (s *ServerHandler) CreatePost(ctx context.Context, request CreatePostRequestObject) (CreatePostResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	post, err := s.postService.CreatePost(ctx, userID, *request.Body)
	if err != nil {
		return nil, err
	}
	return CreatePost201JSONResponse(post), nil
}

func (s *ServerHandler) CreatePostComment(ctx context.Context, request CreatePostCommentRequestObject) (CreatePostCommentResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	comment, err := s.postService.CreateComment(ctx, userID, request.PostId, request.Body.Text, request.Body.ParentId)
	if err != nil {
		return nil, err
	}
	return CreatePostComment201JSONResponse(comment), nil
}
