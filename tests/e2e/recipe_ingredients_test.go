package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

func TestRecipeIngredients(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("recipe without ingredients returns empty array", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)

		resp, err := c.GetRecipeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.NotNil(t, resp.JSON200.Ingredients)
		assert.Empty(t, resp.JSON200.Ingredients)
	})

	t.Run("create recipe with ingredients", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		payload := defaultPayload
		payload.Ingredients = &[]models.RecipeIngredientPayload{
			{Name: "Coffee", Amount: new(float32(15)), Unit: new("g")},
			{Name: "Water", Amount: new(float32(250)), Unit: new("ml")},
			{Name: "Salt"},
		}
		recipe := createRecipe(t, c, payload)

		require.Len(t, recipe.Ingredients, 3)
		assert.Equal(t, "Coffee", recipe.Ingredients[0].Name)
		assert.Equal(t, float32(15), *recipe.Ingredients[0].Amount)
		assert.Equal(t, "g", *recipe.Ingredients[0].Unit)
		assert.Equal(t, "Water", recipe.Ingredients[1].Name)
		assert.Equal(t, "Salt", recipe.Ingredients[2].Name)
		assert.Nil(t, recipe.Ingredients[2].Amount)
		assert.Nil(t, recipe.Ingredients[2].Unit)
	})

	t.Run("order is preserved", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		payload := defaultPayload
		payload.Ingredients = &[]models.RecipeIngredientPayload{
			{Name: "Third"},
			{Name: "First"},
			{Name: "Second"},
		}
		recipe := createRecipe(t, c, payload)

		require.Len(t, recipe.Ingredients, 3)
		assert.Equal(t, "Third", recipe.Ingredients[0].Name)
		assert.Equal(t, "First", recipe.Ingredients[1].Name)
		assert.Equal(t, "Second", recipe.Ingredients[2].Name)
	})

	t.Run("update replaces ingredients", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		payload := defaultPayload
		payload.Ingredients = &[]models.RecipeIngredientPayload{
			{Name: "Old ingredient"},
		}
		recipe := createRecipe(t, c, payload)

		updated := defaultPayload
		updated.Ingredients = &[]models.RecipeIngredientPayload{
			{Name: "New ingredient 1"},
			{Name: "New ingredient 2"},
		}
		resp, err := c.UpdateRecipeWithResponse(t.Context(), recipe.Id, updated)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, resp.JSON200.Ingredients, 2)
		assert.Equal(t, "New ingredient 1", resp.JSON200.Ingredients[0].Name)
		assert.Equal(t, "New ingredient 2", resp.JSON200.Ingredients[1].Name)
	})

	t.Run("update with empty array clears ingredients", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		payload := defaultPayload
		payload.Ingredients = &[]models.RecipeIngredientPayload{
			{Name: "Some ingredient"},
		}
		recipe := createRecipe(t, c, payload)

		cleared := defaultPayload
		cleared.Ingredients = &[]models.RecipeIngredientPayload{}
		resp, err := c.UpdateRecipeWithResponse(t.Context(), recipe.Id, cleared)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Ingredients)
	})

	t.Run("ingredients visible in listing", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		payload := defaultPayload
		payload.Ingredients = &[]models.RecipeIngredientPayload{
			{Name: "Espresso", Amount: new(float32(30)), Unit: new("ml")},
		}
		recipe := createRecipe(t, c, payload)

		listResp, err := c.ListRecipesWithResponse(t.Context(), &client.ListRecipesParams{})
		require.NoError(t, err)
		require.Equal(t, 200, listResp.StatusCode())

		var found *models.Recipe
		for i := range listResp.JSON200.Items {
			if listResp.JSON200.Items[i].Id == recipe.Id {
				found = &listResp.JSON200.Items[i]
				break
			}
		}
		require.NotNil(t, found)
		require.Len(t, found.Ingredients, 1)
		assert.Equal(t, "Espresso", found.Ingredients[0].Name)
	})

	t.Run("invalid: empty ingredient name returns 422", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		payload := defaultPayload
		payload.Ingredients = &[]models.RecipeIngredientPayload{
			{Name: ""},
		}
		resp, err := c.CreateRecipeWithResponse(t.Context(), payload)
		require.NoError(t, err)
		assert.Equal(t, 422, resp.StatusCode())
	})

	t.Run("invalid: negative amount returns 422", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		payload := defaultPayload
		payload.Ingredients = &[]models.RecipeIngredientPayload{
			{Name: "Coffee", Amount: new(float32(-5))},
		}
		resp, err := c.CreateRecipeWithResponse(t.Context(), payload)
		require.NoError(t, err)
		assert.Equal(t, 422, resp.StatusCode())
	})
}
