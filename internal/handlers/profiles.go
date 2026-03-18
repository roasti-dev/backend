package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/requestctx"
)

func (s *ServerHandler) GetMyProfile(ctx context.Context, request GetMyProfileRequestObject) (GetMyProfileResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	profile, err := s.authService.CurrentUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return GetMyProfile200JSONResponse(profile), nil
}
