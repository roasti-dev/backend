package users

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
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

func (r *UserRepository) ListRecommended(ctx context.Context, excludeUserID string, limit, offset int) ([]User, int, error) {
	q := `
SELECT u.id, u.username, u.name, u.avatar_id, COUNT(*) OVER() AS total
FROM users u
JOIN recipes r ON r.author_id = u.id AND r.public = true
JOIN likes l ON l.target_id = r.id AND l.target_type = 'recipe' AND l.created_at >= datetime('now', '-30 days')
WHERE u.id != ?
GROUP BY u.id
ORDER BY COUNT(l.id) DESC
LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, q, excludeUserID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list recommended users: %w", err)
	}
	defer rows.Close()

	var users []User
	var total int
	for rows.Next() {
		var u User
		var name sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &name, &u.AvatarID, &total); err != nil {
			return nil, 0, fmt.Errorf("scan recommended user: %w", err)
		}
		if name.Valid {
			u.Name = &name.String
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate recommended users: %w", err)
	}
	return users, total, nil
}

func (r *UserRepository) ExistsByID(ctx context.Context, userID string) (bool, error) {
	var count int
	err := r.psql.Select("COUNT(1)").
		From(usersTable).
		Where(sq.Eq{"id": userID}).
		QueryRowContext(ctx).
		Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check user exists by id: %w", err)
	}
	return count > 0, nil
}

func (r *UserRepository) GetPreviewsByIDs(ctx context.Context, ids []string) ([]models.UserPreview, error) {
	if len(ids) == 0 {
		return []models.UserPreview{}, nil
	}
	rows, err := r.psql.Select("id", "username", "name", "avatar_id").
		From(usersTable).
		Where(sq.Eq{"id": ids}).
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user previews: %w", err)
	}
	defer rows.Close()

	index := make(map[string]models.UserPreview, len(ids))
	for rows.Next() {
		var p models.UserPreview
		var name sql.NullString
		if err := rows.Scan(&p.Id, &p.Username, &name, &p.AvatarId); err != nil {
			return nil, fmt.Errorf("scan user preview: %w", err)
		}
		if name.Valid {
			p.Name = &name.String
		}
		index[p.Id] = p
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user previews: %w", err)
	}

	// preserve input order
	previews := make([]models.UserPreview, 0, len(ids))
	for _, id := range ids {
		if p, ok := index[id]; ok {
			previews = append(previews, p)
		}
	}
	return previews, nil
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
