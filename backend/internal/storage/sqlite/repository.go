package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS pack_sizes (
	size INTEGER PRIMARY KEY NOT NULL CHECK(size > 0)
);`

var (
	mkdirAll   = os.MkdirAll
	openDB     = sql.Open
	runMigrate = func(ctx context.Context, repository *Repository) error {
		return repository.Migrate(ctx)
	}
)

type Repository struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Repository, error) {
	if err := mkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := openDB("sqlite", path)
	if err != nil {
		return nil, err
	}

	repository := NewRepository(db)
	if err := runMigrate(ctx, repository); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repository, nil
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, schema)
	return err
}

func (r *Repository) GetPackSizes(ctx context.Context) ([]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT size FROM pack_sizes ORDER BY size ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sizes []int
	for rows.Next() {
		var size int
		if err := rows.Scan(&size); err != nil {
			return nil, err
		}
		sizes = append(sizes, size)
	}

	return sizes, rows.Err()
}

func (r *Repository) ReplacePackSizes(ctx context.Context, sizes []int) ([]int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM pack_sizes`); err != nil {
		return nil, err
	}

	for _, size := range sizes {
		if _, err := tx.ExecContext(ctx, `INSERT INTO pack_sizes(size) VALUES (?)`, size); err != nil {
			return nil, err
		}
	}

	return sizes, tx.Commit()
}

func (r *Repository) SeedPackSizesIfEmpty(ctx context.Context, sizes []int) error {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pack_sizes`).Scan(&count); err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	_, err := r.ReplacePackSizes(ctx, sizes)
	return err
}

func (r *Repository) Close() error {
	return r.db.Close()
}
