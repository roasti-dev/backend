package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	firebaseAuth "firebase.google.com/go/v4/auth"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/users"
)

type FirebasePasswordSigner interface {
	SignInWithPassword(ctx context.Context, email, password string) (SignInResult, error)
	RefreshToken(ctx context.Context, refreshToken string) (SignInResult, error)
}

type SignInResult struct {
	IDToken      string
	RefreshToken string
}

type RevokedTokenStore interface {
	IsRevoked(ctx context.Context, token string) (bool, error)
	Add(ctx context.Context, token string) error
}

// UserService is the subset of users.Service that auth needs.
type UserService interface {
	Register(ctx context.Context, input users.RegisterInput) (users.User, error)
	FindByUsername(ctx context.Context, username string) (users.User, error)
	FindByID(ctx context.Context, userID string) (users.User, error)
}

type Service struct {
	logger        *slog.Logger
	policy        PasswordPolicy
	users         UserService
	revokedTokens RevokedTokenStore
	firebaseAuth  *firebaseAuth.Client
	signer        FirebasePasswordSigner
}

func NewService(
	users UserService,
	revokedTokens RevokedTokenStore,
	firebaseAuth *firebaseAuth.Client,
	passwordSigner FirebasePasswordSigner,
	policy PasswordPolicy,
) *Service {
	return &Service{
		logger:        slog.Default(),
		policy:        policy,
		users:         users,
		revokedTokens: revokedTokens,
		firebaseAuth:  firebaseAuth,
		signer:        passwordSigner,
	}
}

func (s *Service) Register(ctx context.Context, req models.RegisterRequest) (models.AuthResponse, error) {
	password, err := NewPassword(req.Password, s.policy)
	if err != nil {
		return models.AuthResponse{}, err
	}

	user, err := s.users.Register(ctx, users.RegisterInput{
		Email:    req.Email,
		Username: req.Username,
		Password: password.Value(),
		Name:     req.Name,
		AvatarID: req.AvatarId,
		Bio:      req.Bio,
	})
	if err != nil {
		return models.AuthResponse{}, err
	}

	signIn, err := s.signer.SignInWithPassword(ctx, string(req.Email), req.Password)
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("sign in after register: %w", err)
	}

	return models.AuthResponse{
		AccessToken:  signIn.IDToken,
		RefreshToken: signIn.RefreshToken,
		User:         user.ToAccount(),
	}, nil
}

func (s *Service) Login(ctx context.Context, req models.LoginRequest) (models.AuthResponse, error) {
	user, err := s.users.FindByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			return models.AuthResponse{}, ErrInvalidCredentials
		}
		return models.AuthResponse{}, fmt.Errorf("get user by username: %w", err)
	}

	signIn, err := s.signer.SignInWithPassword(ctx, user.Email, req.Password)
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("sign in: %w", err)
	}

	return models.AuthResponse{
		AccessToken:  signIn.IDToken,
		RefreshToken: signIn.RefreshToken,
		User:         user.ToAccount(),
	}, nil
}

func (s *Service) Refresh(ctx context.Context, token string) (models.TokensResponse, error) {
	revoked, err := s.revokedTokens.IsRevoked(ctx, token)
	if err != nil {
		return models.TokensResponse{}, fmt.Errorf("check revoked token: %w", err)
	}
	if revoked {
		return models.TokensResponse{}, ErrTokenRevoked
	}

	signIn, err := s.signer.RefreshToken(ctx, token)
	if err != nil {
		return models.TokensResponse{}, fmt.Errorf("refresh token: %w", err)
	}

	return models.TokensResponse{
		AccessToken:  signIn.IDToken,
		RefreshToken: signIn.RefreshToken,
	}, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID string, req models.ChangePasswordRequest) (models.TokensResponse, error) {
	newPassword, err := NewPassword(strings.TrimSpace(req.NewPassword), s.policy)
	if err != nil {
		return models.TokensResponse{}, err
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return models.TokensResponse{}, fmt.Errorf("get current user: %w", err)
	}

	if _, err := s.signer.SignInWithPassword(ctx, user.Email, req.CurrentPassword); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return models.TokensResponse{}, ErrIncorrectPassword
		}
		return models.TokensResponse{}, fmt.Errorf("verify current password: %w", err)
	}

	params := (&firebaseAuth.UserToUpdate{}).Password(newPassword.Value())
	if _, err := s.firebaseAuth.UpdateUser(ctx, userID, params); err != nil {
		return models.TokensResponse{}, fmt.Errorf("update firebase password: %w", err)
	}

	if err := s.firebaseAuth.RevokeRefreshTokens(ctx, userID); err != nil {
		return models.TokensResponse{}, fmt.Errorf("revoke sessions: %w", err)
	}

	signIn, err := s.signer.SignInWithPassword(ctx, user.Email, req.NewPassword)
	if err != nil {
		return models.TokensResponse{}, fmt.Errorf("sign in after password change: %w", err)
	}

	return models.TokensResponse{
		AccessToken:  signIn.IDToken,
		RefreshToken: signIn.RefreshToken,
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if err := s.revokedTokens.Add(ctx, refreshToken); err != nil {
		return fmt.Errorf("mark token as revoked: %w", err)
	}
	return nil
}
