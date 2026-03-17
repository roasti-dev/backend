package handlers

import (
	"context"
)

func (s *ServerHandler) PostApiV1AuthRegister(ctx context.Context, request PostApiV1AuthRegisterRequestObject) (PostApiV1AuthRegisterResponseObject, error) {
	resp, err := s.authService.Register(ctx, *request.Body)
	if err != nil {
		return nil, err
	}
	return PostApiV1AuthRegister201JSONResponse(resp), nil
}

func (s *ServerHandler) PostApiV1AuthLogin(ctx context.Context, request PostApiV1AuthLoginRequestObject) (PostApiV1AuthLoginResponseObject, error) {
	resp, err := s.authService.Login(ctx, *request.Body)
	if err != nil {
		return nil, err
	}
	return PostApiV1AuthLogin200JSONResponse(resp), nil
}

func (s *ServerHandler) PostApiV1AuthRefresh(ctx context.Context, request PostApiV1AuthRefreshRequestObject) (PostApiV1AuthRefreshResponseObject, error) {
	resp, err := s.authService.Refresh(ctx, *request.Body)
	if err != nil {
		return nil, err
	}
	return PostApiV1AuthRefresh200JSONResponse(resp), nil
}

func (s *ServerHandler) PostApiV1AuthLogout(ctx context.Context, request PostApiV1AuthLogoutRequestObject) (PostApiV1AuthLogoutResponseObject, error) {
	if err := s.authService.Logout(ctx, request.Body.RefreshToken); err != nil {
		return nil, err
	}
	return PostApiV1AuthLogout204Response{}, nil
}
