package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/ptr"
	"github.com/nikpivkin/roasti-app-backend/internal/requestctx"
)

func (s *ServerHandler) GetCurrentUser(ctx context.Context, request GetCurrentUserRequestObject) (GetCurrentUserResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	user, err := s.userService.CurrentUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return GetCurrentUser200JSONResponse(user), nil
}

func (s *ServerHandler) ListUserLikes(ctx context.Context, request ListUserLikesRequestObject) (ListUserLikesResponseObject, error) {
	currentUserID := requestctx.GetUserID(ctx)
	params := ptr.GetOr(request.Params.ListUserLikes, models.ListUserLikesParams{})
	liked, err := s.userService.ListLikedRecipes(ctx, currentUserID, request.UserId, params)
	if err != nil {
		return nil, err
	}
	return ListUserLikes200JSONResponse(liked), nil
}
