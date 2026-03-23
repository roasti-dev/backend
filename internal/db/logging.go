package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	sq "github.com/Masterminds/squirrel"
)

var (
	_ sq.StdSqlCtx = (*loggingRunner)(nil)
)

type loggingRunner struct {
	db     sq.StdSqlCtx
	logger *slog.Logger
}

func NewRunner(db *sql.DB, logger *slog.Logger, debug bool) sq.StdSqlCtx {
	if debug {
		return newLoggingRunner(db, logger)
	}
	return db
}

func newLoggingRunner(db sq.StdSqlCtx, logger *slog.Logger) *loggingRunner {
	return &loggingRunner{db: db, logger: logger}
}

func (l *loggingRunner) BeginTx(ctx context.Context, opts *sql.TxOptions) (*loggingRunner, error) {
	db, ok := l.db.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("not a *sql.DB")
	}
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	return newLoggingRunner(tx, l.logger), nil
}

func (l *loggingRunner) Rollback() error {
	tx, ok := l.db.(*sql.Tx)
	if !ok {
		return fmt.Errorf("not a *sql.Tx")
	}
	return tx.Rollback()
}

func (l *loggingRunner) Commit() error {
	tx, ok := l.db.(*sql.Tx)
	if !ok {
		return fmt.Errorf("not a *sql.Tx")
	}
	return tx.Commit()
}

func (l *loggingRunner) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	result, err := l.db.ExecContext(ctx, query, args...)
	l.logger.DebugContext(ctx, "exec",
		slog.String("query", query),
		slog.Any("args", args),
		slog.Duration("duration", time.Since(start)))
	return result, err
}

func (l *loggingRunner) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := l.db.QueryContext(ctx, query, args...)
	l.logger.DebugContext(ctx, "query",
		slog.String("query", query),
		slog.Any("args", args),
		slog.Duration("duration", time.Since(start)),
	)
	return rows, err
}

func (l *loggingRunner) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := l.db.QueryRowContext(ctx, query, args...)
	l.logger.DebugContext(ctx, "queryRow",
		slog.String("query", query),
		slog.Any("args", args),
		slog.Duration("duration", time.Since(start)),
	)
	return row
}

func (l *loggingRunner) Exec(query string, args ...any) (sql.Result, error) {
	return l.db.Exec(query, args...)
}

func (l *loggingRunner) Query(query string, args ...any) (*sql.Rows, error) {
	return l.db.Query(query, args...)
}

func (l *loggingRunner) QueryRow(query string, args ...any) *sql.Row {
	return l.db.QueryRow(query, args...)
}
