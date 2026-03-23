package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/requestctx"
)

func (s *ServerHandler) GetCurrentUser(ctx context.Context, request GetCurrentUserRequestObject) (GetCurrentUserResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	profile, err := s.userService.CurrentUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return GetCurrentUser200JSONResponse(profile), nil
}
