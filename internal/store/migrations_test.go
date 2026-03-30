package store

import (
	"path/filepath"
	"testing"
)

func TestEmbeddedMigrationPairsExist(t *testing.T) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	seenUp := make(map[string]bool)
	seenDown := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		switch filepath.Ext(name) {
		case ".sql":
			switch {
			case len(name) > len(".up.sql") && name[len(name)-len(".up.sql"):] == ".up.sql":
				seenUp[name[:len(name)-len(".up.sql")]] = true
			case len(name) > len(".down.sql") && name[len(name)-len(".down.sql"):] == ".down.sql":
				seenDown[name[:len(name)-len(".down.sql")]] = true
			}
		}
	}

	if len(seenUp) == 0 {
		t.Fatal("expected at least one up migration")
	}

	for version := range seenUp {
		if !seenDown[version] {
			t.Fatalf("missing down migration for %s", version)
		}
	}
}

func TestLoadMigrationsReturnsRollbackSQL(t *testing.T) {
	migrations, err := LoadMigrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	if len(migrations) == 0 {
		t.Fatal("expected migrations to load")
	}
	for _, migration := range migrations {
		if migration.Version == "" || migration.UpSQL == "" || migration.DownSQL == "" {
			t.Fatalf("expected complete migration set, got %+v", migration)
		}
	}
}
