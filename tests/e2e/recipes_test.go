package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/ids"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

var defaultPayload = models.RecipePayload{
	Title:       "Test Recipe",
	Description: "Test description",
	BrewMethod:  models.V60,
	Difficulty:  models.DifficultyEasy,
	Public:      new(true),
	Steps: []models.BrewStepPayload{
		{Order: 1, Title: "Boil water"},
	},
}

func createRecipe(t *testing.T, c *authenticatedClient, payload models.RecipePayload) *models.Recipe {
	t.Helper()
	resp, err := c.CreateRecipeWithResponse(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode())
	return resp.JSON201
}

func toggleRecipeLike(t *testing.T, c *authenticatedClient, recipeID string) {
	t.Helper()
	resp, err := c.ToggleRecipeLikeWithResponse(t.Context(), recipeID)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode())
}

func TestCreateRecipe(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.CreateRecipeWithResponse(t.Context(), defaultPayload)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.Equal(t, defaultPayload.Title, resp.JSON201.Title)
		assert.NotEmpty(t, resp.JSON201.AuthorId)
		assert.NotEmpty(t, resp.JSON201.Id)
	})

	t.Run("empty title", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		payload := defaultPayload
		payload.Title = ""
		resp, err := c.CreateRecipeWithResponse(t.Context(), payload)
		require.NoError(t, err)
		assert.Equal(t, 422, resp.StatusCode())
	})

	t.Run("empty description", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		payload := defaultPayload
		payload.Description = ""
		resp, err := c.CreateRecipeWithResponse(t.Context(), payload)
		require.NoError(t, err)
		assert.Equal(t, 422, resp.StatusCode())
	})
}

func TestUpdateRecipe(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		updated := defaultPayload
		updated.Title = "Updated Title"

		resp, err := c.UpdateRecipeWithResponse(t.Context(), recipe.Id, updated)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, "Updated Title", resp.JSON200.Title)
	})

	t.Run("forbidden - not author", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		other := newAuthenticatedTestClient(t, srv)
		resp, err := other.UpdateRecipeWithResponse(t.Context(), recipe.Id, defaultPayload)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("not found", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.UpdateRecipeWithResponse(t.Context(), "non-existent-id", defaultPayload)
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})
}

func TestDeleteRecipe(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		resp, err := c.DeleteRecipeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})

	t.Run("forbidden - not author", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		other := newAuthenticatedTestClient(t, srv)
		resp, err := other.DeleteRecipeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})
}

func TestGetRecipeByID(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy - your own recipe", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		resp, err := c.GetRecipeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.NotEmpty(t, resp.JSON200.AuthorId)
		assert.Equal(t, recipe.Id, resp.JSON200.Id)
	})

	t.Run("happy - another user's public recipe", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		public := defaultPayload
		public.Public = new(true)
		recipe := createRecipe(t, c1, public)

		c2 := newAuthenticatedTestClient(t, srv)
		resp, err := c2.GetRecipeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.NotEmpty(t, resp.JSON200.AuthorId)
		assert.Equal(t, recipe.Id, resp.JSON200.Id)
	})

	t.Run("not found - recipe not found ", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.GetRecipeWithResponse(t.Context(), ids.NewID())
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("forbidden - another user's private recipe", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		private := defaultPayload
		private.Public = new(false)
		recipe := createRecipe(t, c1, private)

		c2 := newAuthenticatedTestClient(t, srv)
		resp, err := c2.GetRecipeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("recipe contains author", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		resp, err := c.GetRecipeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, recipe.AuthorId, resp.JSON200.Author.Id)
		assert.NotEmpty(t, resp.JSON200.Author.Username)
	})
}

func TestListRecipes(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns only public recipes", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)

		public := defaultPayload
		createRecipe(t, c1, public)

		private := defaultPayload
		private.Public = new(false)
		createRecipe(t, c2, private)

		resp, err := c2.ListRecipesWithResponse(t.Context(), &client.ListRecipesParams{})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		for _, r := range resp.JSON200.Items {
			assert.True(t, r.Public)
		}
	})

	t.Run("filter by brew method", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.ListRecipesWithResponse(t.Context(), &client.ListRecipesParams{
			BrewMethod: new(models.V60),
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		for _, r := range resp.JSON200.Items {
			assert.Equal(t, models.V60, r.BrewMethod)
		}
	})

	t.Run("filter by query", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		p1 := defaultPayload
		p1.Title = "V60 Recipe"
		p1.Description = "Test description"

		p2 := defaultPayload
		p2.Title = "Aeropress Recipe"
		p2.Description = "Test description"

		createRecipe(t, c, p1)
		createRecipe(t, c, p2)

		t.Run("matches title", func(t *testing.T) {
			resp, err := c.ListRecipesWithResponse(t.Context(), &client.ListRecipesParams{
				Query: new("V60"),
			})
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode())
			for _, r := range resp.JSON200.Items {
				assert.Contains(t, r.Title, "V60")
			}
		})

		t.Run("returns empty for no match", func(t *testing.T) {
			resp, err := c.ListRecipesWithResponse(t.Context(), &client.ListRecipesParams{
				Query: new("nomatch"),
			})
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode())
			assert.Empty(t, resp.JSON200.Items)
		})
	})

	t.Run("list recipes contains author", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		createRecipe(t, c, defaultPayload)

		resp, err := c.ListRecipesWithResponse(t.Context(), &client.ListRecipesParams{})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		for _, r := range resp.JSON200.Items {
			assert.NotEmpty(t, r.Author.Id)
			assert.NotEmpty(t, r.Author.Username)
		}
	})

}

func TestRecipeWithImage(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("create recipe with image", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		imageID := uploadImage(t, c, generateTestImage(t))

		payload := defaultPayload
		payload.ImageId = &imageID

		recipe := createRecipe(t, c, payload)
		assert.Equal(t, &imageID, recipe.ImageId)

		resp, err := c.GetImageWithResponse(context.Background(), imageID)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
	})
}

func TestToggleRecipeLike(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("like a recipe", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		resp, err := c.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.True(t, resp.JSON200.Liked)
		assert.Equal(t, int32(1), resp.JSON200.LikesCount)
	})

	t.Run("unlike a recipe", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		toggleRecipeLike(t, c, recipe.Id)

		resp, err := c.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.False(t, resp.JSON200.Liked)
		assert.Equal(t, int32(0), resp.JSON200.LikesCount)
	})

	t.Run("two users like same recipe", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c1, defaultPayload)

		toggleRecipeLike(t, c1, recipe.Id)

		resp, err := c2.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.True(t, resp.JSON200.Liked)
		assert.Equal(t, int32(2), resp.JSON200.LikesCount)
	})

	t.Run("like does not affect other recipes", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		r1 := createRecipe(t, c, defaultPayload)
		r2 := createRecipe(t, c, defaultPayload)

		toggleRecipeLike(t, c, r1.Id)

		resp, err := c.ToggleRecipeLikeWithResponse(t.Context(), r2.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.True(t, resp.JSON200.Liked)
		assert.Equal(t, int32(1), resp.JSON200.LikesCount)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.ToggleRecipeLikeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})

	t.Run("non-existent recipe returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.ToggleRecipeLikeWithResponse(t.Context(), ids.NewID())
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})
}

func TestCloneRecipe(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("clones recipe", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		original := createRecipe(t, c1, defaultPayload)

		resp, err := c2.CloneRecipeWithResponse(t.Context(), original.Id)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.NotEqual(t, original.Id, resp.JSON201.Id)
		assert.Equal(t, "Copy of "+original.Title, resp.JSON201.Title)
		assert.NotNil(t, resp.JSON201.Origin)
		assert.Equal(t, original.Id, resp.JSON201.Origin.RecipeId)
	})

	t.Run("cannot clone own recipe", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		resp, err := c.CloneRecipeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("cannot clone private recipe", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)

		private := defaultPayload
		private.Public = new(false)
		recipe := createRecipe(t, c1, private)

		resp, err := c2.CloneRecipeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("clone appears in author likes", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		original := createRecipe(t, c1, defaultPayload)

		resp, err := c2.CloneRecipeWithResponse(t.Context(), original.Id)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())

		list, err := c2.ListRecipesWithResponse(t.Context(), &client.ListRecipesParams{})
		require.NoError(t, err)
		assert.Equal(t, 200, list.StatusCode())
		var found bool
		for _, r := range list.JSON200.Items {
			if r.Id == resp.JSON201.Id {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		unauth := newTestClient(t, srv)
		resp, err := unauth.CloneRecipeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})

	t.Run("non-existent recipe returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.CloneRecipeWithResponse(t.Context(), ids.NewID())
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})
}
