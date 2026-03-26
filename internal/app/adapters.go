package app

import (
	"context"
	"fmt"
	"time"

	firebaseAdminAuth "firebase.google.com/go/v4/auth"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
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

type likedRecipesFetcher struct {
	users   users.UserStore
	likes   *likes.Service
	recipes *recipes.Service
}

func (f *likedRecipesFetcher) ListLikedRecipes(ctx context.Context, currentUserID, targetUserID string, params models.ListUserLikesParams) (models.GenericPage[models.LikedRecipe], error) {
	if _, err := f.users.GetByID(ctx, targetUserID); err != nil {
		return models.GenericPage[models.LikedRecipe]{}, err
	}

	pag := params.Pagination()

	total, err := f.likes.CountByUser(ctx, targetUserID, models.LikeTargetTypeRecipe)
	if err != nil {
		return models.GenericPage[models.LikedRecipe]{}, fmt.Errorf("count liked recipes: %w", err)
	}

	if total == 0 {
		return models.NewPage([]models.LikedRecipe{}, pag, 0), nil
	}

	likedList, err := f.likes.ListByUser(ctx, targetUserID, models.LikeTargetTypeRecipe, int(pag.GetLimit()), int(pag.Offset()))
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
