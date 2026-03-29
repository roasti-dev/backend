package e2e

import (
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

func TestUpdateCurrentUser(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("updates username", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		newUsername := "updated_" + randomString(5)

		resp, err := c.UpdateCurrentUserWithResponse(t.Context(), client.UpdateCurrentUserJSONRequestBody{
			Username: &newUsername,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, newUsername, resp.JSON200.Username)
	})

	t.Run("updates bio", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bio := "my bio"

		resp, err := c.UpdateCurrentUserWithResponse(t.Context(), client.UpdateCurrentUserJSONRequestBody{
			Bio: nullable.NewNullableWithValue(bio),
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, &bio, resp.JSON200.Bio)
	})

	t.Run("partial update does not change other fields", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		// Get original username
		meResp, err := c.GetCurrentUserWithResponse(t.Context())
		require.NoError(t, err)
		originalUsername := meResp.JSON200.Username

		bio := "partial update bio"
		resp, err := c.UpdateCurrentUserWithResponse(t.Context(), client.UpdateCurrentUserJSONRequestBody{
			Bio: nullable.NewNullableWithValue(bio),
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, originalUsername, resp.JSON200.Username)
		assert.Equal(t, &bio, resp.JSON200.Bio)
	})

	t.Run("clears avatar when avatarId is null", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		avatarID := uploadImage(t, c, generateTestImage(t))

		setResp, err := c.UpdateCurrentUserWithResponse(t.Context(), client.UpdateCurrentUserJSONRequestBody{
			AvatarId: nullable.NewNullableWithValue(avatarID),
		})
		require.NoError(t, err)
		require.Equal(t, 200, setResp.StatusCode())
		assert.Equal(t, &avatarID, setResp.JSON200.AvatarId)

		clearResp, err := c.UpdateCurrentUserWithResponse(t.Context(), client.UpdateCurrentUserJSONRequestBody{
			AvatarId: nullable.NewNullNullable[string](),
		})
		require.NoError(t, err)
		require.Equal(t, 200, clearResp.StatusCode())
		assert.Nil(t, clearResp.JSON200.AvatarId)
	})

	t.Run("returns 409 when username already taken", func(t *testing.T) {
		c1 := newAuthenticatedTestClient(t, srv)
		c2 := newAuthenticatedTestClient(t, srv)

		// Get c1's username
		meResp, err := c1.GetCurrentUserWithResponse(t.Context())
		require.NoError(t, err)
		takenUsername := meResp.JSON200.Username

		resp, err := c2.UpdateCurrentUserWithResponse(t.Context(), client.UpdateCurrentUserJSONRequestBody{
			Username: &takenUsername,
		})
		require.NoError(t, err)
		assert.Equal(t, 409, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		unauth := newTestClient(t, srv)
		bio := "bio"

		resp, err := unauth.UpdateCurrentUserWithResponse(t.Context(), client.UpdateCurrentUserJSONRequestBody{
			Bio: nullable.NewNullableWithValue(bio),
		})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestGetCurrentUser(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns current user profile", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.GetCurrentUserWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, c.ID, resp.JSON200.Id)
		assert.NotEmpty(t, resp.JSON200.Username)
		assert.NotEmpty(t, resp.JSON200.Email)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		unauth := newTestClient(t, srv)

		resp, err := unauth.GetCurrentUserWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestGetUserProfile(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("returns public profile by user id", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		meResp, err := c.GetCurrentUserWithResponse(t.Context())
		require.NoError(t, err)
		userID := meResp.JSON200.Id

		anon := newTestClient(t, srv)
		resp, err := anon.GetUserProfileWithResponse(t.Context(), userID)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, userID, resp.JSON200.Id)
		assert.Equal(t, meResp.JSON200.Username, resp.JSON200.Username)
	})

	t.Run("does not expose email", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		meResp, err := c.GetCurrentUserWithResponse(t.Context())
		require.NoError(t, err)

		anon := newTestClient(t, srv)
		resp, err := anon.GetUserProfileWithResponse(t.Context(), meResp.JSON200.Id)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())

		// UserProfile must not have an email field — verify raw JSON
		assert.NotContains(t, string(resp.Body), "email")
	})

	t.Run("returns bio when set", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		bio := "my public bio"

		_, err := c.UpdateCurrentUserWithResponse(t.Context(), client.UpdateCurrentUserJSONRequestBody{
			Bio: nullable.NewNullableWithValue(bio),
		})
		require.NoError(t, err)

		meResp, err := c.GetCurrentUserWithResponse(t.Context())
		require.NoError(t, err)

		anon := newTestClient(t, srv)
		resp, err := anon.GetUserProfileWithResponse(t.Context(), meResp.JSON200.Id)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, &bio, resp.JSON200.Bio)
	})

	t.Run("returns 404 for unknown user id", func(t *testing.T) {
		anon := newTestClient(t, srv)
		resp, err := anon.GetUserProfileWithResponse(t.Context(), "nonexistent-user-id")
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})
}

func TestCheckUsernameAvailability(t *testing.T) {
	srv := setupTestServer(t)

	check := func(t *testing.T, username string) *client.CheckUsernameAvailabilityResponse {
		t.Helper()
		c := newTestClient(t, srv)
		resp, err := c.CheckUsernameAvailabilityWithResponse(t.Context(), &client.CheckUsernameAvailabilityParams{
			Username: username,
		})
		require.NoError(t, err)
		return resp
	}

	t.Run("returns available for unused username", func(t *testing.T) {
		resp := check(t, "never_taken_"+randomString(8))
		assert.Equal(t, 200, resp.StatusCode())
		assert.True(t, resp.JSON200.Available)
	})

	t.Run("returns unavailable for taken username", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		meResp, err := c.GetCurrentUserWithResponse(t.Context())
		require.NoError(t, err)
		takenUsername := meResp.JSON200.Username

		resp := check(t, takenUsername)
		assert.Equal(t, 200, resp.StatusCode())
		assert.False(t, resp.JSON200.Available)
	})
}

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
