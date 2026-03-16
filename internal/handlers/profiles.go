package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/requestctx"
)

func (s *ServerHandler) GetApiV1ProfilesMe(
	ctx context.Context, request GetApiV1ProfilesMeRequestObject,
) (GetApiV1ProfilesMeResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	profile, err := s.authService.CurrentUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return GetApiV1ProfilesMe200JSONResponse(profile), nil
}
