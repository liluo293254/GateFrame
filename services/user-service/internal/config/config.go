package config

import (
	"log"
	"os"
	"strconv"

	"github.com/gateframe/gateconfig"
)

type Config struct {
	ListenAddr    string
	DatabaseURL   string
	JWTSecret     string
	JWTExpirySecs int
	InternalToken string
	RedisURL      string
	GinMode       string
}

func Load() Config {
	expiry := 3600
	if v := os.Getenv("JWT_EXPIRY_SECS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			expiry = n
		}
	}
	cfg := Config{
		ListenAddr:    getenv("USER_SERVICE_LISTEN", "0.0.0.0:8082"),
		DatabaseURL:   getenv("DATABASE_URL", "postgres://db:db@127.0.0.1:5432/db?sslmode=disable"),
		JWTSecret:     getenv("JWT_SECRET", "dev-only-change-me-in-production"),
		JWTExpirySecs: expiry,
		InternalToken: getenv("INTERNAL_TOKEN", "dev-internal-token-change-me"),
		RedisURL:      getenv("REDIS_URL", ""),
		GinMode:       getenv("GIN_MODE", "release"),
	}
	if err := gateconfig.ValidateProduction(map[string]string{
		"JWT_SECRET":     cfg.JWTSecret,
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
