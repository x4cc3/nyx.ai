package main

import (
	"context"
	"flag"
	"log"

	"nyx/internal/config"
	"nyx/internal/store"
	"nyx/internal/version"
)

func main() {
	rollbackSteps := flag.Int("rollback", 0, "rollback the most recent N migrations instead of applying up migrations")
	flag.Parse()

	cfg := config.Load()
	if err := cfg.Validate("migrate"); err != nil {
		log.Fatal(err)
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required for migrations")
	}

	if *rollbackSteps > 0 {
		if err := store.Rollback(context.Background(), cfg.DatabaseURL, *rollbackSteps); err != nil {
			log.Fatal(err)
		}
		log.Printf("nyx migrations rolled back %d step(s) [%s]", *rollbackSteps, version.String())
		return
	}

	if err := store.Migrate(context.Background(), cfg.DatabaseURL); err != nil {
		log.Fatal(err)
	}

	log.Printf("nyx migrations applied [%s]", version.String())
}
