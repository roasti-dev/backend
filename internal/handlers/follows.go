package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/follows"
	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
)

func (s *ServerHandler) FollowUser(ctx context.Context, request FollowUserRequestObject) (FollowUserResponseObject, error) {
	currentUserID := requestctx.GetUserID(ctx)
	if err := s.followService.Follow(ctx, currentUserID, request.UserId); err != nil {
		return nil, err
	}
	return FollowUser200Response{}, nil
}

func (s *ServerHandler) UnfollowUser(ctx context.Context, request UnfollowUserRequestObject) (UnfollowUserResponseObject, error) {
	currentUserID := requestctx.GetUserID(ctx)
	if err := s.followService.Unfollow(ctx, currentUserID, request.UserId); err != nil {
		return nil, err
	}
	return UnfollowUser204Response{}, nil
}

func (s *ServerHandler) ListFollowing(ctx context.Context, request ListFollowingRequestObject) (ListFollowingResponseObject, error) {
	currentUserID := requestctx.GetUserID(ctx)
	page, err := s.followService.ListFollowing(ctx, currentUserID, follows.ListParams{
		Page:  request.Params.Page,
		Limit: request.Params.Limit,
	})
	if err != nil {
		return nil, err
	}
	return ListFollowing200JSONResponse(models.UserPreviewPage(page)), nil
}

func (s *ServerHandler) ListFollowers(ctx context.Context, request ListFollowersRequestObject) (ListFollowersResponseObject, error) {
	currentUserID := requestctx.GetUserID(ctx)
	page, err := s.followService.ListFollowers(ctx, currentUserID, follows.ListParams{
		Page:  request.Params.Page,
		Limit: request.Params.Limit,
	})
	if err != nil {
		return nil, err
	}
	return ListFollowers200JSONResponse(models.UserPreviewPage(page)), nil
}
