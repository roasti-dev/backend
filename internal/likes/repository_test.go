package likes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
)

func TestLikeRepository_Create(t *testing.T) {
	repo := likes.NewRepository(testutil.SetupTestDB(t))

	like := likes.Like{
		ID:         "like-1",
		UserID:     "user-1",
		TargetID:   "recipe-1",
		TargetType: models.LikeTargetTypeRecipe,
	}

	err := repo.Create(t.Context(), like)
	require.NoError(t, err)

	exists, err := repo.Exists(t.Context(), "user-1", "recipe-1", models.LikeTargetTypeRecipe)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestLikeRepository_Create_Duplicate(t *testing.T) {
	repo := likes.NewRepository(testutil.SetupTestDB(t))

	like := likes.Like{
		ID:         "like-1",
		UserID:     "user-1",
		TargetID:   "recipe-1",
		TargetType: models.LikeTargetTypeRecipe,
	}

	require.NoError(t, repo.Create(t.Context(), like))
	require.NoError(t, repo.Create(t.Context(), like))

	exists, err := repo.Exists(t.Context(), "user-1", "recipe-1", models.LikeTargetTypeRecipe)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestLikeRepository_Delete(t *testing.T) {
	repo := likes.NewRepository(testutil.SetupTestDB(t))

	like := likes.Like{
		ID:         "like-1",
		UserID:     "user-1",
		TargetID:   "recipe-1",
		TargetType: models.LikeTargetTypeRecipe,
	}

	require.NoError(t, repo.Create(t.Context(), like))

	err := repo.Delete(t.Context(), "user-1", "recipe-1", models.LikeTargetTypeRecipe)
	require.NoError(t, err)

	exists, err := repo.Exists(t.Context(), "user-1", "recipe-1", models.LikeTargetTypeRecipe)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestLikeRepository_Exists_NotFound(t *testing.T) {
	repo := likes.NewRepository(testutil.SetupTestDB(t))

	exists, err := repo.Exists(t.Context(), "user-1", "recipe-1", models.LikeTargetTypeRecipe)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestLikeRepository_GetLikedIDs(t *testing.T) {
	repo := likes.NewRepository(testutil.SetupTestDB(t))

	require.NoError(t, repo.Create(t.Context(), likes.Like{ID: "like-1", UserID: "user-1", TargetID: "recipe-1", TargetType: models.LikeTargetTypeRecipe}))
	require.NoError(t, repo.Create(t.Context(), likes.Like{ID: "like-2", UserID: "user-1", TargetID: "recipe-2", TargetType: models.LikeTargetTypeRecipe}))
	require.NoError(t, repo.Create(t.Context(), likes.Like{ID: "like-3", UserID: "user-2", TargetID: "recipe-1", TargetType: models.LikeTargetTypeRecipe}))

	result, err := repo.GetLikedIDs(t.Context(), "user-1", models.LikeTargetTypeRecipe, []string{"recipe-1", "recipe-2", "recipe-3"})
	require.NoError(t, err)
	assert.True(t, result["recipe-1"])
	assert.True(t, result["recipe-2"])
	assert.False(t, result["recipe-3"])
}

func TestLikeRepository_GetLikedIDs_OtherUser(t *testing.T) {
	repo := likes.NewRepository(testutil.SetupTestDB(t))

	require.NoError(t, repo.Create(t.Context(), likes.Like{ID: "like-1", UserID: "user-1", TargetID: "recipe-1", TargetType: models.LikeTargetTypeRecipe}))

	result, err := repo.GetLikedIDs(t.Context(), "user-2", models.LikeTargetTypeRecipe, []string{"recipe-1"})
	require.NoError(t, err)
	assert.False(t, result["recipe-1"])
}
