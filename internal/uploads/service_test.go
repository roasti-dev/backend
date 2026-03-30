package uploads_test

import (
	"bytes"
	"mime/multipart"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nikpivkin/roasti-app-backend/internal/db"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
)

func setupTestService(t *testing.T) *uploads.Service {
	t.Helper()
	dir := t.TempDir()
	database, err := db.NewSQLite(t.Context(), ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate(database))
	t.Cleanup(func() { database.Close() }) //nolint: errcheck
	return uploads.NewService(dir, uploads.NewRepository(database))
}

func jpegBytes() []byte {
	return []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46,
		0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01,
		0x00, 0x01, 0x00, 0x00, 0xFF, 0xD9,
	}
}

func jpegFixture() *bytes.Reader {
	return bytes.NewReader(jpegBytes())
}

func multipartFixture(t *testing.T, fieldName string, filename string, data []byte) *multipart.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(fieldName, filename)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return multipart.NewReader(&buf, w.Boundary())
}

func TestUploadMultipart(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		svc := setupTestService(t)
		mr := multipartFixture(t, "file", "image.jpg", jpegBytes())
		id, err := svc.UploadMultipart(t.Context(), mr)
		require.NoError(t, err)
		assert.NotEmpty(t, id)
	})

	t.Run("no file field", func(t *testing.T) {
		svc := setupTestService(t)
		mr := multipartFixture(t, "other", "image.jpg", jpegBytes())
		_, err := svc.UploadMultipart(t.Context(), mr)
		assert.ErrorIs(t, err, uploads.ErrNoFile)
	})

	t.Run("empty multipart", func(t *testing.T) {
		svc := setupTestService(t)
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		require.NoError(t, w.Close())
		mr := multipart.NewReader(&buf, w.Boundary())
		_, err := svc.UploadMultipart(t.Context(), mr)
		assert.ErrorIs(t, err, uploads.ErrNoFile)
	})

	t.Run("unsupported mime type", func(t *testing.T) {
		svc := setupTestService(t)
		mr := multipartFixture(t, "file", "file.txt", []byte("plain text"))
		_, err := svc.UploadMultipart(t.Context(), mr)
		assert.ErrorIs(t, err, uploads.ErrUnsupportedMIMEType)
	})
}

func TestUpload(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		svc := setupTestService(t)
		id, err := svc.Upload(t.Context(), jpegFixture())
		require.NoError(t, err)
		assert.NotEmpty(t, id)
	})

	t.Run("file too large", func(t *testing.T) {
		svc := setupTestService(t)
		_, err := svc.Upload(t.Context(), bytes.NewReader(make([]byte, 11*1024*1024)))
		assert.ErrorIs(t, err, uploads.ErrFileTooLarge)
	})

	t.Run("unsupported mime type", func(t *testing.T) {
		svc := setupTestService(t)
		_, err := svc.Upload(t.Context(), bytes.NewReader([]byte("plain text")))
		assert.ErrorIs(t, err, uploads.ErrUnsupportedMIMEType)
	})
}

func TestConfirm(t *testing.T) {
	t.Run("moves file from tmp to images", func(t *testing.T) {
		svc := setupTestService(t)
		id, err := svc.Upload(t.Context(), jpegFixture())
		require.NoError(t, err)

		err = svc.Confirm(t.Context(), id)
		require.NoError(t, err)

		f, err := svc.Resolve(t.Context(), id)
		require.NoError(t, err)
		require.NoError(t, f.Body.Close())
	})

	t.Run("idempotent", func(t *testing.T) {
		svc := setupTestService(t)
		id, err := svc.Upload(t.Context(), jpegFixture())
		require.NoError(t, err)

		require.NoError(t, svc.Confirm(t.Context(), id))
		require.NoError(t, svc.Confirm(t.Context(), id))
	})

	t.Run("not found", func(t *testing.T) {
		svc := setupTestService(t)
		err := svc.Confirm(t.Context(), "nonexistent")
		assert.ErrorIs(t, err, uploads.ErrNotFound)
	})

	t.Run("invalid id", func(t *testing.T) {
		svc := setupTestService(t)
		err := svc.Confirm(t.Context(), "../../etc/passwd")
		assert.ErrorIs(t, err, uploads.ErrNotFound)
	})
}

func TestDeleteUnconfirmed(t *testing.T) {
	t.Run("deletes old files", func(t *testing.T) {
		svc := setupTestService(t)
		id, err := svc.Upload(t.Context(), jpegFixture())
		require.NoError(t, err)

		err = svc.DeleteUnconfirmed(t.Context(), 0)
		require.NoError(t, err)

		_, err = svc.Resolve(t.Context(), id)
		assert.ErrorIs(t, err, uploads.ErrNotFound)
	})

	t.Run("keeps recent files", func(t *testing.T) {
		svc := setupTestService(t)
		id, err := svc.Upload(t.Context(), jpegFixture())
		require.NoError(t, err)

		err = svc.DeleteUnconfirmed(t.Context(), 24*time.Hour)
		require.NoError(t, err)

		_, err = svc.Resolve(t.Context(), id)
		require.NoError(t, err)
	})

	t.Run("does not delete confirmed", func(t *testing.T) {
		svc := setupTestService(t)
		id, err := svc.Upload(t.Context(), jpegFixture())
		require.NoError(t, err)

		require.NoError(t, svc.Confirm(t.Context(), id))

		err = svc.DeleteUnconfirmed(t.Context(), 0)
		require.NoError(t, err)

		f, err := svc.Resolve(t.Context(), id)
		require.NoError(t, err)
		require.NoError(t, f.Body.Close())
	})

	t.Run("no unconfirmed files", func(t *testing.T) {
		svc := setupTestService(t)
		err := svc.DeleteUnconfirmed(t.Context(), 24*time.Hour)
		assert.NoError(t, err)
	})
}
