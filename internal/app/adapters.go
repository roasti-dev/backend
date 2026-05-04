package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	firebaseAdminAuth "firebase.google.com/go/v4/auth"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/articles"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/users"
)

type firebaseIdentityCreator struct {
	client *firebaseAdminAuth.Client
}

func (f *firebaseIdentityCreator) CreateIdentity(ctx context.Context, email, password string) (string, error) {
	params := new(firebaseAdminAuth.UserToCreate).Email(email).Password(password)
	user, err := f.client.CreateUser(ctx, params)
	if err != nil {
		if firebaseAdminAuth.IsEmailAlreadyExists(err) {
			return "", users.ErrEmailTaken
		}
		return "", fmt.Errorf("create firebase user: %w", err)
	}
	return user.UID, nil
}

type userStore interface {
	GetByID(ctx context.Context, userID string) (users.User, error)
}

type userLibrary struct {
	users    userStore
	likes    *likes.Service
	recipes  *recipes.Service
	articles *articles.Service
}

// TODO: ListLikedArticles and ListLikedRecipes share identical pagination/like-fetching logic.
// Consider extracting a generic helper if more likeable types are added.
func (f *userLibrary) ListLikedArticles(ctx context.Context, currentUserID, targetUserID string, params models.ListUserLikesParams) (models.GenericPage[models.LikedArticle], error) {
	if _, err := f.users.GetByID(ctx, targetUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GenericPage[models.LikedArticle]{}, users.ErrNotFound
		}
		return models.GenericPage[models.LikedArticle]{}, err
	}

	pag := params.Pagination()

	likedList, total, err := f.likes.ListByUser(ctx, targetUserID, models.LikeTargetTypeArticle, int(pag.GetLimit()), int(pag.Offset()))
	if err != nil {
		return models.GenericPage[models.LikedArticle]{}, fmt.Errorf("list liked articles: %w", err)
	}

	if total == 0 {
		return models.NewPage([]models.LikedArticle{}, pag, 0), nil
	}

	ids := make([]string, len(likedList))
	likedAtMap := make(map[string]time.Time, len(likedList))
	for i, l := range likedList {
		ids[i] = l.TargetID
		likedAtMap[l.TargetID] = l.CreatedAt
	}

	articleList, err := f.articles.GetArticlesByIDs(ctx, currentUserID, ids)
	if err != nil {
		return models.GenericPage[models.LikedArticle]{}, fmt.Errorf("get articles: %w", err)
	}

	result := make([]models.LikedArticle, len(articleList))
	for i, p := range articleList {
		result[i] = models.LikedArticle{LikedAt: likedAtMap[p.Id], Article: p}
	}
	return models.NewPage(result, pag, total), nil
}

func (f *userLibrary) ListLikedRecipes(ctx context.Context, currentUserID, targetUserID string, params models.ListUserLikesParams) (models.GenericPage[models.LikedRecipe], error) {
	if _, err := f.users.GetByID(ctx, targetUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.GenericPage[models.LikedRecipe]{}, users.ErrNotFound
		}
		return models.GenericPage[models.LikedRecipe]{}, err
	}

	pag := params.Pagination()

	likedList, total, err := f.likes.ListByUser(ctx, targetUserID, models.LikeTargetTypeRecipe, int(pag.GetLimit()), int(pag.Offset()))
	if err != nil {
		return models.GenericPage[models.LikedRecipe]{}, fmt.Errorf("list liked recipes: %w", err)
	}

	if total == 0 {
		return models.NewPage([]models.LikedRecipe{}, pag, 0), nil
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

	recipeList, err := f.recipes.GetRecipesByIDs(ctx, currentUserID, ids)
	if err != nil {
		return models.GenericPage[models.LikedRecipe]{}, fmt.Errorf("get recipes: %w", err)
	}

	result := make([]models.LikedRecipe, len(recipeList))
	for i, p := range recipeList {
		result[i] = models.LikedRecipe{LikedAt: likedAtMap[p.Id], Recipe: p}
	}

	return models.NewPage(result, pag, total), nil
}
