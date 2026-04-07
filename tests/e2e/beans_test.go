package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

var defaultBeanPayload = models.BeanPayload{
	Name:      "Ethiopia Yirgacheffe",
	RoastType: models.BeanRoastTypeFilter,
	Roaster:   "Nordic Roasters",
}

func createBean(t *testing.T, c *authenticatedClient, payload models.BeanPayload) *models.Bean {
	t.Helper()
	resp, err := c.CreateBeanWithResponse(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode())
	return resp.JSON201
}

func TestCreateBean(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path with required fields", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.CreateBeanWithResponse(t.Context(), defaultBeanPayload)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())

		bean := resp.JSON201
		assert.NotEmpty(t, bean.Id)
		assert.Equal(t, defaultBeanPayload.Name, bean.Name)
		assert.Equal(t, defaultBeanPayload.RoastType, bean.RoastType)
		assert.Equal(t, defaultBeanPayload.Roaster, bean.Roaster)
		assert.Equal(t, c.Username, bean.Author.Username)
		assert.NotZero(t, bean.CreatedAt)
	})

	t.Run("with all optional fields", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		country := "Ethiopia"
		region := "Yirgacheffe"
		farm := "Konga"
		process := "Washed"
		url := "https://example.com/bean"
		qScore := float32(89.5)
		descriptors := []string{"floral", "citrus", "blueberry"}

		payload := models.BeanPayload{
			Name:        "Full Fields Bean",
			RoastType:   models.BeanRoastTypeEspresso,
			Roaster:     "Specialty Co",
			Country:     &country,
			Region:      &region,
			Farm:        &farm,
			Process:     &process,
			Url:         &url,
			QScore:      &qScore,
			Descriptors: &descriptors,
		}
		resp, err := c.CreateBeanWithResponse(t.Context(), payload)
		require.NoError(t, err)
		require.Equal(t, 201, resp.StatusCode())

		bean := resp.JSON201
		assert.Equal(t, country, *bean.Country)
		assert.Equal(t, region, *bean.Region)
		assert.Equal(t, farm, *bean.Farm)
		assert.Equal(t, process, *bean.Process)
		assert.Equal(t, url, *bean.Url)
		assert.InDelta(t, qScore, *bean.QScore, 0.01)
		assert.Equal(t, descriptors, *bean.Descriptors)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newTestClient(t, srv)
		resp, err := c.CreateBeanWithResponse(t.Context(), defaultBeanPayload)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestGetBean(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns bean by id", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createBean(t, c, defaultBeanPayload)

		resp, err := c.GetBeanWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, created.Id, resp.JSON200.Id)
		assert.Equal(t, defaultBeanPayload.Name, resp.JSON200.Name)
	})

	t.Run("non-existent returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.GetBeanWithResponse(t.Context(), "non-existent-id")
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		created := createBean(t, c1, defaultBeanPayload)

		c := newTestClient(t, srv)
		resp, err := c.GetBeanWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestListBeans(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns empty list", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.ListBeansWithResponse(t.Context(), &client.ListBeansParams{})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Empty(t, resp.JSON200.Items)
		assert.Equal(t, int32(0), resp.JSON200.Pagination.ItemsCount)
	})

	t.Run("returns created beans", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		roaster := "UniqueRoaster-" + randomString(5)
		createBean(t, c, models.BeanPayload{Name: "Bean One", RoastType: models.BeanRoastTypeFilter, Roaster: roaster})
		createBean(t, c, models.BeanPayload{Name: "Bean Two", RoastType: models.BeanRoastTypeEspresso, Roaster: roaster})

		q := roaster
		resp, err := c.ListBeansWithResponse(t.Context(), &client.ListBeansParams{Q: &q})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 2)
	})

	t.Run("search by name", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		createBean(t, c, models.BeanPayload{Name: "Kenya AA", RoastType: models.BeanRoastTypeFilter, Roaster: "Roaster One"})
		createBean(t, c, models.BeanPayload{Name: "Brazil Santos", RoastType: models.BeanRoastTypeEspresso, Roaster: "Roaster Two"})

		q := "kenya"
		resp, err := c.ListBeansWithResponse(t.Context(), &client.ListBeansParams{Q: &q})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, resp.JSON200.Items, 1)
		assert.Equal(t, "Kenya AA", resp.JSON200.Items[0].Name)
	})

	t.Run("search by roaster", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		createBean(t, c, models.BeanPayload{Name: "Bean A", RoastType: models.BeanRoastTypeFilter, Roaster: "Square Mile"})
		createBean(t, c, models.BeanPayload{Name: "Bean B", RoastType: models.BeanRoastTypeFilter, Roaster: "Ozone Coffee"})

		q := "square mile"
		resp, err := c.ListBeansWithResponse(t.Context(), &client.ListBeansParams{Q: &q})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.Len(t, resp.JSON200.Items, 1)
		assert.Equal(t, "Square Mile", resp.JSON200.Items[0].Roaster)
	})

	t.Run("respects pagination", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		roaster := "PaginationRoaster-" + randomString(5)
		for i := range 3 {
			createBean(t, c, models.BeanPayload{
				Name:      "Paginated Bean " + string(rune('A'+i)),
				RoastType: models.BeanRoastTypeFilter,
				Roaster:   roaster,
			})
		}

		limit := int32(2)
		resp, err := c.ListBeansWithResponse(t.Context(), &client.ListBeansParams{Limit: &limit, Q: &roaster})
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Len(t, resp.JSON200.Items, 2)
		assert.Equal(t, int32(2), resp.JSON200.Pagination.LastPage)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newTestClient(t, srv)
		resp, err := c.ListBeansWithResponse(t.Context(), &client.ListBeansParams{})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestUpdateBean(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createBean(t, c, defaultBeanPayload)

		updated := models.BeanPayload{
			Name:      "Updated Bean Name",
			RoastType: models.BeanRoastTypeEspresso,
			Roaster:   "New Roaster",
		}
		resp, err := c.UpdateBeanWithResponse(t.Context(), created.Id, updated)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, updated.Name, resp.JSON200.Name)
		assert.Equal(t, updated.RoastType, resp.JSON200.RoastType)
		assert.Equal(t, updated.Roaster, resp.JSON200.Roaster)
	})

	t.Run("non-owner gets 403", func(t *testing.T) {
		owner := newAuthenticatedTestClient(t, srv)
		other := newAuthenticatedTestClient(t, srv)
		created := createBean(t, owner, defaultBeanPayload)

		resp, err := other.UpdateBeanWithResponse(t.Context(), created.Id, defaultBeanPayload)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("non-existent returns 404", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.UpdateBeanWithResponse(t.Context(), "non-existent-id", defaultBeanPayload)
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		created := createBean(t, c1, defaultBeanPayload)

		c := newTestClient(t, srv)
		resp, err := c.UpdateBeanWithResponse(t.Context(), created.Id, defaultBeanPayload)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestDeleteBean(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path returns 204", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createBean(t, c, defaultBeanPayload)

		resp, err := c.DeleteBeanWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})

	t.Run("deleted bean not returned in list", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createBean(t, c, defaultBeanPayload)

		_, err := c.DeleteBeanWithResponse(t.Context(), created.Id)
		require.NoError(t, err)

		resp, err := c.ListBeansWithResponse(t.Context(), &client.ListBeansParams{})
		require.NoError(t, err)
		for _, b := range resp.JSON200.Items {
			assert.NotEqual(t, created.Id, b.Id)
		}
	})

	t.Run("deleted bean not returned by get", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createBean(t, c, defaultBeanPayload)

		_, err := c.DeleteBeanWithResponse(t.Context(), created.Id)
		require.NoError(t, err)

		resp, err := c.GetBeanWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})

	t.Run("idempotent — second delete returns 204", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		created := createBean(t, c, defaultBeanPayload)

		_, err := c.DeleteBeanWithResponse(t.Context(), created.Id)
		require.NoError(t, err)

		resp, err := c.DeleteBeanWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})

	t.Run("non-owner gets 403", func(t *testing.T) {
		owner := newAuthenticatedTestClient(t, srv)
		other := newAuthenticatedTestClient(t, srv)
		created := createBean(t, owner, defaultBeanPayload)

		resp, err := other.DeleteBeanWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		created := createBean(t, c1, defaultBeanPayload)

		c := newTestClient(t, srv)
		resp, err := c.DeleteBeanWithResponse(t.Context(), created.Id)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestBeanInRecipe(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("recipe with bean returns populated bean ref", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)

		payload := defaultPayload
		payload.BeanId = &bean.Id
		recipe := createRecipe(t, c, payload)

		require.NotNil(t, recipe.Bean)
		assert.Equal(t, bean.Id, recipe.Bean.Id)
		assert.Equal(t, bean.Name, recipe.Bean.Name)
		assert.Equal(t, bean.Roaster, recipe.Bean.Roaster)
		assert.Equal(t, bean.RoastType, recipe.Bean.RoastType)
		assert.Equal(t, models.BeanRefStatusAvailable, recipe.Bean.Status)
	})

	t.Run("recipe without bean has nil bean", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		recipe := createRecipe(t, c, defaultPayload)
		assert.Nil(t, recipe.Bean)
	})

	t.Run("deleted bean shows as unavailable in recipe", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)

		payload := defaultPayload
		payload.BeanId = &bean.Id
		recipe := createRecipe(t, c, payload)

		_, err := c.DeleteBeanWithResponse(t.Context(), bean.Id)
		require.NoError(t, err)

		resp, err := c.GetRecipeWithResponse(t.Context(), recipe.Id)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotNil(t, resp.JSON200.Bean)
		assert.Equal(t, bean.Id, resp.JSON200.Bean.Id)
		assert.Equal(t, models.BeanRefStatusUnavailable, resp.JSON200.Bean.Status)
	})

	t.Run("can attach bean via update", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)
		recipe := createRecipe(t, c, defaultPayload)
		assert.Nil(t, recipe.Bean)

		updatedPayload := defaultPayload
		updatedPayload.BeanId = &bean.Id
		resp, err := c.UpdateRecipeWithResponse(t.Context(), recipe.Id, updatedPayload)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		require.NotNil(t, resp.JSON200.Bean)
		assert.Equal(t, bean.Id, resp.JSON200.Bean.Id)
	})

	t.Run("can detach bean by setting bean_id to null", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bean := createBean(t, c, defaultBeanPayload)

		payload := defaultPayload
		payload.BeanId = &bean.Id
		recipe := createRecipe(t, c, payload)
		require.NotNil(t, recipe.Bean)

		detachPayload := defaultPayload
		detachPayload.BeanId = nil
		resp, err := c.UpdateRecipeWithResponse(t.Context(), recipe.Id, detachPayload)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())
		assert.Nil(t, resp.JSON200.Bean)
	})
}
