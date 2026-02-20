package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type CopyCapable interface {
	CopyFrom(ctx context.Context, table string, columns []string, filePath string) (int64, error)
}

type Database struct {
	db   *sql.DB
	pool *pgxpool.Pool
}

func NewDatabaseConnection(ctx context.Context, domainStringName string) (*Database, error) {
	db, err := sql.Open("pgx", domainStringName)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db ping: %w", err)
	}

	pool, err := pgxpool.New(ctx, domainStringName)
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_ = db.Close()
		return nil, fmt.Errorf("pgxpool ping: %w", err)
	}

	return &Database{db: db, pool: pool}, nil
}

func (db *Database) Close() error {
	if db == nil || db.db == nil {
		return nil
	}
	fmt.Println("DB closed.")
	return db.db.Close()
}

func (db *Database) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.db.ExecContext(ctx, query, args...)
}

func (db *Database) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.db.QueryContext(ctx, query, args...)
}

func (db *Database) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.db.QueryRowContext(ctx, query, args...)
}

func quoteProtect(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func buildCopyQuery(tableName string, columns []string) string {
	protectedColumns := make([]string, len(columns))
	for i, col := range columns {
		protectedColumns[i] = quoteProtect(col)
	}

	return fmt.Sprintf(
		"COPY %s (%s) FROM STDIN WITH (FORMAT csv, HEADER true)",
		tableName,
		strings.Join(protectedColumns, ", "),
	)
}

func (db *Database) CopyFromCSVFile(ctx context.Context, table string, columns []string, filePath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire connection from pool: %w", err)
	}
	defer conn.Release()

	copyQuery := buildCopyQuery(table, columns)
	res, err := conn.Conn().PgConn().CopyFrom(ctx, file, copyQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to copy from CSV file: %w", err)
	}
	return res.RowsAffected(), nil
}

func (db *Database) CopyFrom(ctx context.Context, table string, columns []string, filePath string) (int64, error) {
	return db.CopyFromCSVFile(ctx, table, columns, filePath)
}
