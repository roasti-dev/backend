package users

import (
	"context"
	"fmt"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

type Service struct {
	repo *UserRepository
}

func NewUserService(repo *UserRepository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CurrentUser(ctx context.Context, userID string) (models.CurrentUser, error) {
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

func (s *Service) GetByUsername(ctx context.Context, username string) (User, error) {
	return s.repo.GetByUsername(ctx, username)
}

func (s *Service) Create(ctx context.Context, user User) (User, error) {
	if err := s.repo.Create(ctx, user); err != nil {
		return User{}, nil
	}
	created, err := s.repo.GetByUsername(ctx, user.Username)
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return created, nil
}

func (s *Service) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return s.repo.ExistsByUsername(ctx, username)
}

func (s *Service) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return s.repo.ExistsByEmail(ctx, email)
}
