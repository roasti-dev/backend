package uploads

import (
	"context"
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
	"github.com/nikpivkin/roasti-app-backend/internal/x/ids"
)

var allowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

const maxFileSize = 10 * 1024 * 1024 // 10MB

type Service struct {
	basePath string
	repo     *Repository
}

func NewService(basePath string, repo *Repository) *Service {
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

	id := ids.NewID()
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

func (s *Service) Resolve(ctx context.Context, id string) (*ImageFile, error) {
	if !ids.IsValidID(id) {
		return nil, ErrNotFound
	}

	path, err := s.repo.GetPath(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.openImage(path)
}

func (s *Service) Confirm(ctx context.Context, id string) error {
	if !ids.IsValidID(id) {
		return ErrNotFound
	}
	return s.repo.Confirm(ctx, id)
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
