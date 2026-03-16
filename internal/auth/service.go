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
	logger       *slog.Logger
	repo         *Repository
	uploader     *uploads.Service
	firebaseAuth *firebaseAuth.Client
	signer       FirebasePasswordSigner
}

func NewService(
	repo *Repository,
	uploader *uploads.Service,
	firebaseAuth *firebaseAuth.Client,
	passwordSigner FirebasePasswordSigner,
) *Service {
	return &Service{
		logger:       slog.Default(),
		repo:         repo,
		uploader:     uploader,
		firebaseAuth: firebaseAuth,
		signer:       passwordSigner,
	}
}

func (s *Service) CurrentUser(ctx context.Context, userID string) (models.MyProfile, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return models.UserResponse{}, fmt.Errorf("get user by id: %w", err)
	}
	return models.UserResponse{
		Id:       user.ID,
		Username: user.Username,
		AvatarId: user.AvatarID,
		Bio:      user.Bio,
	}, nil
}

func (s *Service) Register(ctx context.Context, req models.RegisterRequest) (models.AuthResponse, error) {
	if err := validateRegisterRequest(req); err != nil {
		return models.AuthResponse{}, err
	}

	exists, err := s.repo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("check username: %w", err)
	}
	if exists {
		return models.AuthResponse{}, ErrUsernameTaken
	}

	exists, err = s.repo.ExistsByEmail(ctx, string(req.Email))
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

	if err := s.repo.Create(ctx, User{
		ID:       firebaseUser.UID,
		Email:    string(req.Email),
		Username: req.Username,
		AvatarID: req.AvatarId,
		Bio:      req.Bio,
	}); err != nil {
		return models.AuthResponse{}, fmt.Errorf("create user: %w", err)
	}

	user, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("get user: %w", err)
	}

	s.confirmAvatar(ctx, user)

	signIn, err := s.signer.SignInWithPassword(ctx, string(req.Email), req.Password)
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("sign in after register: %w", err)
	}

	return models.AuthResponse{
		AccessToken:  signIn.IDToken,
		RefreshToken: signIn.RefreshToken,
		User:         userToResponse(user),
	}, nil
}

func (s *Service) Login(ctx context.Context, req models.LoginRequest) (models.AuthResponse, error) {
	if err := validateLoginRequest(req); err != nil {
		return models.AuthResponse{}, err
	}

	user, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
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
		User:         userToResponse(user),
	}, nil
}

func (s *Service) Refresh(ctx context.Context, req models.RefreshRequest) (models.TokensResponse, error) {
	signIn, err := s.signer.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return models.TokensResponse{}, err
	}

	return models.TokensResponse{
		AccessToken:  signIn.IDToken,
		RefreshToken: signIn.RefreshToken,
	}, nil
}

func (s *Service) Logout(ctx context.Context, userID string) error {
	if err := s.firebaseAuth.RevokeRefreshTokens(ctx, userID); err != nil {
		return fmt.Errorf("revoke refresh tokens: %w", err)
	}
	return nil
}

func (s *Service) confirmAvatar(ctx context.Context, user User) {
	if user.AvatarID != nil {
		if err := s.uploader.Confirm(*user.AvatarID); err != nil {
			s.logger.WarnContext(ctx, "failed to confirm recipe image",
				slog.String("recipe_id", user.ID),
				slog.String("image_id", *user.AvatarID),
			)
		}
	}
}

func userToResponse(u User) models.UserResponse {
	return models.UserResponse{
		Id:       u.ID,
		Username: u.Username,
		AvatarId: u.AvatarID,
		Bio:      u.Bio,
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
