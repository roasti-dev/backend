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

func (s *ServerHandler) ListMyLikes(ctx context.Context, request ListMyLikesRequestObject) (ListMyLikesResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	params := ptr.GetOr(request.Params.ListMyLikes, models.ListMyLikesParams{})
	liked, err := s.userService.ListLikedRecipes(ctx, userID, params)
	if err != nil {
		return nil, err
	}
	return ListMyLikes200JSONResponse(liked), nil
}
