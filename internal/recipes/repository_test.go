package recipes_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/testutil"
)

func setupRecipeRepo(t *testing.T) *recipes.Repository {
	return recipes.NewRepository(testutil.SetupTestDB(t), slog.Default())
}

func defaultTestRecipe() models.Recipe {
	return models.Recipe{
		Id:          "recipe-1",
		AuthorId:    "user-1",
		Title:       "Test Recipe",
		Description: "Test description",
		BrewMethod:  models.V60,
		Difficulty:  models.DifficultyEasy,
		Public:      true,
		Steps:       []models.BrewStep{},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

func TestRecipeRepository_UpsertRecipe_Create(t *testing.T) {
	repo := setupRecipeRepo(t)
	r := defaultTestRecipe()

	require.NoError(t, repo.UpsertRecipe(t.Context(), r))

	got, err := repo.GetRecipeByID(t.Context(), r.Id)
	require.NoError(t, err)
	assert.Equal(t, r.Id, got.Id)
	assert.Equal(t, r.Title, got.Title)
	assert.Equal(t, r.AuthorId, got.AuthorId)
	assert.Equal(t, r.Public, got.Public)
}

func TestRecipeRepository_UpsertRecipe_Update(t *testing.T) {
	repo := setupRecipeRepo(t)
	r := defaultTestRecipe()
	require.NoError(t, repo.UpsertRecipe(t.Context(), r))

	r.Title = "Updated Title"
	require.NoError(t, repo.UpsertRecipe(t.Context(), r))

	got, err := repo.GetRecipeByID(t.Context(), r.Id)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", got.Title)
}

func TestRecipeRepository_GetRecipeByID_NotFound(t *testing.T) {
	repo := setupRecipeRepo(t)

	_, err := repo.GetRecipeByID(t.Context(), "non-existent")
	assert.ErrorIs(t, err, recipes.ErrNotFound)
}

func TestRecipeRepository_ListRecipes(t *testing.T) {
	repo := setupRecipeRepo(t)

	r1 := defaultTestRecipe()
	r2 := defaultTestRecipe()
	r2.Id = "recipe-2"
	r2.AuthorId = "user-2"
	r2.Public = false

	require.NoError(t, repo.UpsertRecipe(t.Context(), r1))
	require.NoError(t, repo.UpsertRecipe(t.Context(), r2))

	t.Run("returns public and own recipes", func(t *testing.T) {
		page, err := repo.ListRecipes(t.Context(), "user-2", models.ListRecipesParams{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(page.Items), 2)
	})

	t.Run("does not return other user private recipes", func(t *testing.T) {
		page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{})
		require.NoError(t, err)
		for _, item := range page.Items {
			if item.Id == r2.Id {
				t.Fatal("private recipe of another user should not be returned")
			}
		}
	})
}

func TestRecipeRepository_DeleteRecipe(t *testing.T) {
	repo := setupRecipeRepo(t)
	r := defaultTestRecipe()
	require.NoError(t, repo.UpsertRecipe(t.Context(), r))

	require.NoError(t, repo.DeleteRecipe(t.Context(), r.AuthorId, r.Id))

	_, err := repo.GetRecipeByID(t.Context(), r.Id)
	assert.ErrorIs(t, err, recipes.ErrNotFound)
}

func TestRecipeRepository_IncrementLikes(t *testing.T) {
	repo := setupRecipeRepo(t)
	r := defaultTestRecipe()
	require.NoError(t, repo.UpsertRecipe(t.Context(), r))

	count, err := repo.IncrementLikes(t.Context(), nil, r.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = repo.IncrementLikes(t.Context(), nil, r.Id)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestRecipeRepository_DecrementLikes(t *testing.T) {
	repo := setupRecipeRepo(t)
	r := defaultTestRecipe()
	require.NoError(t, repo.UpsertRecipe(t.Context(), r))

	_, err := repo.IncrementLikes(t.Context(), nil, r.Id)
	require.NoError(t, err)

	count, err := repo.DecrementLikes(t.Context(), nil, r.Id)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestRecipeRepository_DecrementLikes_NotBelowZero(t *testing.T) {
	repo := setupRecipeRepo(t)
	r := defaultTestRecipe()
	require.NoError(t, repo.UpsertRecipe(t.Context(), r))

	count, err := repo.DecrementLikes(t.Context(), nil, r.Id)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestRecipeRepository_IncrementLikes_NotFound(t *testing.T) {
	repo := setupRecipeRepo(t)

	_, err := repo.IncrementLikes(t.Context(), nil, "non-existent")
	assert.ErrorIs(t, err, likes.ErrTargetNotFound)
}
