package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
)

const usersTable = "users"

var userColumns = []string{"id", "email", "username", "avatar_id", "bio", "created_at"}

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

func (r *Repository) Create(ctx context.Context, user User) error {
	_, err := r.psql.Insert(usersTable).
		Columns(userColumns...).
		Values(
			user.ID,
			user.Email,
			user.Username,
			user.AvatarID,
			user.Bio,
			time.Now().UTC(),
		).
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := r.psql.Select(userColumns...).
		From(usersTable).
		Where(sq.Eq{"username": username}).
		QueryRowContext(ctx).
		Scan(&user.ID, &user.Email, &user.Username, &user.AvatarID, &user.Bio, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("select user: %w", err)
	}
	return user, nil
}

func (r *Repository) GetByID(ctx context.Context, userID string) (User, error) {
	var user User
	err := r.psql.Select(userColumns...).
		From(usersTable).
		Where(sq.Eq{"id": userID}).
		QueryRowContext(ctx).
		Scan(&user.ID, &user.Email, &user.Username, &user.AvatarID, &user.Bio, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

func (r *Repository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int
	err := r.psql.Select("COUNT(1)").
		From(usersTable).
		Where(sq.Eq{"username": username}).
		QueryRowContext(ctx).
		Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check username exists: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int
	err := r.psql.Select("COUNT(1)").
		From(usersTable).
		Where(sq.Eq{"email": email}).
		QueryRowContext(ctx).
		Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return count > 0, nil
}
