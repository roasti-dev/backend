package recipes_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
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

type mockLikeToggler struct {
	liked bool
	count int
}

func (m *mockLikeToggler) Toggle(_ context.Context, _, _ string, _ models.LikeTargetType) (likes.ToggleResult, error) {
	m.liked = !m.liked
	if m.liked {
		m.count++
	} else {
		m.count--
	}
	return likes.ToggleResult{Liked: m.liked, LikesCount: m.count}, nil
}

func setupRecipeService(t *testing.T) (*recipes.Service, *recipes.Repository) {
	database := testutil.SetupTestDB(t)
	testutil.CreateTestUser(t, database, "user-1")
	testutil.CreateTestUser(t, database, "user-2")
	repo := recipes.NewRepository(database, database)
	svc := recipes.NewService(repo, nil, &mockLikeChecker{likedIDs: make(map[string]bool)}, &mockLikeToggler{})
	return svc, repo
}

func TestRecipeService_GetRecipeByID(t *testing.T) {

	t.Run("returns not found", func(t *testing.T) {
		svc, _ := setupRecipeService(t)

		_, err := svc.GetRecipeByID(t.Context(), "user-1", "non-existent")
		assert.ErrorIs(t, err, recipes.ErrNotFound)
	})

	t.Run("returns forbidden for private recipe", func(t *testing.T) {
		svc, repo := setupRecipeService(t)
		private := createTestRecipe(t, repo)
		private.Public = false
		require.NoError(t, repo.UpsertRecipe(t.Context(), private))

		_, err := svc.GetRecipeByID(t.Context(), "other-user", private.Id)
		assert.ErrorIs(t, err, recipes.ErrForbidden)
	})
}

func TestRecipeService_CloneRecipe(t *testing.T) {
	t.Run("clones recipe", func(t *testing.T) {
		svc, repo := setupRecipeService(t)
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
		svc, repo := setupRecipeService(t)
		original := createTestRecipe(t, repo)

		_, err := svc.CloneRecipe(t.Context(), original.AuthorId, original.Id)
		assert.ErrorIs(t, err, recipes.ErrForbidden)
	})

	t.Run("cannot clone private recipe", func(t *testing.T) {
		svc, repo := setupRecipeService(t)
		private := createTestRecipe(t, repo)
		private.Public = false
		require.NoError(t, repo.UpsertRecipe(t.Context(), private))

		_, err := svc.CloneRecipe(t.Context(), "user-2", private.Id)
		assert.ErrorIs(t, err, recipes.ErrNotFound)
	})

	t.Run("returns not found for non-existent recipe", func(t *testing.T) {
		svc, _ := setupRecipeService(t)

		_, err := svc.CloneRecipe(t.Context(), "user-2", "non-existent")
		assert.ErrorIs(t, err, recipes.ErrNotFound)
	})

	t.Run("clones steps", func(t *testing.T) {
		svc, repo := setupRecipeService(t)
		original := createTestRecipe(t, repo)

		result, err := svc.CloneRecipe(t.Context(), "user-2", original.Id)
		require.NoError(t, err)
		assert.Len(t, result.Steps, len(original.Steps))
	})
}

func TestRecipeService_ToggleLike(t *testing.T) {
	t.Run("returns not found for non-existent recipe", func(t *testing.T) {
		svc, _ := setupRecipeService(t)

		_, err := svc.ToggleLike(t.Context(), "user-1", "non-existent")
		assert.ErrorIs(t, err, recipes.ErrNotFound)
	})

	t.Run("returns forbidden for private recipe of another user", func(t *testing.T) {
		svc, repo := setupRecipeService(t)
		r := createTestRecipe(t, repo)
		r.Public = false
		require.NoError(t, repo.UpsertRecipe(t.Context(), r))

		_, err := svc.ToggleLike(t.Context(), "user-2", r.Id)
		assert.ErrorIs(t, err, recipes.ErrForbidden)
	})

	t.Run("toggles like on accessible recipe", func(t *testing.T) {
		svc, repo := setupRecipeService(t)
		r := createTestRecipe(t, repo)

		result, err := svc.ToggleLike(t.Context(), "user-1", r.Id)
		require.NoError(t, err)
		assert.True(t, result.Liked)
		assert.Equal(t, 1, result.LikesCount)
	})

	t.Run("unlike removes like", func(t *testing.T) {
		svc, repo := setupRecipeService(t)
		r := createTestRecipe(t, repo)

		_, err := svc.ToggleLike(t.Context(), "user-1", r.Id)
		require.NoError(t, err)

		result, err := svc.ToggleLike(t.Context(), "user-1", r.Id)
		require.NoError(t, err)
		assert.False(t, result.Liked)
		assert.Equal(t, 0, result.LikesCount)
	})
}
