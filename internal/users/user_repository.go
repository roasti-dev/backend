package users

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
)

const usersTable = "users"

var userColumns = []string{"id", "email", "username", "name", "avatar_id", "bio", "created_at"}

type UserRepository struct {
	db   *sql.DB
	psql sq.StatementBuilderType
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db:   db,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(db),
	}
}

func (r *UserRepository) GetByID(ctx context.Context, userID string) (User, error) {
	var user User
	var name sql.NullString
	err := r.psql.Select(userColumns...).
		From(usersTable).
		Where(sq.Eq{"id": userID}).
		QueryRowContext(ctx).
		Scan(&user.ID, &user.Email, &user.Username, &name, &user.AvatarID, &user.Bio, &user.CreatedAt)
	if err != nil {
		return User{}, fmt.Errorf("get user by id: %w", err)
	}
	if name.Valid {
		user.Name = &name.String
	}
	return user, nil
}

func (r *UserRepository) Create(ctx context.Context, user User) error {
	_, err := r.psql.Insert(usersTable).
		Columns(userColumns...).
		Values(
			user.ID,
			user.Email,
			user.Username,
			user.Name,
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

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (User, error) {
	var user User
	var name sql.NullString
	err := r.psql.Select(userColumns...).
		From(usersTable).
		Where(sq.Eq{"username": username}).
		QueryRowContext(ctx).
		Scan(&user.ID, &user.Email, &user.Username, &name, &user.AvatarID, &user.Bio, &user.CreatedAt)
	if err != nil {
		return User{}, fmt.Errorf("select user: %w", err)
	}
	if name.Valid {
		user.Name = &name.String
	}
	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, userID string, req UpdateUserFields) error {
	if !req.HasFields() {
		return nil
	}
	q := r.psql.Update(usersTable).Where(sq.Eq{"id": userID})
	if req.Username != nil {
		q = q.Set("username", *req.Username)
	}
	if req.Name.IsSpecified() {
		if req.Name.IsNull() {
			q = q.Set("name", nil)
		} else {
			q = q.Set("name", req.Name.MustGet())
		}
	}
	if req.Bio.IsSpecified() {
		if req.Bio.IsNull() {
			q = q.Set("bio", nil)
		} else {
			q = q.Set("bio", req.Bio.MustGet())
		}
	}
	if req.AvatarID.IsSpecified() {
		if req.AvatarID.IsNull() {
			q = q.Set("avatar_id", nil)
		} else {
			q = q.Set("avatar_id", req.AvatarID.MustGet())
		}
	}
	if _, err := q.ExecContext(ctx); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
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

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
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
