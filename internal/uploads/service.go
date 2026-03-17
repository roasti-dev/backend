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

	"github.com/nikpivkin/roasti-app-backend/internal/ids"
	"github.com/nikpivkin/roasti-app-backend/internal/log"
)

var allowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

const maxFileSize = 10 * 1024 * 1024 // 10MB

type Service struct {
	basePath string
}

func NewService(basePath string) *Service {
	return &Service{basePath: basePath}
}

func (s *Service) UploadMultipart(mr *multipart.Reader) (string, error) {
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
			id, err := s.Upload(part)
			part.Close()
			return id, err
		default:
			part.Close()
		}
	}
}

func (s *Service) Upload(r io.Reader) (string, error) {
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
	filename := id + ext

	tmpDir := filepath.Join(s.basePath, "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("create tmp dir: %w", err)
	}

	dst, err := os.Create(filepath.Join(tmpDir, filename))
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := dst.Write(buf); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return id, nil
}

type ImageFile struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
}

func (s *Service) Resolve(id string) (*ImageFile, error) {
	for _, dir := range []string{"images", "tmp"} {
		path, err := s.findInDir(dir, id)
		if err == nil {
			return s.openImage(path)
		}
	}
	return nil, ErrNotFound
}

func (s *Service) Confirm(id string) error {
	if _, err := s.findInDir("images", id); err == nil {
		return nil
	}

	src, err := s.findInDir("tmp", id)
	if err != nil {
		return err
	}

	dst := filepath.Join(s.basePath, "images", filepath.Base(src))

	if err := os.MkdirAll(filepath.Join(s.basePath, "images"), 0755); err != nil {
		return fmt.Errorf("create images dir: %w", err)
	}

	return os.Rename(src, dst)
}

func (s *Service) DeleteExpiredTmp(ctx context.Context, maxAge time.Duration) error {
	tmpDir := filepath.Join(s.basePath, "tmp")
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read tmp dir: %w", err)
	}

	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > maxAge {
			if err := os.Remove(filepath.Join(tmpDir, e.Name())); err != nil {
				slog.ErrorContext(ctx, "remove expired tmp file",
					slog.String("file", e.Name()),
					log.Err(err),
				)
			}
		}
	}
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

func (s *Service) findInDir(dir, id string) (string, error) {
	entries, err := os.ReadDir(filepath.Join(s.basePath, dir))
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("read dir: %w", err)
	}

	for _, e := range entries {
		name := e.Name()
		if name[:len(name)-len(filepath.Ext(name))] == id {
			return filepath.Join(s.basePath, dir, name), nil
		}
	}

	return "", ErrNotFound
}
