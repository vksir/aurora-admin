package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?cache=shared&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &DB{DB: db}, nil
}

func OpenMemory() (*DB, error) {
	db, err := sql.Open("sqlite", "file:ent?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	return &DB{DB: db}, nil
}

func (db *DB) Migrate(ctx context.Context) error {
	_, err := db.ExecContext(ctx, schema)
	return err
}

// WithTx runs fn in a transaction. Nested calls reuse the same tx via context.
type txKey struct{}

func (db *DB) WithTx(ctx context.Context, fn func(ctx context.Context, tx *sql.Tx) error) error {
	tx, exists := ctx.Value(txKey{}).(*sql.Tx)
	if !exists {
		var err error
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, txKey{}, tx)

		defer func() {
			if v := recover(); v != nil {
				_ = tx.Rollback()
				panic(v)
			}
		}()
	}

	err := fn(ctx, tx)
	if !exists {
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		return tx.Commit()
	}
	return err
}

// Tx extracts the transaction from context, or returns nil.
func Tx(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value(txKey{}).(*sql.Tx)
	return tx
}

// Exec is a helper that uses tx if available, otherwise db.
func (db *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tx := Tx(ctx)
	if tx != nil {
		return tx.ExecContext(ctx, query, args...)
	}
	return db.DB.ExecContext(ctx, query, args...)
}

// Query is a helper that uses tx if available, otherwise db.
func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	tx := Tx(ctx)
	if tx != nil {
		return tx.QueryContext(ctx, query, args...)
	}
	return db.DB.QueryContext(ctx, query, args...)
}

// QueryRow is a helper that uses tx if available, otherwise db.
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	tx := Tx(ctx)
	if tx != nil {
		return tx.QueryRowContext(ctx, query, args...)
	}
	return db.DB.QueryRowContext(ctx, query, args...)
}

const schema = `
CREATE TABLE IF NOT EXISTS dont_starve_archives (
    id TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    remark TEXT,
    cluster_name TEXT,
    cluster_password TEXT,
    cluster_description TEXT,
    max_players INTEGER,
    pvp INTEGER,
    world TEXT
);

CREATE TABLE IF NOT EXISTS dont_starve_mods (
    id TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    workshop_id TEXT NOT NULL UNIQUE,
    name TEXT,
    description TEXT,
    image TEXT,
    time_created INTEGER,
    time_updated INTEGER,
    raw_config TEXT
);

CREATE TABLE IF NOT EXISTS archive_mods (
    archive_id TEXT NOT NULL,
    mod_id TEXT NOT NULL,
    PRIMARY KEY (archive_id, mod_id),
    FOREIGN KEY (archive_id) REFERENCES dont_starve_archives(id) ON DELETE CASCADE,
    FOREIGN KEY (mod_id) REFERENCES dont_starve_mods(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS dont_starve_admins (
    id TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    name TEXT,
    klei_id TEXT NOT NULL UNIQUE
);
`
