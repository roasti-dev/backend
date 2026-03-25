package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	firebaseAuth "firebase.google.com/go/v4/auth"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
	"github.com/nikpivkin/roasti-app-backend/internal/users"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 32
	minUsernameLen = 6
	maxUsernameLen = 16
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type FirebasePasswordSigner interface {
	SignInWithPassword(ctx context.Context, email, password string) (SignInResult, error)
	RefreshToken(ctx context.Context, refreshToken string) (SignInResult, error)
}

type SignInResult struct {
	IDToken      string
	RefreshToken string
}

type Service struct {
	logger        *slog.Logger
	users         *users.Service
	revokedTokens *RevokedTokenRepository
	uploader      *uploads.Service
	firebaseAuth  *firebaseAuth.Client
	signer        FirebasePasswordSigner
}

func NewService(
	users *users.Service,
	revokedTokens *RevokedTokenRepository,
	uploader *uploads.Service,
	firebaseAuth *firebaseAuth.Client,
	passwordSigner FirebasePasswordSigner,
) *Service {
	return &Service{
		logger:        slog.Default(),
		users:         users,
		revokedTokens: revokedTokens,
		uploader:      uploader,
		firebaseAuth:  firebaseAuth,
		signer:        passwordSigner,
	}
}

func (s *Service) Register(ctx context.Context, req models.RegisterRequest) (models.AuthResponse, error) {
	if err := validateRegisterRequest(req); err != nil {
		return models.AuthResponse{}, err
	}

	exists, err := s.users.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("check username: %w", err)
	}
	if exists {
		return models.AuthResponse{}, ErrUsernameTaken
	}

	exists, err = s.users.ExistsByEmail(ctx, string(req.Email))
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return models.AuthResponse{}, ErrEmailTaken
	}

	userToCreate := new(firebaseAuth.UserToCreate).Email(string(req.Email)).Password(req.Password)
	firebaseUser, err := s.firebaseAuth.CreateUser(ctx, userToCreate)
	if err != nil {
		if firebaseAuth.IsEmailAlreadyExists(err) {
			return models.AuthResponse{}, ErrEmailTaken
		}
		return models.AuthResponse{}, fmt.Errorf("create firebase user: %w", err)
	}

	created, err := s.users.Create(ctx, users.User{
		ID:       firebaseUser.UID,
		Email:    string(req.Email),
		Username: req.Username,
		AvatarID: req.AvatarId,
		Bio:      req.Bio,
	})
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("create user: %w", err)
	}

	s.confirmAvatar(ctx, created)

	signIn, err := s.signer.SignInWithPassword(ctx, string(req.Email), req.Password)
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("sign in after register: %w", err)
	}

	return models.AuthResponse{
		AccessToken:  signIn.IDToken,
		RefreshToken: signIn.RefreshToken,
		User:         created,
	}, nil
}

func (s *Service) Login(ctx context.Context, req models.LoginRequest) (models.AuthResponse, error) {
	if err := validateLoginRequest(req); err != nil {
		return models.AuthResponse{}, err
	}

	user, err := s.users.GetByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			return models.AuthResponse{}, ErrInvalidCredentials
		}
		return models.AuthResponse{}, fmt.Errorf("get user by username: %w", err)
	}

	signIn, err := s.signer.SignInWithPassword(ctx, string(user.Email), req.Password)
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("sign in: %w", err)
	}

	return models.AuthResponse{
		AccessToken:  signIn.IDToken,
		RefreshToken: signIn.RefreshToken,
		User:         user,
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
		return models.TokensResponse{}, err
	}

	return models.TokensResponse{
		AccessToken:  signIn.IDToken,
		RefreshToken: signIn.RefreshToken,
	}, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID string, req models.ChangePasswordRequest) error {
	newPassword := strings.TrimSpace(req.NewPassword)
	if len(newPassword) < minPasswordLen {
		return ErrPasswordTooShort
	}
	if len(newPassword) > maxPasswordLen {
		return ErrPasswordTooLong
	}

	user, err := s.users.CurrentUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}

	if _, err := s.signer.SignInWithPassword(ctx, string(user.Email), req.CurrentPassword); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return ErrIncorrectPassword
		}
		return fmt.Errorf("verify current password: %w", err)
	}

	params := (&firebaseAuth.UserToUpdate{}).Password(newPassword)
	if _, err := s.firebaseAuth.UpdateUser(ctx, userID, params); err != nil {
		return fmt.Errorf("update firebase password: %w", err)
	}

	return nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if err := s.revokedTokens.Add(ctx, refreshToken); err != nil {
		return fmt.Errorf("mark token as revoked: %w", err)
	}
	return nil
}

func (s *Service) confirmAvatar(ctx context.Context, user models.UserAccount) {
	if user.AvatarId != nil {
		if err := s.uploader.Confirm(ctx, *user.AvatarId); err != nil {
			s.logger.WarnContext(ctx, "failed to confirm recipe image",
				slog.String("user_id", user.Id),
				slog.String("image_id", *user.AvatarId),
			)
		}
	}
}

func validateRegisterRequest(req models.RegisterRequest) error {
	if strings.TrimSpace(string(req.Email)) == "" {
		return ErrInvalidEmail
	}

	password := strings.TrimSpace(req.Password)
	if len(password) < minPasswordLen {
		return ErrPasswordTooShort
	}
	if len(password) > maxPasswordLen {
		return ErrPasswordTooLong
	}

	username := strings.TrimSpace(req.Username)
	if len(username) < minUsernameLen {
		return ErrUsernameTooShort
	}
	if len(username) > maxUsernameLen {
		return ErrUsernameTooLong
	}
	if !usernameRegex.MatchString(username) {
		return ErrInvalidUsernameFormat
	}
	return nil
}

func validateLoginRequest(req models.LoginRequest) error {
	if strings.TrimSpace(req.Username) == "" {
		return ErrUsernameRequired
	}
	if strings.TrimSpace(req.Password) == "" {
		return ErrPasswordRequired
	}
	return nil
}
