package e2e

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/app"
	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	a, err := app.New(app.Config{
		DBPath:      ":memory:",
		UploadsPath: t.TempDir(),
		AppVersion:  "test",
	}, slog.Default())
	require.NoError(t, err)

	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)

	return srv
}

func newTestClient(t *testing.T, srv *httptest.Server, userID string) *client.ClientWithResponses {
	t.Helper()

	c, err := client.NewClientWithResponses(srv.URL,
		client.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("X-User-ID", userID)
			return nil
		}),
	)
	require.NoError(t, err)

	return c
}
