package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
)

const revokedTokensTable = "revoked_tokens"

type RevokedTokenRepository struct {
	db   *sql.DB
	psql sq.StatementBuilderType
}

func NewRevokedTokenRepository(db *sql.DB) *RevokedTokenRepository {
	return &RevokedTokenRepository{
		db:   db,
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(db),
	}
}

func (r *RevokedTokenRepository) Add(ctx context.Context, token string) error {
	hash := hashToken(token)
	_, err := r.psql.Insert(revokedTokensTable).
		Columns("token_hash", "revoked_at").
		Values(hash, time.Now().UTC()).
		Suffix("ON CONFLICT (token_hash) DO NOTHING").
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert revoked token: %w", err)
	}
	return nil
}

func (r *RevokedTokenRepository) IsRevoked(ctx context.Context, token string) (bool, error) {
	hash := hashToken(token)
	var count int
	err := r.psql.Select("COUNT(*)").
		From(revokedTokensTable).
		Where(sq.Eq{"token_hash": hash}).
		QueryRowContext(ctx).
		Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check revoked token: %w", err)
	}
	return count > 0, nil
}

func (r *RevokedTokenRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.psql.Delete(revokedTokensTable).
		Where("expires_at < datetime('now')").
		ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete expired tokens: %w", err)
	}
	return nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
