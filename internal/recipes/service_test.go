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

func (m *mockLikeChecker) CountByTarget(_ context.Context, _ string, _ models.LikeTargetType) (int, error) {
	return 0, nil
}

func (m *mockLikeChecker) CountByTargets(_ context.Context, targetIDs []string, _ models.LikeTargetType) (map[string]int, error) {
	return make(map[string]int, len(targetIDs)), nil
}

func setupRecipeService(t *testing.T, likeChecker recipes.LikeChecker) (*recipes.Service, *recipes.Repository) {
	database := testutil.SetupTestDB(t)
	testutil.CreateTestUser(t, database, "user-1")
	testutil.CreateTestUser(t, database, "user-2")
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

func TestRecipeService_CloneRecipe(t *testing.T) {
	t.Run("clones recipe", func(t *testing.T) {
		svc, repo := setupRecipeService(t, &mockLikeChecker{})
		original := createTestRecipe(t, repo)

		result, err := svc.CloneRecipe(t.Context(), "user-2", original.Id)
		require.NoError(t, err)
		assert.NotEqual(t, original.Id, result.Id)
		assert.Equal(t, "user-2", result.AuthorId)
		assert.Equal(t, "Copy of "+original.Title, result.Title)
		assert.NotNil(t, result.Origin)
		assert.Equal(t, original.Id, result.Origin.RecipeId)
	})

	t.Run("cannot clone own recipe", func(t *testing.T) {
		svc, repo := setupRecipeService(t, &mockLikeChecker{})
		original := createTestRecipe(t, repo)

		_, err := svc.CloneRecipe(t.Context(), original.AuthorId, original.Id)
		assert.ErrorIs(t, err, recipes.ErrForbidden)
	})

	t.Run("cannot clone private recipe", func(t *testing.T) {
		svc, repo := setupRecipeService(t, &mockLikeChecker{})
		private := createTestRecipe(t, repo)
		private.Public = false
		require.NoError(t, repo.UpsertRecipe(t.Context(), private))

		_, err := svc.CloneRecipe(t.Context(), "user-2", private.Id)
		assert.ErrorIs(t, err, recipes.ErrNotFound)
	})

	t.Run("returns not found for non-existent recipe", func(t *testing.T) {
		svc, _ := setupRecipeService(t, &mockLikeChecker{})

		_, err := svc.CloneRecipe(t.Context(), "user-2", "non-existent")
		assert.ErrorIs(t, err, recipes.ErrNotFound)
	})

	t.Run("clones steps", func(t *testing.T) {
		svc, repo := setupRecipeService(t, &mockLikeChecker{})
		original := createTestRecipe(t, repo)

		result, err := svc.CloneRecipe(t.Context(), "user-2", original.Id)
		require.NoError(t, err)
		assert.Len(t, result.Steps, len(original.Steps))
	})
}
