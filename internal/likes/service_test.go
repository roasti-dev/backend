package likes_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/recipe"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
)

func setupLikeService(t *testing.T) (*likes.Service, *recipe.Repository) {
	database := testutil.SetupTestDB(t)
	likeRepo := likes.NewRepository(database)
	recipeRepo := recipe.NewRepository(database, slog.Default())
	svc := likes.NewService(database, likeRepo, recipeRepo)
	return svc, recipeRepo
}

func createTestRecipe(t *testing.T, repo *recipe.Repository) models.Recipe {
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

func TestLikeService_Toggle_Like(t *testing.T) {
	svc, recipeRepo := setupLikeService(t)
	r := createTestRecipe(t, recipeRepo)

	result, err := svc.Toggle(t.Context(), "user-1", r.Id, models.LikeTargetTypeRecipe)
	require.NoError(t, err)
	assert.True(t, result.Liked)
	assert.Equal(t, 1, result.LikesCount)
}

func TestLikeService_Toggle_Unlike(t *testing.T) {
	svc, recipeRepo := setupLikeService(t)
	r := createTestRecipe(t, recipeRepo)

	_, err := svc.Toggle(t.Context(), "user-1", r.Id, models.LikeTargetTypeRecipe)
	require.NoError(t, err)

	result, err := svc.Toggle(t.Context(), "user-1", r.Id, models.LikeTargetTypeRecipe)
	require.NoError(t, err)
	assert.False(t, result.Liked)
	assert.Equal(t, 0, result.LikesCount)
}

func TestLikeService_Toggle_MultipleUsers(t *testing.T) {
	svc, recipeRepo := setupLikeService(t)
	r := createTestRecipe(t, recipeRepo)

	_, err := svc.Toggle(t.Context(), "user-1", r.Id, models.LikeTargetTypeRecipe)
	require.NoError(t, err)

	result, err := svc.Toggle(t.Context(), "user-2", r.Id, models.LikeTargetTypeRecipe)
	require.NoError(t, err)
	assert.True(t, result.Liked)
	assert.Equal(t, 2, result.LikesCount)
}

func TestLikeService_Toggle_RecipeNotFound(t *testing.T) {
	svc, _ := setupLikeService(t)

	_, err := svc.Toggle(t.Context(), "user-1", "non-existent", models.LikeTargetTypeRecipe)
	assert.ErrorIs(t, err, likes.ErrTargetNotFound)
}
