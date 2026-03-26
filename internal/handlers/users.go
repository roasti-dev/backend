package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/users"
	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
)

func (s *ServerHandler) GetCurrentUser(ctx context.Context, request GetCurrentUserRequestObject) (GetCurrentUserResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	user, err := s.userService.CurrentUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return GetCurrentUser200JSONResponse(user), nil
}

func (s *ServerHandler) UpdateCurrentUser(ctx context.Context, request UpdateCurrentUserRequestObject) (UpdateCurrentUserResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	updated, err := s.userService.UpdateProfile(ctx, userID, users.UpdateUserFields{
		Username: request.Body.Username,
		Bio:      request.Body.Bio,
		AvatarID: request.Body.AvatarId,
	})
	if err != nil {
		return nil, err
	}
	return UpdateCurrentUser200JSONResponse(updated), nil
}

func (s *ServerHandler) GetUserProfile(ctx context.Context, request GetUserProfileRequestObject) (GetUserProfileResponseObject, error) {
	profile, err := s.userService.GetPublicProfile(ctx, request.UserId)
	if err != nil {
		return nil, err
	}
	return GetUserProfile200JSONResponse(profile), nil
}

func (s *ServerHandler) CheckUsernameAvailability(ctx context.Context, request CheckUsernameAvailabilityRequestObject) (CheckUsernameAvailabilityResponseObject, error) {
	exists, err := s.userService.ExistsByUsername(ctx, request.Params.Username)
	if err != nil {
		return nil, err
	}
	return CheckUsernameAvailability200JSONResponse{Available: !exists}, nil
}

func (s *ServerHandler) ListUserLikes(ctx context.Context, request ListUserLikesRequestObject) (ListUserLikesResponseObject, error) {
	currentUserID := requestctx.GetUserID(ctx)
	liked, err := s.userLibrary.ListLikedRecipes(ctx, currentUserID, request.UserId, models.ListUserLikesParams{
		Type:          request.Params.Type,
		Limit:         request.Params.Limit,
		Page:          request.Params.Page,
		SortDirection: request.Params.SortDirection,
	})
	if err != nil {
		return nil, err
	}
	return ListUserLikes200JSONResponse(liked), nil
}
