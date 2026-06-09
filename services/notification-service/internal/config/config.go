package config

import (
	"log"
	"os"

	"github.com/gateframe/gateconfig"
)

type Config struct {
	ListenAddr    string
	DatabaseURL   string
	InternalToken string
	GinMode       string
}

func Load() Config {
	cfg := Config{
		ListenAddr:    getenv("NOTIFICATION_SERVICE_LISTEN", "0.0.0.0:8087"),
		DatabaseURL:   getenv("DATABASE_URL", "postgres://db:db@127.0.0.1:5432/db?sslmode=disable"),
		InternalToken: getenv("INTERNAL_TOKEN", "dev-internal-token-change-me"),
		GinMode:       getenv("GIN_MODE", "release"),
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
