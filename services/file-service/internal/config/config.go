package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gateframe/file-service/internal/validation"
	"github.com/gateframe/gateconfig"
)

type Config struct {
	ListenAddr           string
	DatabaseURL          string
	InternalToken        string
	GinMode              string
	MaxFileBytes         int64
	MaxRequestBodyBytes  int64
	MinIO                MinIOConfig
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

func Load() Config {
	cfg := Config{
		ListenAddr:          getenv("FILE_SERVICE_LISTEN", "0.0.0.0:8088"),
		DatabaseURL:         getenv("DATABASE_URL", "postgres://db:db@127.0.0.1:5432/db?sslmode=disable"),
		InternalToken:       getenv("INTERNAL_TOKEN", "dev-internal-token-change-me"),
		GinMode:             getenv("GIN_MODE", "release"),
		MaxFileBytes:        getenvInt64("FILE_MAX_BYTES", validation.DefaultMaxFileBytes),
		MaxRequestBodyBytes: getenvInt64("FILE_MAX_REQUEST_BODY_BYTES", validation.DefaultMaxRequestBodyBytes),
		MinIO: MinIOConfig{
			Endpoint:  getenv("MINIO_ENDPOINT", "minio:9000"),
			AccessKey: getenv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: getenv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    getenv("MINIO_BUCKET", "gateframe"),
			UseSSL:    getenvBool("MINIO_USE_SSL", false),
		},
	}
	if err := gateconfig.ValidateProduction(map[string]string{
		"INTERNAL_TOKEN":   cfg.InternalToken,
		"MINIO_ACCESS_KEY": cfg.MinIO.AccessKey,
		"MINIO_SECRET_KEY": cfg.MinIO.SecretKey,
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

func getenvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvInt64(key string, fallback int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
