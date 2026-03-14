package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

var defaultPayload = models.RecipePayload{
	Title:       "V60 Recipe",
	Description: "A great V60 recipe",
	BrewMethod:  models.V60,
	Difficulty:  models.DifficultyEasy,
	Steps: []models.BrewStepPayload{
		{Order: 1, Title: "Boil water", Description: "Boil water to 92°C"},
	},
}

func createRecipe(t *testing.T, c *client.ClientWithResponses, payload models.RecipePayload) *models.Recipe {
	t.Helper()
	resp, err := c.PostApiV1RecipesWithResponse(t.Context(), nil, payload)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode())
	return resp.JSON201
}

func TestCreateRecipe(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newTestClient(t, srv, "user-1")
		resp, err := c.PostApiV1RecipesWithResponse(t.Context(), nil, defaultPayload)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.Equal(t, defaultPayload.Title, resp.JSON201.Title)
		assert.Equal(t, "user-1", resp.JSON201.AuthorId)
		assert.NotEmpty(t, resp.JSON201.Id)
	})

	t.Run("empty title", func(t *testing.T) {
		c := newTestClient(t, srv, "user-1")
		payload := defaultPayload
		payload.Title = ""
		resp, err := c.PostApiV1RecipesWithResponse(t.Context(), nil, payload)
		require.NoError(t, err)
		assert.Equal(t, 422, resp.StatusCode())
	})

	t.Run("empty description", func(t *testing.T) {
		c := newTestClient(t, srv, "user-1")
		payload := defaultPayload
		payload.Description = ""
		resp, err := c.PostApiV1RecipesWithResponse(t.Context(), nil, payload)
		require.NoError(t, err)
		assert.Equal(t, 422, resp.StatusCode())
	})
}

func TestUpdateRecipe(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newTestClient(t, srv, "user-1")
		recipe := createRecipe(t, c, defaultPayload)

		updated := defaultPayload
		updated.Title = "Updated Title"

		resp, err := c.PutApiV1RecipesRecipeIdWithResponse(t.Context(), recipe.Id, nil, updated)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, "Updated Title", resp.JSON200.Title)
	})

	t.Run("forbidden - not author", func(t *testing.T) {
		c := newTestClient(t, srv, "user-1")
		recipe := createRecipe(t, c, defaultPayload)

		other := newTestClient(t, srv, "user-2")
		resp, err := other.PutApiV1RecipesRecipeIdWithResponse(t.Context(), recipe.Id, nil, defaultPayload)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("not found", func(t *testing.T) {
		c := newTestClient(t, srv, "user-1")
		resp, err := c.PutApiV1RecipesRecipeIdWithResponse(t.Context(), "non-existent-id", nil, defaultPayload)
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})
}

func TestDeleteRecipe(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newTestClient(t, srv, "user-1")
		recipe := createRecipe(t, c, defaultPayload)

		resp, err := c.DeleteApiV1RecipesRecipeIdWithResponse(t.Context(), recipe.Id, nil)
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})

	t.Run("forbidden - not author", func(t *testing.T) {
		c := newTestClient(t, srv, "user-1")
		recipe := createRecipe(t, c, defaultPayload)

		other := newTestClient(t, srv, "user-2")
		resp, err := other.DeleteApiV1RecipesRecipeIdWithResponse(t.Context(), recipe.Id, nil)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})
}

func TestListRecipes(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns own and public recipes", func(t *testing.T) {
		c1 := newTestClient(t, srv, "user-1")
		c2 := newTestClient(t, srv, "user-2")

		public := defaultPayload
		public.Public = new(true)
		createRecipe(t, c1, public)

		private := defaultPayload
		private.Public = new(false)
		createRecipe(t, c2, private)

		resp, err := c2.GetApiV1RecipesWithResponse(t.Context(), &client.GetApiV1RecipesParams{})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.GreaterOrEqual(t, resp.JSON200.TotalCount, 2)
	})

	t.Run("filter by brew method", func(t *testing.T) {
		c := newTestClient(t, srv, "user-1")
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
		c := newTestClient(t, srv, "user-1")
		imageID := uploadImage(t, c, generateTestImage(t))

		payload := defaultPayload
		payload.ImageId = &imageID

		recipe := createRecipe(t, c, payload)
		assert.Equal(t, &imageID, recipe.ImageId)

		resp, err := c.GetApiV1UploadsImagesImageIdWithResponse(context.Background(), imageID, nil)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
	})
}
