package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/articles"
	"github.com/nikpivkin/roasti-app-backend/internal/x/ptr"
	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
)

func (s *ServerHandler) ListArticleComments(ctx context.Context, request ListArticleCommentsRequestObject) (ListArticleCommentsResponseObject, error) {
	pag := models.NewPaginationParams(ptr.FromPtr(request.Params.Page), ptr.FromPtr(request.Params.Limit))
	page, err := s.articleService.ListComments(ctx, request.ArticleId, pag)
	if err != nil {
		return nil, err
	}
	return ListArticleComments200JSONResponse(models.CommentPage(page)), nil
}

func (s *ServerHandler) ListArticles(ctx context.Context, request ListArticlesRequestObject) (ListArticlesResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	params := articles.ListArticlesParams{
		AuthorID: request.Params.AuthorId,
		Limit:    request.Params.Limit,
		Page:     request.Params.Page,
	}
	if request.Params.Filter != nil && *request.Params.Filter == Following {
		ids, err := s.followService.ListFollowingUserIDs(ctx, userID)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return ListArticles200JSONResponse(models.ArticlePage(models.EmptyPage[models.Article]())), nil
		}
		params.AuthorIDs = ids
	}
	page, err := s.articleService.ListArticles(ctx, userID, params)
	if err != nil {
		return nil, err
	}
	return ListArticles200JSONResponse(models.ArticlePage(page)), nil
}

func (s *ServerHandler) GetArticle(ctx context.Context, request GetArticleRequestObject) (GetArticleResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	article, err := s.articleService.GetArticle(ctx, userID, request.ArticleId)
	if err != nil {
		return nil, err
	}
	return GetArticle200JSONResponse(article), nil
}

func (s *ServerHandler) ToggleArticleLike(ctx context.Context, request ToggleArticleLikeRequestObject) (ToggleArticleLikeResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	result, err := s.articleService.ToggleLike(ctx, userID, request.ArticleId)
	if err != nil {
		return nil, err
	}
	return ToggleArticleLike200JSONResponse(models.ToggleLikeResponse{
		Liked:      result.Liked,
		LikesCount: int32(result.LikesCount),
	}), nil
}

func (s *ServerHandler) UpdateArticle(ctx context.Context, request UpdateArticleRequestObject) (UpdateArticleResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	article, err := s.articleService.UpdateArticle(ctx, userID, request.ArticleId, *request.Body)
	if err != nil {
		return nil, err
	}
	return UpdateArticle200JSONResponse(article), nil
}

func (s *ServerHandler) DeleteArticle(ctx context.Context, request DeleteArticleRequestObject) (DeleteArticleResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	if err := s.articleService.DeleteArticle(ctx, userID, request.ArticleId); err != nil {
		return nil, err
	}
	return DeleteArticle204Response{}, nil
}

func (s *ServerHandler) CreateArticle(ctx context.Context, request CreateArticleRequestObject) (CreateArticleResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	article, err := s.articleService.CreateArticle(ctx, userID, *request.Body)
	if err != nil {
		return nil, err
	}
	return CreateArticle201JSONResponse(article), nil
}

func (s *ServerHandler) CreateArticleComment(ctx context.Context, request CreateArticleCommentRequestObject) (CreateArticleCommentResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	comment, err := s.articleService.CreateComment(ctx, userID, request.ArticleId, request.Body.Text, request.Body.ParentId)
	if err != nil {
		return nil, err
	}
	return CreateArticleComment201JSONResponse(comment), nil
}
