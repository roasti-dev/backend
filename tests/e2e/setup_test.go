package e2e

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/app"
	"github.com/nikpivkin/roasti-app-backend/internal/ids"
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
		FirebaseCredentialsJSON: os.Getenv("FIREBASE_CREDENTIALS_JSON"),
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

func newAuthenticatedTestClient(t *testing.T, srv *httptest.Server) *client.ClientWithResponses {
	t.Helper()

	c := newTestClient(t, srv)

	username := "user_" + randomString(5)
	resp, err := c.PostApiV1AuthRegisterWithResponse(t.Context(), models.RegisterRequest{
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

	return authenticated
}

func randomString(n int) string {
	return ids.NewID()[:n]
}
