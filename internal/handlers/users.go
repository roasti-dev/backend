package handlers

import (
	"context"
	"encoding/json"
	"fmt"

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
		Name:     request.Body.Name,
		Bio:      request.Body.Bio,
		AvatarID: request.Body.AvatarId,
	})
	if err != nil {
		return nil, err
	}
	return UpdateCurrentUser200JSONResponse(updated), nil
}

func (s *ServerHandler) GetUserProfile(ctx context.Context, request GetUserProfileRequestObject) (GetUserProfileResponseObject, error) {
	profile, err := s.userService.GetPublicProfileByUsername(ctx, request.Username)
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

func (s *ServerHandler) ListUsers(ctx context.Context, request ListUsersRequestObject) (ListUsersResponseObject, error) {
	if request.Params.Sort == nil || *request.Params.Sort != Recommended {
		return ListUsers200JSONResponse(models.UserPreviewPage(models.EmptyPage[models.UserPreview]())), nil
	}
	currentUserID := requestctx.GetUserID(ctx)
	page, err := s.userService.ListRecommended(ctx, currentUserID, users.ListRecommendedParams{
		Page:  request.Params.Page,
		Limit: request.Params.Limit,
	})
	if err != nil {
		return nil, err
	}
	return ListUsers200JSONResponse(models.UserPreviewPage(page)), nil
}

func (s *ServerHandler) ListUserLikes(ctx context.Context, request ListUserLikesRequestObject) (ListUserLikesResponseObject, error) {
	currentUserID := requestctx.GetUserID(ctx)

	params := models.ListUserLikesParams{
		Type:          request.Params.Type,
		Limit:         request.Params.Limit,
		Page:          request.Params.Page,
		SortDirection: request.Params.SortDirection,
	}

	var page any
	var err error
	switch request.Params.Type {
	case models.LikeTargetTypePost:
		page, err = s.userLibrary.ListLikedPosts(ctx, currentUserID, request.UserId, params)
	case models.LikeTargetTypeRecipe:
		page, err = s.userLibrary.ListLikedRecipes(ctx, currentUserID, request.UserId, params)
	case models.LikeTargetTypeBean:
	default:
		return ListUserLikes400JSONResponse{Error: fmt.Sprintf("unsupported type: %s", request.Params.Type)}, nil
	}
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(page)
	if err != nil {
		return nil, err
	}
	return ListUserLikes200JSONResponse{union: data}, nil
}
