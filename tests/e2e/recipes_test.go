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
	Title:       "V60 Recipe",
	Description: "A great V60 recipe",
	BrewMethod:  models.V60,
	Difficulty:  models.DifficultyEasy,
	Steps: []models.BrewStepPayload{
		{Order: 1, Title: "Boil water"},
	},
}

func createRecipe(t *testing.T, c *authenticatedClient, payload models.RecipePayload) *models.Recipe {
	t.Helper()
	resp, err := c.PostApiV1RecipesWithResponse(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode())
	return resp.JSON201
}

func TestCreateRecipe(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.PostApiV1RecipesWithResponse(t.Context(), defaultPayload)
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
		resp, err := c.PostApiV1RecipesWithResponse(t.Context(), payload)
		require.NoError(t, err)
		assert.Equal(t, 422, resp.StatusCode())
	})

	t.Run("empty description", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		payload := defaultPayload
		payload.Description = ""
		resp, err := c.PostApiV1RecipesWithResponse(t.Context(), payload)
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

		resp, err := c.PutApiV1RecipesRecipeIdWithResponse(t.Context(), recipe.Id, updated)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, "Updated Title", resp.JSON200.Title)
	})

	t.Run("forbidden - not author", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		other := newAuthenticatedTestClient(t, srv)
		resp, err := other.PutApiV1RecipesRecipeIdWithResponse(t.Context(), recipe.Id, defaultPayload)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("not found", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.PutApiV1RecipesRecipeIdWithResponse(t.Context(), "non-existent-id", defaultPayload)
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})
}

func TestDeleteRecipe(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		resp, err := c.DeleteApiV1RecipesRecipeIdWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})

	t.Run("forbidden - not author", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		other := newAuthenticatedTestClient(t, srv)
		resp, err := other.DeleteApiV1RecipesRecipeIdWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})
}

func TestGetRecipeByID(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy - your own recipe", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		resp, err := c.GetApiV1RecipesRecipeIdWithResponse(t.Context(), recipe.Id)
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
		resp, err := c2.GetApiV1RecipesRecipeIdWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.NotEmpty(t, resp.JSON200.AuthorId)
		assert.Equal(t, recipe.Id, resp.JSON200.Id)
	})

	t.Run("not found - recipe not found ", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.GetApiV1RecipesRecipeIdWithResponse(t.Context(), ids.NewID())
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("forbidden - another user's private recipe", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		private := defaultPayload
		private.Public = new(false)
		recipe := createRecipe(t, c1, private)

		c2 := newAuthenticatedTestClient(t, srv)
		resp, err := c2.GetApiV1RecipesRecipeIdWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})
}

func TestListRecipes(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns own and public recipes", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)

		public := defaultPayload
		public.Public = new(true)
		createRecipe(t, c1, public)

		private := defaultPayload
		private.Public = new(false)
		createRecipe(t, c2, private)

		resp, err := c2.GetApiV1RecipesWithResponse(t.Context(), &client.GetApiV1RecipesParams{})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.GreaterOrEqual(t, resp.JSON200.Pagination.ItemsCount, int32(2))
	})

	t.Run("filter by brew method", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.GetApiV1RecipesWithResponse(t.Context(), &client.GetApiV1RecipesParams{
			ListRecipes: &models.ListRecipesParams{
				BrewMethod: new(models.V60),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		for _, r := range resp.JSON200.Items {
			assert.Equal(t, models.V60, r.BrewMethod)
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

		resp, err := c.GetApiV1UploadsImagesImageIdWithResponse(context.Background(), imageID)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
	})
}
