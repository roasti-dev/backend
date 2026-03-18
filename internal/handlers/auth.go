package handlers

import (
	"context"
)

func (s *ServerHandler) RegisterUser(ctx context.Context, request RegisterUserRequestObject) (RegisterUserResponseObject, error) {
	resp, err := s.authService.Register(ctx, *request.Body)
	if err != nil {
		return nil, err
	}
	return RegisterUser201JSONResponse(resp), nil
}

func (s *ServerHandler) LoginUser(ctx context.Context, request LoginUserRequestObject) (LoginUserResponseObject, error) {
	resp, err := s.authService.Login(ctx, *request.Body)
	if err != nil {
		return nil, err
	}
	return LoginUser200JSONResponse(resp), nil
}

func (s *ServerHandler) RefreshToken(ctx context.Context, request RefreshTokenRequestObject) (RefreshTokenResponseObject, error) {
	resp, err := s.authService.Refresh(ctx, *request.Body)
	if err != nil {
		return nil, err
	}
	return RefreshToken200JSONResponse(resp), nil
}

func (s *ServerHandler) LogoutUser(ctx context.Context, request LogoutUserRequestObject) (LogoutUserResponseObject, error) {
	if err := s.authService.Logout(ctx, request.Body.RefreshToken); err != nil {
		return nil, err
	}
	return LogoutUser204Response{}, nil
}
