package config

import (
	"log"
	"os"
	"strconv"

	"github.com/gateframe/gateconfig"
)

type Config struct {
	ListenAddr       string
	DatabaseURL      string
	InternalToken    string
	GinMode          string
	DefaultLimit     int
	RetentionMonths  int
	PruneIntervalHrs int
}

func Load() Config {
	limit := 50
	if v := os.Getenv("AUDIT_DEFAULT_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	retention := 12
	if v := os.Getenv("AUDIT_RETENTION_MONTHS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			retention = n
		}
	}
	pruneHours := 24
	if v := os.Getenv("AUDIT_PRUNE_INTERVAL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			pruneHours = n
		}
	}
	cfg := Config{
		ListenAddr:       getenv("AUDIT_SERVICE_LISTEN", "0.0.0.0:8084"),
		DatabaseURL:      getenv("DATABASE_URL", "postgres://db:db@127.0.0.1:5432/db?sslmode=disable"),
		InternalToken:    getenv("INTERNAL_TOKEN", "dev-internal-token-change-me"),
		GinMode:          getenv("GIN_MODE", "release"),
		DefaultLimit:     limit,
		RetentionMonths:  retention,
		PruneIntervalHrs: pruneHours,
	}
	if err := gateconfig.ValidateProduction(map[string]string{
		"INTERNAL_TOKEN": cfg.InternalToken,
	}); err != nil {
		log.Fatalf("config: %v", err)
	}
	return cfg
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
