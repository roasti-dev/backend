package users_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
)

func TestLikeRepository_ListByUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	likesRepo := likes.NewRepository(db)
	recipesRepo := recipes.NewRepository(db, db)

	testutil.CreateTestRecipe(t, recipesRepo, "recipe-1", "user-1")
	testutil.CreateTestRecipe(t, recipesRepo, "recipe-2", "user-1")
	testutil.CreateTestLike(t, likesRepo, "user-1", "recipe-1", models.LikeTargetTypeRecipe)
	testutil.CreateTestLike(t, likesRepo, "user-1", "recipe-2", models.LikeTargetTypeRecipe)

	t.Run("returns liked recipes", func(t *testing.T) {
		result, err := likesRepo.ListByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe, 10, 0)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("respects limit", func(t *testing.T) {
		result, err := likesRepo.ListByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe, 1, 0)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("respects offset", func(t *testing.T) {
		result, err := likesRepo.ListByUser(t.Context(), "user-1", models.LikeTargetTypeRecipe, 10, 1)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("returns empty for unknown user", func(t *testing.T) {
		result, err := likesRepo.ListByUser(t.Context(), "unknown", models.LikeTargetTypeRecipe, 10, 0)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}
