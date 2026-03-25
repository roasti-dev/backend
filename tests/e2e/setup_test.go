package e2e

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/app"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	a, err := app.New(t.Context(), app.Config{
		DBPath:                  ":memory:",
		UploadsPath:             t.TempDir(),
		AppVersion:              "test",
		FirebaseProjectID:       os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseAPIKey:          os.Getenv("FIREBASE_API_KEY"),
		FirebaseIdentityBaseURL: os.Getenv("FIREBASE_IDENTITY_BASE_URL"),
		FirebaseTokenBaseURL:    os.Getenv("FIREBASE_TOKEN_BASE_URL"),
	}, slog.Default())
	require.NoError(t, err)

	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)

	return srv
}

func newTestClient(t *testing.T, srv *httptest.Server) *client.ClientWithResponses {
	t.Helper()

	c, err := client.NewClientWithResponses(srv.URL)
	require.NoError(t, err)

	return c
}

type authenticatedClient struct {
	*client.ClientWithResponses

	ID           string
	AccessToken  string
	RefreshToken string
}

func newAuthenticatedTestClient(t *testing.T, srv *httptest.Server) *authenticatedClient {
	t.Helper()

	c := newTestClient(t, srv)

	username := "user_" + randomString(5)
	resp, err := c.RegisterUserWithResponse(t.Context(), models.RegisterRequest{
		Email:    openapi_types.Email(username + "@test.com"),
		Password: "password123",
		Username: username,
	})
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode())

	token := resp.JSON201.AccessToken

	authenticated, err := client.NewClientWithResponses(srv.URL,
		client.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}),
	)
	require.NoError(t, err)

	return &authenticatedClient{
		ClientWithResponses: authenticated,
		ID:                  resp.JSON201.User.Id,
		AccessToken:         resp.JSON201.AccessToken,
		RefreshToken:        resp.JSON201.RefreshToken,
	}
}

func newCookieClient(t *testing.T, srv *httptest.Server) *client.ClientWithResponses {
	t.Helper()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	httpClient := &http.Client{Jar: jar}

	c, err := client.NewClientWithResponses(srv.URL,
		client.WithHTTPClient(httpClient),
	)
	require.NoError(t, err)

	return c
}

func randomString(n int) string {
	return id.NewID()[:n]
}
