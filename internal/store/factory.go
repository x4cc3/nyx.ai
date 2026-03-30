package store

import (
	"context"
	"time"
)

type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func OpenRepository(ctx context.Context, databaseURL string, pool ...PoolConfig) (Repository, error) {
	if databaseURL == "" {
		repo := NewMemoryStore()
		if err := repo.Init(ctx); err != nil {
			return nil, err
		}
		return repo, nil
	}

	var repo *PostgresStore
	var err error
	if len(pool) > 0 {
		p := pool[0]
		repo, err = NewPostgresStoreWithConfig(databaseURL, p.MaxOpenConns, p.MaxIdleConns, p.ConnMaxLifetime, p.ConnMaxIdleTime)
	} else {
		repo, err = NewPostgresStore(databaseURL)
	}
	if err != nil {
		return nil, err
	}
	if err := repo.Init(ctx); err != nil {
		_ = repo.Close()
		return nil, err
	}
	return repo, nil
}
