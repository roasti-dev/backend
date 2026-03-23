package recipes_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
)

func createTestRecipe(t *testing.T, repo *recipes.Repository) models.Recipe {
	t.Helper()
	r := models.Recipe{
		Id:          "recipe-1",
		AuthorId:    "user-1",
		Title:       "Test Recipe",
		Description: "Test",
		BrewMethod:  models.V60,
		Difficulty:  models.DifficultyEasy,
		Public:      true,
		Steps:       []models.BrewStep{},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.UpsertRecipe(t.Context(), r))
	return r
}

type mockLikeChecker struct {
	likedIDs map[string]bool
	isLiked  bool
}

func (m *mockLikeChecker) IsLiked(_ context.Context, _, _ string, _ models.LikeTargetType) (bool, error) {
	return m.isLiked, nil
}

func (m *mockLikeChecker) GetLikedIDs(_ context.Context, _ string, _ models.LikeTargetType, targetIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(targetIDs))
	for _, id := range targetIDs {
		result[id] = m.likedIDs[id]
	}
	return result, nil
}

func setupRecipeService(t *testing.T, likeChecker recipes.LikeChecker) (*recipes.Service, *recipes.Repository) {
	database := testutil.SetupTestDB(t)
	repo := recipes.NewRepository(database, database)
	svc := recipes.NewService(repo, nil, likeChecker)
	return svc, repo
}

func TestRecipeService_GetRecipeByID(t *testing.T) {
	t.Run("returns recipe with is_liked", func(t *testing.T) {
		checker := &mockLikeChecker{isLiked: true}
		svc, repo := setupRecipeService(t, checker)
		r := createTestRecipe(t, repo)

		result, err := svc.GetRecipeByID(t.Context(), r.AuthorId, r.Id)
		require.NoError(t, err)
		assert.True(t, result.IsLiked)
		assert.Equal(t, r.Id, result.Id)
	})

	t.Run("returns not found", func(t *testing.T) {
		svc, _ := setupRecipeService(t, &mockLikeChecker{})

		_, err := svc.GetRecipeByID(t.Context(), "user-1", "non-existent")
		assert.ErrorIs(t, err, recipes.ErrNotFound)
	})

	t.Run("returns forbidden for private recipe", func(t *testing.T) {
		svc, repo := setupRecipeService(t, &mockLikeChecker{})
		private := createTestRecipe(t, repo)
		private.Public = false
		require.NoError(t, repo.UpsertRecipe(t.Context(), private))

		_, err := svc.GetRecipeByID(t.Context(), "other-user", private.Id)
		assert.ErrorIs(t, err, recipes.ErrForbidden)
	})
}

func TestRecipeService_ListRecipes(t *testing.T) {
	t.Run("sets is_liked for authenticated user", func(t *testing.T) {
		checker := &mockLikeChecker{likedIDs: map[string]bool{}}
		svc, repo := setupRecipeService(t, checker)

		r := createTestRecipe(t, repo)
		checker.likedIDs[r.Id] = true

		page, err := svc.ListRecipes(t.Context(), r.AuthorId, models.ListRecipesParams{})
		require.NoError(t, err)

		found := false
		for _, item := range page.Items {
			if item.Id == r.Id {
				assert.True(t, item.IsLiked)
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("is_liked false for unauthenticated", func(t *testing.T) {
		svc, repo := setupRecipeService(t, &mockLikeChecker{})
		createTestRecipe(t, repo)

		page, err := svc.ListRecipes(t.Context(), "", models.ListRecipesParams{})
		require.NoError(t, err)

		for _, item := range page.Items {
			assert.False(t, item.IsLiked)
		}
	})
}
