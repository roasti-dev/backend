package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

func TestListMyLikes(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns liked recipes", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		r1 := createRecipe(t, c, defaultPayload)
		r2 := createRecipe(t, c, defaultPayload)

		toggleRecipeLike(t, c, r1.Id)
		toggleRecipeLike(t, c, r2.Id)

		resp, err := c.ListMyLikesWithResponse(t.Context(), &client.ListMyLikesParams{
			ListMyLikes: &models.ListMyLikesParams{
				Type: "recipe",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 2)
	})

	t.Run("does not return unliked recipes", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		r1 := createRecipe(t, c, defaultPayload)

		toggleRecipeLike(t, c, r1.Id)
		toggleRecipeLike(t, c, r1.Id) // unlike

		resp, err := c.ListMyLikesWithResponse(t.Context(), &client.ListMyLikesParams{
			ListMyLikes: &models.ListMyLikesParams{
				Type: "recipe",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
	})

	t.Run("does not return other user likes", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c1, defaultPayload)

		toggleRecipeLike(t, c1, recipe.Id)

		resp, err := c2.ListMyLikesWithResponse(t.Context(), &client.ListMyLikesParams{
			ListMyLikes: &models.ListMyLikesParams{
				Type: "recipe",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
	})

	t.Run("liked_at is returned", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		toggleRecipeLike(t, c, recipe.Id)

		resp, err := c.ListMyLikesWithResponse(t.Context(), &client.ListMyLikesParams{
			ListMyLikes: &models.ListMyLikesParams{
				Type: "recipe",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.NotZero(t, resp.JSON200.Items[0].LikedAt)
	})

	t.Run("respects pagination", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		r1 := createRecipe(t, c, defaultPayload)
		r2 := createRecipe(t, c, defaultPayload)

		toggleRecipeLike(t, c, r1.Id)
		toggleRecipeLike(t, c, r2.Id)

		resp, err := c.ListMyLikesWithResponse(t.Context(), &client.ListMyLikesParams{
			ListMyLikes: &models.ListMyLikesParams{
				Type:  "recipe",
				Limit: new(models.LimitParam(1)),
				Page:  new(models.PageParam(1)),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 1)
		assert.Equal(t, int32(1), resp.JSON200.Pagination.ItemsCount)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newTestClient(t, srv)

		resp, err := c.ListMyLikesWithResponse(t.Context(), &client.ListMyLikesParams{
			ListMyLikes: &models.ListMyLikesParams{
				Type: "recipe",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}
