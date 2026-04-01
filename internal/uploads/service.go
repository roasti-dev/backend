package uploads

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/log"
	"github.com/nikpivkin/roasti-app-backend/internal/x/id"
)

var allowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

const maxFileSize = 10 * 1024 * 1024 // 10MB

type UploadStore interface {
	Add(ctx context.Context, id, path, mimeType string) error
	GetPath(ctx context.Context, id string) (string, error)
	Confirm(ctx context.Context, id string) error
	Copy(ctx context.Context, srcID, dstID, dstPath string) error
	Delete(ctx context.Context, id string) (string, error)
	DeleteUnconfirmed(ctx context.Context, maxAge time.Duration) ([]string, error)
}

type Service struct {
	basePath string
	repo     UploadStore
}

func NewService(basePath string, repo UploadStore) *Service {
	return &Service{basePath: basePath, repo: repo}
}

func (s *Service) UploadMultipart(ctx context.Context, mr *multipart.Reader) (string, error) {
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			return "", ErrNoFile
		}
		if err != nil {
			return "", fmt.Errorf("read part: %w", err)
		}

		switch part.FormName() {
		case "file":
			id, err := s.Upload(ctx, part)
			part.Close()
			return id, err
		default:
			part.Close()
		}
	}
}

func (s *Service) Upload(ctx context.Context, r io.Reader) (string, error) {
	buf, err := io.ReadAll(io.LimitReader(r, maxFileSize+1))
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	if len(buf) > maxFileSize {
		return "", ErrFileTooLarge
	}

	mimeType := http.DetectContentType(buf)
	mimeType, _, _ = mime.ParseMediaType(mimeType)

	ext, ok := allowedMIMETypes[mimeType]
	if !ok {
		return "", ErrUnsupportedMIMEType
	}

	id := id.NewID()
	path := filepath.Join(s.basePath, "images", id+ext)

	if err := os.MkdirAll(filepath.Join(s.basePath, "images"), 0755); err != nil {
		return "", fmt.Errorf("create images dir: %w", err)
	}

	dst, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := dst.Write(buf); err != nil {
		if err := os.Remove(path); err != nil {
			slog.WarnContext(ctx, "failed to remove temp file", "path", path, "error", err)
		}
		return "", fmt.Errorf("write file: %w", err)
	}

	if err := s.repo.Add(ctx, id, path, mimeType); err != nil {
		if err := os.Remove(path); err != nil {
			slog.WarnContext(ctx, "failed to remove temp file", "path", path, "error", err)
		}
		return "", fmt.Errorf("save upload: %w", err)
	}
	return id, nil
}

type ImageFile struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
}

func (s *Service) Resolve(ctx context.Context, imageId string) (*ImageFile, error) {
	if !id.IsValidID(imageId) {
		return nil, ErrNotFound
	}

	path, err := s.repo.GetPath(ctx, imageId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return s.openImage(path)
}

func (s *Service) Confirm(ctx context.Context, imageId string) error {
	if !id.IsValidID(imageId) {
		return ErrNotFound
	}
	if err := s.repo.Confirm(ctx, imageId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) Copy(ctx context.Context, srcID string) (string, error) {
	if !id.IsValidID(srcID) {
		return "", ErrNotFound
	}

	srcPath, err := s.repo.GetPath(ctx, srcID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("get source path: %w", err)
	}

	newID := id.NewID()
	dstPath := filepath.Join(filepath.Dir(srcPath), newID+filepath.Ext(srcPath))

	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("open source file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("create dest file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		if rmErr := os.Remove(dstPath); rmErr != nil {
			slog.ErrorContext(ctx, "Failed to remove destionation file",
				slog.String("upload_id", srcID),
				slog.String("source", srcPath),
				slog.String("destination", dstPath),
				log.Err(err),
			)
		}
		return "", fmt.Errorf("copy file: %w", err)
	}

	if err := s.repo.Copy(ctx, srcID, newID, dstPath); errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	} else if err != nil {
		if rmErr := os.Remove(dstPath); rmErr != nil {
			slog.ErrorContext(ctx, "Failed to remove destionation file",
				slog.String("upload_id", srcID),
				slog.String("source", srcPath),
				slog.String("destination", dstPath),
				log.Err(err),
			)
		}
		return "", fmt.Errorf("save copied upload: %w", err)
	}

	return newID, nil
}

func (s *Service) Delete(ctx context.Context, imageId string) error {
	if !id.IsValidID(imageId) {
		return ErrNotFound
	}
	path, err := s.repo.Delete(ctx, imageId)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		slog.WarnContext(ctx, "failed to remove upload file", slog.String("path", path), log.Err(err))
	}
	return nil
}

func (s *Service) DeleteUnconfirmed(ctx context.Context, maxAge time.Duration) error {
	paths, err := s.repo.DeleteUnconfirmed(ctx, maxAge)
	if err != nil {
		return fmt.Errorf("delete unconfirmed records: %w", err)
	}

	var removed int
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			slog.ErrorContext(ctx, "remove unconfirmed file",
				slog.String("file", path),
				log.Err(err),
			)
			continue
		}
		removed++
	}

	slog.InfoContext(ctx, "unconfirmed uploads cleanup",
		slog.Int("removed", removed),
		slog.Int("total", len(paths)),
	)
	return nil
}

func (s *Service) openImage(path string) (*ImageFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("stat file: %w", err)
	}

	return &ImageFile{
		Body:        f,
		ContentType: mime.TypeByExtension(filepath.Ext(path)),
		Size:        stat.Size(),
	}, nil
}
