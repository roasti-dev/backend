package users

import (
	"context"
	"database/sql"
	"errors"
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

// Uploader manages uploaded files.
type Uploader interface {
	Confirm(ctx context.Context, fileID string) error
	Delete(ctx context.Context, fileID string) error
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

	email, err := NewEmail(input.Email)
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

	exists, err = s.repo.ExistsByEmail(ctx, email.Value())
	if err != nil {
		return User{}, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return User{}, ErrEmailTaken
	}

	uid, err := s.identity.CreateIdentity(ctx, email.Value(), input.Password)
	if err != nil {
		return User{}, err
	}

	user := User{
		ID:       uid,
		Email:    email.Value(),
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
	user, err := s.repo.GetByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (s *Service) FindByID(ctx context.Context, userID string) (User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (s *Service) CurrentUser(ctx context.Context, userID string) (models.UserAccount, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.UserAccount{}, ErrNotFound
		}
		return models.UserAccount{}, fmt.Errorf("get user by id: %w", err)
	}
	return user.ToAccount(), nil
}

func (s *Service) GetPublicProfile(ctx context.Context, userID string) (models.UserProfile, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.UserProfile{}, ErrNotFound
		}
		return models.UserProfile{}, fmt.Errorf("get user by id: %w", err)
	}
	return user.ToPublicProfile(), nil
}

func (s *Service) GetPublicProfileByUsername(ctx context.Context, username string) (models.UserProfile, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.UserProfile{}, ErrNotFound
		}
		return models.UserProfile{}, fmt.Errorf("get user by username: %w", err)
	}
	return user.ToPublicProfile(), nil
}

func (s *Service) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return s.repo.ExistsByUsername(ctx, username)
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateUserFields) (models.UserAccount, error) {
	if req.Username != nil {
		username, err := NewUsername(*req.Username)
		if err != nil {
			return models.UserAccount{}, err
		}
		exists, err := s.repo.ExistsByUsername(ctx, username.Value())
		if err != nil {
			return models.UserAccount{}, fmt.Errorf("check username: %w", err)
		}
		if exists {
			return models.UserAccount{}, ErrUsernameTaken
		}
	}

	var oldAvatarID *string
	if req.AvatarID.IsSpecified() {
		current, err := s.repo.GetByID(ctx, userID)
		if err != nil {
			return models.UserAccount{}, fmt.Errorf("get current user: %w", err)
		}
		oldAvatarID = current.AvatarID
	}

	if err := s.repo.Update(ctx, userID, req); err != nil {
		return models.UserAccount{}, err
	}

	if req.AvatarID.IsSpecified() {
		if !req.AvatarID.IsNull() {
			newID := req.AvatarID.MustGet()
			if err := s.uploader.Confirm(ctx, newID); err != nil {
				slog.WarnContext(ctx, "failed to confirm avatar", slog.String("avatar_id", newID))
			}
		}
		if oldAvatarID != nil {
			if err := s.uploader.Delete(ctx, *oldAvatarID); err != nil {
				slog.WarnContext(ctx, "failed to delete old avatar", slog.String("avatar_id", *oldAvatarID))
			}
		}
	}

	return s.CurrentUser(ctx, userID)
}
