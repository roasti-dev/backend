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

		resp, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
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
		assert.Equal(t, openapi_types.Email(email), resp.JSON201.User.Email)
	})

	t.Run("duplicate username", func(t *testing.T) {
		c := newTestClient(t, srv)
		username, email, password := randomCredentials()

		_, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		resp, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
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

		_, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		resp, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
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

		resp, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
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

		resp, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
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

		_, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		resp, err := c.LoginUserWithResponse(t.Context(), models.LoginRequest{
			Username: username,
			Password: password,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.NotEmpty(t, resp.JSON200.AccessToken)
		assert.NotEmpty(t, resp.JSON200.RefreshToken)
		assert.Equal(t, openapi_types.Email(email), resp.JSON200.User.Email)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		c := newTestClient(t, srv)
		username, _, _ := randomCredentials()

		resp, err := c.LoginUserWithResponse(t.Context(), models.LoginRequest{
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

		reg, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		resp, err := c.RefreshTokenWithResponse(t.Context(), models.RefreshRequest{
			RefreshToken: reg.JSON201.RefreshToken,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.NotEmpty(t, resp.JSON200.AccessToken)
		assert.NotEmpty(t, resp.JSON200.RefreshToken)
	})

	t.Run("refresh token rotates and new token is usable", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		first, err := c.RefreshTokenWithResponse(t.Context(), models.RefreshRequest{
			RefreshToken: c.RefreshToken,
		})
		require.NoError(t, err)
		require.Equal(t, 200, first.StatusCode())

		// NOTE: In the Firebase production environment, the token rotates, but the emulator returns the same one
		// assert.NotEqual(t, c.RefreshToken, first.JSON200.RefreshToken)

		second, err := c.RefreshTokenWithResponse(t.Context(), models.RefreshRequest{
			RefreshToken: first.JSON200.RefreshToken,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, second.StatusCode())
		assert.NotEmpty(t, second.JSON200.AccessToken)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		c := newTestClient(t, srv)

		resp, err := c.RefreshTokenWithResponse(t.Context(), models.RefreshRequest{
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

		resp, err := c.LogoutUserWithResponse(t.Context(), models.LogoutRequest{
			RefreshToken: c.RefreshToken,
		})
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())
	})

	t.Run("double logout is idempotent", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		for range 2 {
			resp, err := c.LogoutUserWithResponse(t.Context(), models.LogoutRequest{
				RefreshToken: c.RefreshToken,
			})
			require.NoError(t, err)
			assert.Equal(t, 204, resp.StatusCode())
		}
	})

	t.Run("revoked token cannot be used to refresh", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		_, err := c.LogoutUserWithResponse(t.Context(), models.LogoutRequest{
			RefreshToken: c.RefreshToken,
		})
		require.NoError(t, err)

		refresh, err := c.RefreshTokenWithResponse(t.Context(), models.RefreshRequest{
			RefreshToken: c.RefreshToken,
		})
		require.NoError(t, err)
		assert.Equal(t, 401, refresh.StatusCode())
	})
}

func TestChangePassword(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("changes password successfully", func(t *testing.T) {
		c := newTestClient(t, srv)
		username, email, password := randomCredentials()

		_, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		authC := newAuthenticatedTestClient(t, srv)
		newPassword := "newpassword456"

		resp, err := authC.ChangePasswordWithResponse(t.Context(), models.ChangePasswordRequest{
			CurrentPassword: "password123",
			NewPassword:     newPassword,
		})
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode())

		// old password no longer works
		loginResp, err := c.LoginUserWithResponse(t.Context(), models.LoginRequest{
			Username: authC.Username,
			Password: "password123",
		})
		require.NoError(t, err)
		assert.Equal(t, 401, loginResp.StatusCode())

		// new password works
		loginResp, err = c.LoginUserWithResponse(t.Context(), models.LoginRequest{
			Username: authC.Username,
			Password: newPassword,
		})
		require.NoError(t, err)
		assert.Equal(t, 200, loginResp.StatusCode())
	})

	t.Run("returns 401 for incorrect current password", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.ChangePasswordWithResponse(t.Context(), models.ChangePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword456",
		})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})

	t.Run("returns 422 for weak new password", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)

		resp, err := c.ChangePasswordWithResponse(t.Context(), models.ChangePasswordRequest{
			CurrentPassword: "password123",
			NewPassword:     "123",
		})
		require.NoError(t, err)
		assert.Equal(t, 422, resp.StatusCode())
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		c := newTestClient(t, srv)

		resp, err := c.ChangePasswordWithResponse(t.Context(), models.ChangePasswordRequest{
			CurrentPassword: "password123",
			NewPassword:     "newpassword456",
		})
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode())
	})
}

func TestCookieAuth(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("login sets cookies and cookie can be used for authenticated request", func(t *testing.T) {
		c := newCookieClient(t, srv)
		username, email, password := randomCredentials()

		_, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		meResp, err := c.GetCurrentUserWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 200, meResp.StatusCode())
	})

	t.Run("logout clears cookies and authenticated request fails", func(t *testing.T) {
		c := newCookieClient(t, srv)
		username, email, password := randomCredentials()

		_, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		logoutResp, err := c.LogoutUserWithResponse(t.Context(), models.LogoutRequest{})
		require.NoError(t, err)
		require.Equal(t, 204, logoutResp.StatusCode())

		meResp, err := c.GetCurrentUserWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 401, meResp.StatusCode())
	})

	t.Run("refresh via cookie", func(t *testing.T) {
		c := newCookieClient(t, srv)
		username, email, password := randomCredentials()

		_, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
			Email:    openapi_types.Email(email),
			Username: username,
			Password: password,
		})
		require.NoError(t, err)

		refreshResp, err := c.RefreshTokenWithResponse(t.Context(), models.RefreshRequest{})
		require.NoError(t, err)
		require.Equal(t, 200, refreshResp.StatusCode())
		assert.NotEmpty(t, refreshResp.JSON200.AccessToken)

		meResp, err := c.GetCurrentUserWithResponse(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 200, meResp.StatusCode())
	})
}
