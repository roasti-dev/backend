package users

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
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

type RecipeService interface {
	GetRecipesByIDs(ctx context.Context, currentUserID string, ids []string) ([]models.Recipe, error)
}

type LikesRepository interface {
	CountByUser(ctx context.Context, userID string, targetType models.LikeTargetType) (int, error)
	ListByUser(ctx context.Context, userID string, targetType models.LikeTargetType, limit, offset int) ([]likes.Like, error)
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
	recipes  RecipeService
	likes    LikesRepository
}

func NewUserService(
	repo UserStore,
	identity IdentityCreator,
	uploader Uploader,
	recipes RecipeService,
	likes LikesRepository,
) *Service {
	return &Service{
		repo:     repo,
		identity: identity,
		uploader: uploader,
		recipes:  recipes,
		likes:    likes,
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

func (s *Service) GetByUsername(ctx context.Context, username string) (models.UserAccount, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return models.UserAccount{}, err
	}
	return user.ToAccount(), nil
}

func (s *Service) Create(ctx context.Context, user User) (models.UserAccount, error) {
	if err := s.repo.Create(ctx, user); err != nil {
		return models.UserAccount{}, err
	}
	created, err := s.repo.GetByUsername(ctx, user.Username)
	if err != nil {
		return models.UserAccount{}, fmt.Errorf("get user: %w", err)
	}
	return created.ToAccount(), nil
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

func (s *Service) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return s.repo.ExistsByEmail(ctx, email)
}

func (s *Service) ListLikedRecipes(ctx context.Context, currentUserID, targetUserID string, params models.ListUserLikesParams) (models.GenericPage[models.LikedRecipe], error) {
	_, err := s.repo.GetByID(ctx, targetUserID)
	if err != nil {
		return models.GenericPage[models.LikedRecipe]{}, err
	}

	pag := params.Pagination()

	total, err := s.likes.CountByUser(ctx, targetUserID, models.LikeTargetTypeRecipe)
	if err != nil {
		return models.GenericPage[models.LikedRecipe]{}, fmt.Errorf("count liked recipes: %w", err)
	}

	if total == 0 {
		return models.NewPage([]models.LikedRecipe{}, pag, 0), nil
	}

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

	return models.NewPage(result, pag, total), nil
}
