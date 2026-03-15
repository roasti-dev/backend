package e2e

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/tests/client"
)

func generateTestImage(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	img.Set(50, 50, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)
	return buf.Bytes()
}

func createMultipart(t *testing.T, data []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	part, err := w.CreateFormFile("file", "image.png")
	require.NoError(t, err)

	_, err = part.Write(data)
	require.NoError(t, err)

	require.NoError(t, w.Close())
	return buf.Bytes(), w.FormDataContentType()
}

func uploadImage(t *testing.T, c *client.ClientWithResponses, data []byte) string {
	t.Helper()
	body, contentType := createMultipart(t, data)
	resp, err := c.PostApiV1UploadsImagesWithBodyWithResponse(t.Context(), contentType, bytes.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, 201, resp.StatusCode())
	return resp.JSON201.Id
}

func TestUploadImage(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		body, contentType := createMultipart(t, generateTestImage(t))

		resp, err := c.PostApiV1UploadsImagesWithBodyWithResponse(t.Context(), contentType, bytes.NewReader(body))
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode())
		assert.NotEmpty(t, resp.JSON201.Id)
	})

	t.Run("empty file", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		body, contentType := createMultipart(t, []byte{})

		resp, err := c.PostApiV1UploadsImagesWithBodyWithResponse(t.Context(), contentType, bytes.NewReader(body))
		require.NoError(t, err)
		assert.Equal(t, 415, resp.StatusCode())
	})

	t.Run("unsupported mime type", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		body, contentType := createMultipart(t, []byte("not an image"))

		resp, err := c.PostApiV1UploadsImagesWithBodyWithResponse(t.Context(), contentType, bytes.NewReader(body))
		require.NoError(t, err)
		assert.Equal(t, 415, resp.StatusCode())
	})
}

func TestGetImage(t *testing.T) {
	srv := setupTestServer(t)

	t.Run("happy path", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		imageID := uploadImage(t, c, generateTestImage(t))

		resp, err := c.GetApiV1UploadsImagesImageIdWithResponse(t.Context(), imageID)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, "image/png", resp.HTTPResponse.Header.Get("Content-Type"))
	})

	t.Run("not found", func(t *testing.T) {
		c := newAuthenticatedTestClient(t, srv)
		resp, err := c.GetApiV1UploadsImagesImageIdWithResponse(t.Context(), "non-existent-id")
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})
}
