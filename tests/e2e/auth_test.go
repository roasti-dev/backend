package e2e

import (
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

func randomCredentials() (string, string, string) {
	username := "user_" + randomString(8)
	email := username + "@test.com"
	password := "password123"
	return username, email, password
}

func TestRegister(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newTestClient(t, srv)
		username, email, password := randomCredentials()

		resp, err := c.PostApiV1AuthRegisterWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.NotEmpty(t, resp.JSON201.AccessToken)
		assert.NotEmpty(t, resp.JSON201.RefreshToken)
		assert.Equal(t, username, resp.JSON201.User.Username)
		assert.NotEmpty(t, resp.JSON201.User.Id)
	})

	t.Run("duplicate username", func(t *testing.T) {
		c := newTestClient(t, srv)
		username, email, password := randomCredentials()

		_, err := c.PostApiV1AuthRegisterWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		resp, err := c.PostApiV1AuthRegisterWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email("other_" + email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)
		assert.Equal(t, 409, resp.StatusCode())
	})

	t.Run("duplicate email", func(t *testing.T) {
		c := newTestClient(t, srv)
		username, email, password := randomCredentials()

		_, err := c.PostApiV1AuthRegisterWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		resp, err := c.PostApiV1AuthRegisterWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: "f_" + username,
			Password: password,
		})
		require.NoError(t, err)
		assert.Equal(t, 409, resp.StatusCode())
	})

	t.Run("weak password", func(t *testing.T) {
		c := newTestClient(t, srv)
		username, email, _ := randomCredentials()

		resp, err := c.PostApiV1AuthRegisterWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: "123",
		})
		require.NoError(t, err)
		assert.Equal(t, 422, resp.StatusCode())
	})

	t.Run("invalid username format", func(t *testing.T) {
		c := newTestClient(t, srv)
		_, email, password := randomCredentials()

		resp, err := c.PostApiV1AuthRegisterWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: "invalid username!",
			Password: password,
		})
		require.NoError(t, err)
		assert.Equal(t, 422, resp.StatusCode())
	})
}

func TestLogin(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newTestClient(t, srv)
		username, email, password := randomCredentials()

		_, err := c.PostApiV1AuthRegisterWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		resp, err := c.PostApiV1AuthLoginWithResponse(t.Context(), models.LoginRequest{
			Username: username,
			Password: password,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.NotEmpty(t, resp.JSON200.AccessToken)
		assert.NotEmpty(t, resp.JSON200.RefreshToken)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		c := newTestClient(t, srv)
		username, _, _ := randomCredentials()

		resp, err := c.PostApiV1AuthLoginWithResponse(t.Context(), models.LoginRequest{
			Username: username,
			Password: "wrongpassword",
		})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestRefresh(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newTestClient(t, srv)
		username, email, password := randomCredentials()

		reg, err := c.PostApiV1AuthRegisterWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		resp, err := c.PostApiV1AuthRefreshWithResponse(t.Context(), models.RefreshRequest{
			RefreshToken: reg.JSON201.RefreshToken,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.NotEmpty(t, resp.JSON200.AccessToken)
		assert.NotEmpty(t, resp.JSON200.RefreshToken)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		c := newTestClient(t, srv)

		resp, err := c.PostApiV1AuthRefreshWithResponse(t.Context(), models.RefreshRequest{
			RefreshToken: "invalid-token",
		})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestLogout(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.PostApiV1AuthLogoutWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})
}
