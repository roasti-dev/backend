package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

func TestListUserLikes(t *testing.T) {
	srv := setupTestServer(t)

	listLikes := func(t *testing.T, c *authenticatedClient, userID string) *client.ListUserLikesResponse {
		t.Helper()
		resp, err := c.ListUserLikesWithResponse(t.Context(), userID, &client.ListUserLikesParams{
			Type: "recipe",
		})
		require.NoError(t, err)
		return resp
	}

	t.Run("returns liked recipes", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		r1 := createRecipe(t, c, defaultPayload)
		r2 := createRecipe(t, c, defaultPayload)

		toggleRecipeLike(t, c, r1.Id)
		toggleRecipeLike(t, c, r2.Id)

		resp := listLikes(t, c, c.ID)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 2)
	})

	t.Run("does not return unliked recipes", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		r1 := createRecipe(t, c, defaultPayload)

		toggleRecipeLike(t, c, r1.Id)
		toggleRecipeLike(t, c, r1.Id) // unlike

		resp := listLikes(t, c, c.ID)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
	})

	t.Run("does not return other user likes", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c1, defaultPayload)

		toggleRecipeLike(t, c1, recipe.Id)

		resp := listLikes(t, c2, c2.ID)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
	})

	t.Run("can view other user likes", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c1, defaultPayload)

		toggleRecipeLike(t, c1, recipe.Id)

		resp := listLikes(t, c2, c1.ID)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 1)
	})

	t.Run("does not expose private recipes to other users", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)

		privatePayload := defaultPayload
		privatePayload.Public = new(false)
		recipe := createRecipe(t, c1, privatePayload)

		toggleRecipeLike(t, c1, recipe.Id)

		resp := listLikes(t, c2, c1.ID)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
	})

	t.Run("liked_at is returned", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		toggleRecipeLike(t, c, recipe.Id)

		resp := listLikes(t, c, c.ID)
		assert.Equal(t, 200, resp.StatusCode())
		assert.NotZero(t, resp.JSON200.Items[0].LikedAt)
	})

	t.Run("respects pagination", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		r1 := createRecipe(t, c, defaultPayload)
		r2 := createRecipe(t, c, defaultPayload)

		toggleRecipeLike(t, c, r1.Id)
		toggleRecipeLike(t, c, r2.Id)

		resp, err := c.ListUserLikesWithResponse(t.Context(), c.ID, &client.ListUserLikesParams{
			Type:  "recipe",
			Limit: new(models.LimitParam(1)),
			Page:  new(models.PageParam(1)),
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 1)
	})

	t.Run("unknown user returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp := listLikes(t, c, "unknown-user-id")
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		unauth := newTestClient(t, srv)

		resp, err := unauth.ListUserLikesWithResponse(t.Context(), c1.ID, &client.ListUserLikesParams{
			Type: "recipe",
		})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})

	t.Run("liked recipes contain author", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)
		toggleRecipeLike(t, c, recipe.Id)

		resp, err := c.ListUserLikesWithResponse(t.Context(), c.ID, &client.ListUserLikesParams{
			Type: "recipe",
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.NotEmpty(t, resp.JSON200.Items[0].Recipe.Author.Id)
		assert.NotEmpty(t, resp.JSON200.Items[0].Recipe.Author.Username)
	})
}
