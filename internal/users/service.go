package users

import (
	"context"
	"fmt"
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
)

type RecipeService interface {
	// GetPreviewsByIDs(ctx context.Context, currentUserID, ids []string) ([]models.RecipePreview, error)
	GetRecipesByIDs(ctx context.Context, currentUserID string, ids []string) ([]models.Recipe, error)
}

type LikesRepository interface {
	ListByUser(ctx context.Context, userID string, targetType models.LikeTargetType, limit, offset int) ([]likes.Like, error)
}

type Service struct {
	repo    *UserRepository
	recipes RecipeService
	likes   LikesRepository
}

func NewUserService(repo *UserRepository, recipes RecipeService, likes LikesRepository) *Service {
	return &Service{
		repo:    repo,
		recipes: recipes,
		likes:   likes,
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

func (s *Service) ListLikedRecipes(ctx context.Context, currentUserID, targetUserID string, params models.ListUserLikesParams) (models.GenericPage[models.LikedRecipe], error) {
	_, err := s.repo.GetByID(ctx, targetUserID)
	if err != nil {
		return models.GenericPage[models.LikedRecipe]{}, err
	}

	pag := params.Pagination()

	likedList, err := s.likes.ListByUser(ctx, targetUserID, models.LikeTargetTypeRecipe, int(pag.GetLimit()), int(pag.Offset()))
	if err != nil {
		return models.GenericPage[models.LikedRecipe]{}, fmt.Errorf("list liked recipes: %w", err)
	}

	if len(likedList) == 0 {
		return models.NewPage([]models.LikedRecipe{}, pag, 0), nil
	}

	ids := make([]string, len(likedList))
	likedAtMap := make(map[string]time.Time, len(likedList))
	for i, l := range likedList {
		ids[i] = l.TargetID
		likedAtMap[l.TargetID] = l.CreatedAt
	}

	recipes, err := s.recipes.GetRecipesByIDs(ctx, currentUserID, ids)
	if err != nil {
		return models.GenericPage[models.LikedRecipe]{}, fmt.Errorf("get recipe previews: %w", err)
	}

	result := make([]models.LikedRecipe, len(recipes))
	for i, p := range recipes {
		result[i] = models.LikedRecipe{
			LikedAt: likedAtMap[p.Id],
			Recipe:  p,
		}
	}

	return models.NewPage(result, pag, len(likedList)), nil
}
