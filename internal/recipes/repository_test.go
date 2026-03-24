package recipes_test

import (
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
	database := testutil.SetupTestDB(t)
	testutil.CreateTestUser(t, database, "user-1")
	testutil.CreateTestUser(t, database, "user-2")
	return recipes.NewRepository(database, database)
}

func TestRecipeRepository_UpsertRecipe_Create(t *testing.T) {
	repo := setupRecipeRepo(t)
	r := models.Recipe{
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
	r := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")

	r.Title = "Updated Title"
	require.NoError(t, repo.UpsertRecipe(t.Context(), r))

	got, err := repo.GetRecipeByID(t.Context(), r.Id)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", got.Title)
}

func TestRecipeRepository_Note(t *testing.T) {
	t.Run("persists note", func(t *testing.T) {
		repo := setupRecipeRepo(t)
		r := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
		note := "my private note"
		r.Note = &note

		require.NoError(t, repo.UpsertRecipe(t.Context(), r))

		got, err := repo.GetRecipeByID(t.Context(), r.Id)
		require.NoError(t, err)
		require.NotNil(t, got.Note)
		assert.Equal(t, note, *got.Note)
		assert.Equal(t, r.Id, got.Id)
	})

	t.Run("nil note is persisted as null", func(t *testing.T) {
		repo := setupRecipeRepo(t)
		r := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
		r.Note = nil

		require.NoError(t, repo.UpsertRecipe(t.Context(), r))

		got, err := repo.GetRecipeByID(t.Context(), r.Id)
		require.NoError(t, err)
		assert.Nil(t, got.Note)
	})

	t.Run("update clears note", func(t *testing.T) {
		repo := setupRecipeRepo(t)
		r := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
		note := "my private note"
		r.Note = &note
		require.NoError(t, repo.UpsertRecipe(t.Context(), r))

		r.Note = nil
		require.NoError(t, repo.UpsertRecipe(t.Context(), r))

		got, err := repo.GetRecipeByID(t.Context(), r.Id)
		require.NoError(t, err)
		assert.Nil(t, got.Note)
	})

	t.Run("note is returned in list", func(t *testing.T) {
		repo := setupRecipeRepo(t)
		r := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
		note := "my private note"
		r.Note = &note
		require.NoError(t, repo.UpsertRecipe(t.Context(), r))

		page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		require.NotNil(t, page.Items[0].Note)
		assert.Equal(t, note, *page.Items[0].Note)
	})
}

func TestRecipeRepository_GetRecipeByID_NotFound(t *testing.T) {
	repo := setupRecipeRepo(t)

	_, err := repo.GetRecipeByID(t.Context(), "non-existent")
	assert.ErrorIs(t, err, recipes.ErrNotFound)
}

func TestRecipeRepository_ListRecipes(t *testing.T) {
	repo := setupRecipeRepo(t)

	r1 := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
	r2 := testutil.CreateTestRecipe(t, repo, "recipe-2", "user-2")
	r2.Public = false
	require.NoError(t, repo.UpsertRecipe(t.Context(), r2))

	t.Run("returns only public recipes when no author filter", func(t *testing.T) {
		page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{})
		require.NoError(t, err)
		for _, item := range page.Items {
			assert.True(t, item.Public)
		}
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

	t.Run("filters by query", func(t *testing.T) {
		repo := setupRecipeRepo(t)

		r1 := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
		r1.Title = "V60 Recipe"
		r1.Description = "A great recipe"
		require.NoError(t, repo.UpsertRecipe(t.Context(), r1))

		r2 := testutil.CreateTestRecipe(t, repo, "recipe-2", "user-1")
		r2.Title = "Aeropress Recipe"
		r2.Description = "Quick and easy"
		require.NoError(t, repo.UpsertRecipe(t.Context(), r2))

		t.Run("matches title", func(t *testing.T) {
			q := "V60"
			page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{Query: &q})
			require.NoError(t, err)
			assert.Len(t, page.Items, 1)
			assert.Equal(t, r1.Id, page.Items[0].Id)
		})

		t.Run("matches description", func(t *testing.T) {
			q := "Quick"
			page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{Query: &q})
			require.NoError(t, err)
			assert.Len(t, page.Items, 1)
			assert.Equal(t, r2.Id, page.Items[0].Id)
		})

		t.Run("case insensitive", func(t *testing.T) {
			q := "v60"
			page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{Query: &q})
			require.NoError(t, err)
			assert.Len(t, page.Items, 1)
		})

		t.Run("returns empty for no match", func(t *testing.T) {
			q := "nomatch"
			page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{Query: &q})
			require.NoError(t, err)
			assert.Empty(t, page.Items)
		})
	})

	t.Run("filters by author", func(t *testing.T) {
		repo := setupRecipeRepo(t)

		r1 := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
		r2 := testutil.CreateTestRecipe(t, repo, "recipe-2", "user-1")
		r2.Public = false
		require.NoError(t, repo.UpsertRecipe(t.Context(), r2))
		testutil.CreateTestRecipe(t, repo, "recipe-3", "user-2")

		t.Run("returns only author recipes", func(t *testing.T) {
			page, err := repo.ListRecipes(t.Context(), "user-3", models.ListRecipesParams{AuthorId: &r1.AuthorId})
			require.NoError(t, err)
			for _, item := range page.Items {
				assert.Equal(t, "user-1", item.Author.Id)
			}
		})

		t.Run("owner sees own private recipes", func(t *testing.T) {
			page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{AuthorId: &r1.AuthorId})
			require.NoError(t, err)
			assert.Len(t, page.Items, 2)
		})

		t.Run("other user does not see private recipes of author", func(t *testing.T) {
			page, err := repo.ListRecipes(t.Context(), "user-3", models.ListRecipesParams{AuthorId: &r1.AuthorId})
			require.NoError(t, err)
			assert.Len(t, page.Items, 1)
			assert.Equal(t, r1.Id, page.Items[0].Id)
		})
	})

	t.Run("filters by brew method", func(t *testing.T) {
		repo := setupRecipeRepo(t)

		r1 := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
		r2 := testutil.CreateTestRecipe(t, repo, "recipe-2", "user-1")
		r2.BrewMethod = models.Aeropress
		require.NoError(t, repo.UpsertRecipe(t.Context(), r2))

		page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{BrewMethod: &r1.BrewMethod})
		require.NoError(t, err)
		assert.Len(t, page.Items, 1)
		assert.Equal(t, r1.Id, page.Items[0].Id)
	})

	t.Run("filters by difficulty", func(t *testing.T) {
		repo := setupRecipeRepo(t)

		r1 := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
		r2 := testutil.CreateTestRecipe(t, repo, "recipe-2", "user-1")
		r2.Difficulty = models.DifficultyHard
		require.NoError(t, repo.UpsertRecipe(t.Context(), r2))

		page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{Difficulty: &r1.Difficulty})
		require.NoError(t, err)
		assert.Len(t, page.Items, 1)
		assert.Equal(t, r1.Id, page.Items[0].Id)
	})

	t.Run("populates author", func(t *testing.T) {
		page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{})
		require.NoError(t, err)
		require.NotEmpty(t, page.Items)
		assert.Equal(t, r1.AuthorId, page.Items[0].Author.Id)
	})
}

func TestRecipeRepository_ListRecipes_FilterByRoastLevel(t *testing.T) {
	repo := setupRecipeRepo(t)

	r1 := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
	r1.RoastLevel = new(models.RoastLevelLight)
	require.NoError(t, repo.UpsertRecipe(t.Context(), r1))

	r2 := testutil.CreateTestRecipe(t, repo, "recipe-2", "user-1")
	r2.RoastLevel = new(models.RoastLevelDark)
	require.NoError(t, repo.UpsertRecipe(t.Context(), r2))

	t.Run("returns only matching roast level", func(t *testing.T) {
		page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{
			RoastLevel: new(models.RoastLevelLight)},
		)
		require.NoError(t, err)
		assert.Len(t, page.Items, 1)
		assert.Equal(t, r1.Id, page.Items[0].Id)
	})

	t.Run("returns empty when no match", func(t *testing.T) {
		page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{
			RoastLevel: new(models.RoastLevelMedium)},
		)
		require.NoError(t, err)
		assert.Empty(t, page.Items)
	})

	t.Run("returns all when no filter", func(t *testing.T) {
		page, err := repo.ListRecipes(t.Context(), "user-1", models.ListRecipesParams{})
		require.NoError(t, err)
		assert.Len(t, page.Items, 2)
	})
}

func TestRecipeRepository_DeleteRecipe(t *testing.T) {
	repo := setupRecipeRepo(t)
	r := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")

	require.NoError(t, repo.DeleteRecipe(t.Context(), r.AuthorId, r.Id))

	_, err := repo.GetRecipeByID(t.Context(), r.Id)
	assert.ErrorIs(t, err, recipes.ErrNotFound)
}

func TestRecipeRepository_IncrementLikes(t *testing.T) {
	repo := setupRecipeRepo(t)
	r := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")

	count, err := repo.IncrementLikes(t.Context(), nil, r.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = repo.IncrementLikes(t.Context(), nil, r.Id)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestRecipeRepository_DecrementLikes(t *testing.T) {
	repo := setupRecipeRepo(t)
	r := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")

	_, err := repo.IncrementLikes(t.Context(), nil, r.Id)
	require.NoError(t, err)

	count, err := repo.DecrementLikes(t.Context(), nil, r.Id)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestRecipeRepository_DecrementLikes_NotBelowZero(t *testing.T) {
	repo := setupRecipeRepo(t)
	r := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")

	count, err := repo.DecrementLikes(t.Context(), nil, r.Id)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestRecipeRepository_IncrementLikes_NotFound(t *testing.T) {
	repo := setupRecipeRepo(t)

	_, err := repo.IncrementLikes(t.Context(), nil, "non-existent")
	assert.ErrorIs(t, err, likes.ErrTargetNotFound)
}

func TestRecipeRepository_GetPreviewsByIDs(t *testing.T) {
	repo := setupRecipeRepo(t)

	r1 := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
	r2 := testutil.CreateTestRecipe(t, repo, "recipe-2", "user-2")
	r2.Public = false
	require.NoError(t, repo.UpsertRecipe(t.Context(), r2))

	ids := []string{r1.Id, r2.Id}

	t.Run("returns public recipes", func(t *testing.T) {
		previews, err := repo.GetPreviewsByIDs(t.Context(), "user-1", ids)
		require.NoError(t, err)
		assert.Len(t, previews, 1)
		assert.Equal(t, r1.Id, previews[0].Id)
	})

	t.Run("owner sees own private recipes", func(t *testing.T) {
		previews, err := repo.GetPreviewsByIDs(t.Context(), "user-2", ids)
		require.NoError(t, err)
		assert.Len(t, previews, 2)
	})

	t.Run("does not return other user private recipes", func(t *testing.T) {
		previews, err := repo.GetPreviewsByIDs(t.Context(), "user-1", ids)
		require.NoError(t, err)
		for _, p := range previews {
			assert.NotEqual(t, r2.Id, p.Id)
		}
	})

	t.Run("returns empty for unknown ids", func(t *testing.T) {
		previews, err := repo.GetPreviewsByIDs(t.Context(), "user-1", []string{"unknown"})
		require.NoError(t, err)
		assert.Empty(t, previews)
	})
}

func TestRecipeRepository_GetRecipesByIDs(t *testing.T) {
	repo := setupRecipeRepo(t)

	r1 := testutil.CreateTestRecipe(t, repo, "recipe-1", "user-1")
	r2 := testutil.CreateTestRecipe(t, repo, "recipe-2", "user-2")
	r2.Public = false
	require.NoError(t, repo.UpsertRecipe(t.Context(), r2))

	ids := []string{r1.Id, r2.Id}

	t.Run("returns public recipes", func(t *testing.T) {
		recipes, err := repo.GetRecipesByIDs(t.Context(), "user-1", ids)
		require.NoError(t, err)
		assert.Len(t, recipes, 1)
		assert.Equal(t, r1.Id, recipes[0].Id)
	})

	t.Run("owner sees own private recipes", func(t *testing.T) {
		recipes, err := repo.GetRecipesByIDs(t.Context(), "user-2", ids)
		require.NoError(t, err)
		assert.Len(t, recipes, 2)
	})

	t.Run("does not return other user private recipes", func(t *testing.T) {
		recipes, err := repo.GetRecipesByIDs(t.Context(), "user-1", ids)
		require.NoError(t, err)
		for _, r := range recipes {
			assert.NotEqual(t, r2.Id, r.Id)
		}
	})

	t.Run("returns empty for unknown ids", func(t *testing.T) {
		recipes, err := repo.GetRecipesByIDs(t.Context(), "user-1", []string{"unknown"})
		require.NoError(t, err)
		assert.Empty(t, recipes)
	})
}
