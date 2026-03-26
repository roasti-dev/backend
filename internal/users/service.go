package users

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

type UserStore interface {
	GetByID(ctx context.Context, userID string) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	Create(ctx context.Context, user User) error
	Update(ctx context.Context, userID string, req UpdateUserFields) error
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// IdentityCreator creates a user identity in an external auth provider
// and returns the assigned UID.
type IdentityCreator interface {
	CreateIdentity(ctx context.Context, email, password string) (uid string, err error)
}

// Uploader confirms uploaded files.
type Uploader interface {
	Confirm(ctx context.Context, fileID string) error
}

// RegisterInput holds the data needed to create a new user.
type RegisterInput struct {
	Email    string
	Username string
	Password string
	AvatarID *string
	Bio      *string
}

type Service struct {
	repo     UserStore
	identity IdentityCreator
	uploader Uploader
}

func NewUserService(
	repo UserStore,
	identity IdentityCreator,
	uploader Uploader,
) *Service {
	return &Service{
		repo:     repo,
		identity: identity,
		uploader: uploader,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (User, error) {
	username, err := NewUsername(input.Username)
	if err != nil {
		return User{}, err
	}

	exists, err := s.repo.ExistsByUsername(ctx, username.Value())
	if err != nil {
		return User{}, fmt.Errorf("check username: %w", err)
	}
	if exists {
		return User{}, ErrUsernameTaken
	}

	exists, err = s.repo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return User{}, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return User{}, ErrEmailTaken
	}

	uid, err := s.identity.CreateIdentity(ctx, input.Email, input.Password)
	if err != nil {
		return User{}, err
	}

	user := User{
		ID:       uid,
		Email:    input.Email,
		Username: username.Value(),
		AvatarID: input.AvatarID,
		Bio:      input.Bio,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	if input.AvatarID != nil {
		if err := s.uploader.Confirm(ctx, *input.AvatarID); err != nil {
			slog.WarnContext(ctx, "failed to confirm avatar", slog.String("avatar_id", *input.AvatarID))
		}
	}

	return s.repo.GetByID(ctx, uid)
}

func (s *Service) FindByUsername(ctx context.Context, username string) (User, error) {
	return s.repo.GetByUsername(ctx, username)
}

func (s *Service) FindByID(ctx context.Context, userID string) (User, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *Service) CurrentUser(ctx context.Context, userID string) (models.UserAccount, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return models.UserAccount{}, fmt.Errorf("get user by id: %w", err)
	}
	return user.ToAccount(), nil
}

func (s *Service) GetPublicProfile(ctx context.Context, userID string) (models.UserProfile, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return models.UserProfile{}, fmt.Errorf("get user by id: %w", err)
	}
	return user.ToPublicProfile(), nil
}

func (s *Service) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return s.repo.ExistsByUsername(ctx, username)
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateUserFields) (models.UserAccount, error) {
	if req.Username != nil {
		exists, err := s.repo.ExistsByUsername(ctx, *req.Username)
		if err != nil {
			return models.UserAccount{}, fmt.Errorf("check username: %w", err)
		}
		if exists {
			return models.UserAccount{}, ErrUsernameTaken
		}
	}

	if err := s.repo.Update(ctx, userID, req); err != nil {
		return models.UserAccount{}, err
	}

	return s.CurrentUser(ctx, userID)
}


