package uploads

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
)

const uploadsTable = "uploads"

type Repository struct {
	db   *sql.DB
	psql sq.StatementBuilderType
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db:   db,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(db),
	}
}

func (r *Repository) Add(ctx context.Context, id, path, mimeType string) error {
	_, err := r.psql.Insert(uploadsTable).
		Columns("id", "path", "mime_type", "created_at", "confirmed").
		Values(id, path, mimeType, time.Now().UTC(), false).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert upload: %w", err)
	}
	return nil
}

func (r *Repository) GetPath(ctx context.Context, id string) (string, error) {
	var path string
	err := r.psql.Select("path").
		From(uploadsTable).
		Where(sq.Eq{"id": id}).
		QueryRowContext(ctx).
		Scan(&path)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get upload path: %w", err)
	}
	return path, nil
}

func (r *Repository) Confirm(ctx context.Context, id string) error {
	res, err := r.psql.Update(uploadsTable).
		Set("confirmed", true).
		Where(sq.Eq{"id": id}).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("confirm upload: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("confirm upload rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteUnconfirmed(ctx context.Context, maxAge time.Duration) ([]string, error) {
	cutoff := time.Now().UTC().Add(-maxAge)
	rows, err := r.psql.Select("id", "path").
		From(uploadsTable).
		Where(sq.Eq{"confirmed": false}).
		Where(sq.Lt{"created_at": cutoff}).
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("query unconfirmed uploads: %w", err)
	}
	defer rows.Close()

	var ids []string
	var paths []string
	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, fmt.Errorf("scan unconfirmed upload: %w", err)
		}
		ids = append(ids, id)
		paths = append(paths, path)
	}

	if len(ids) == 0 {
		return nil, nil
	}

	_, err = r.psql.Delete(uploadsTable).
		Where(sq.Eq{"id": ids}).
		ExecContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("delete unconfirmed uploads: %w", err)
	}

	return paths, nil
}
