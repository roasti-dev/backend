package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
)

func (s *ServerHandler) UpdateComment(ctx context.Context, request UpdateCommentRequestObject) (UpdateCommentResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	comment, err := s.commentService.Update(ctx, userID, request.CommentId, request.Body.Text)
	if err != nil {
		return nil, err
	}
	return UpdateComment200JSONResponse(comment), nil
}

func (s *ServerHandler) DeleteComment(ctx context.Context, request DeleteCommentRequestObject) (DeleteCommentResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	if err := s.commentService.Delete(ctx, userID, request.CommentId); err != nil {
		return nil, err
	}
	return DeleteComment204Response{}, nil
}
