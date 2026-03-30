package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Migration struct {
	Version string
	UpSQL   string
	DownSQL string
}

func Migrate(ctx context.Context, databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	return applyMigrations(ctx, db)
}

func Rollback(ctx context.Context, databaseURL string, steps int) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	return rollbackMigrations(ctx, db, steps)
}

func applyMigrations(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return err
	}

	migrations, err := LoadMigrations()
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		var exists bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, migration.Version).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migration.UpSQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", migration.Version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, migration.Version); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func rollbackMigrations(ctx context.Context, db *sql.DB, steps int) error {
	if steps < 1 {
		return nil
	}
	migrations, err := LoadMigrations()
	if err != nil {
		return err
	}
	byVersion := make(map[string]Migration, len(migrations))
	for _, migration := range migrations {
		byVersion[migration.Version] = migration
	}

	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT $1`, steps)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	versions := make([]string, 0, steps)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return err
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, version := range versions {
		migration, ok := byVersion[version]
		if !ok {
			return fmt.Errorf("missing migration definition for rollback version %s", version)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migration.DownSQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("rollback migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = $1`, version); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func LoadMigrations() ([]Migration, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	out := make([]Migration, 0, len(names))
	for _, name := range names {
		version := strings.TrimSuffix(name, ".up.sql")
		upBytes, err := migrationFiles.ReadFile(filepath.Join("migrations", name))
		if err != nil {
			return nil, err
		}
		downBytes, err := migrationFiles.ReadFile(filepath.Join("migrations", version+".down.sql"))
		if err != nil {
			return nil, err
		}
		out = append(out, Migration{
			Version: version,
			UpSQL:   string(upBytes),
			DownSQL: string(downBytes),
		})
	}
	return out, nil
}
