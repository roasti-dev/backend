package handlers

import (
	"cmp"
	"context"
	"log/slog"
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/auth"
	"github.com/nikpivkin/roasti-app-backend/internal/x/ptr"
	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
)

func (s *ServerHandler) RegisterUser(ctx context.Context, request RegisterUserRequestObject) (RegisterUserResponseObject, error) {
	resp, err := s.authService.Register(ctx, *request.Body)
	if err != nil {
		return nil, err
	}
	return RegisterUser201WithCookieResponse{RegisterUser201JSONResponse(resp)}, nil
}

func (s *ServerHandler) LoginUser(ctx context.Context, request LoginUserRequestObject) (LoginUserResponseObject, error) {
	resp, err := s.authService.Login(ctx, *request.Body)
	if err != nil {
		return nil, err
	}
	return LoginUser200WithCookieResponse{LoginUser200JSONResponse(resp)}, nil
}

func (s *ServerHandler) RefreshToken(ctx context.Context, request RefreshTokenRequestObject) (RefreshTokenResponseObject, error) {
	body := ptr.GetOr(request.Body, models.RefreshRequest{})
	refreshToken := cmp.Or(
		body.RefreshToken,
		requestctx.GetRefreshToken(ctx),
	)

	if refreshToken == "" {
		return nil, auth.ErrInvalidRefreshToken
	}

	resp, err := s.authService.Refresh(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	return RefreshToken200WithCookieResponse{RefreshToken200JSONResponse(resp)}, nil
}

func (s *ServerHandler) ChangePassword(ctx context.Context, request ChangePasswordRequestObject) (ChangePasswordResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	if err := s.authService.ChangePassword(ctx, userID, *request.Body); err != nil {
		return nil, err
	}
	return ChangePassword204Response{}, nil
}

func (s *ServerHandler) LogoutUser(ctx context.Context, request LogoutUserRequestObject) (LogoutUserResponseObject, error) {
	body := ptr.GetOr(request.Body, models.LogoutRequest{})
	refreshToken := cmp.Or(
		body.RefreshToken,
		requestctx.GetRefreshToken(ctx),
	)
	slog.InfoContext(ctx, "LogoutUser", slog.String("token", refreshToken))

	if refreshToken == "" {
		return nil, auth.ErrInvalidRefreshToken
	}

	if err := s.authService.Logout(ctx, refreshToken); err != nil {
		return nil, err
	}
	return LogoutUser204WithCookieResponse{}, nil
}

type RegisterUser201WithCookieResponse struct {
	RegisterUser201JSONResponse
}

func (r RegisterUser201WithCookieResponse) VisitRegisterUserResponse(w http.ResponseWriter) error {
	setAuthCookies(w, r.AccessToken, r.RefreshToken)
	return r.RegisterUser201JSONResponse.VisitRegisterUserResponse(w)
}

type LoginUser200WithCookieResponse struct {
	LoginUser200JSONResponse
}

func (r LoginUser200WithCookieResponse) VisitLoginUserResponse(w http.ResponseWriter) error {
	setAuthCookies(w, r.AccessToken, r.RefreshToken)
	return r.LoginUser200JSONResponse.VisitLoginUserResponse(w)
}

type RefreshToken200WithCookieResponse struct {
	RefreshToken200JSONResponse
}

func (r RefreshToken200WithCookieResponse) VisitRefreshTokenResponse(w http.ResponseWriter) error {
	setAuthCookies(w, r.AccessToken, r.RefreshToken)
	return r.RefreshToken200JSONResponse.VisitRefreshTokenResponse(w)
}

type LogoutUser204WithCookieResponse struct {
	LogoutUser204Response
}

func (r LogoutUser204WithCookieResponse) VisitLogoutUserResponse(w http.ResponseWriter) error {
	clearAuthCookies(w)
	return r.LogoutUser204Response.VisitLogoutUserResponse(w)
}

func setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		HttpOnly: true,
		Secure:   false, // TODO: set true for production
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   false, // TODO: set true for production
		SameSite: http.SameSiteLaxMode,
		Path:     "/api/v1/auth",
	})
}

func clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   "access_token",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:   "refresh_token",
		Value:  "",
		MaxAge: -1,
		Path:   "/api/v1/auth",
	})
}
