package gateconfig

import (
	"fmt"
	"os"
	"strings"
)

const (
	DevJWTSecret     = "dev-only-change-me-in-production"
	DevInternalToken = "dev-internal-token-change-me"
	MinSecretLength  = 32
)

// IsProduction returns true when GATEFRAME_ENV is production or prod.
func IsProduction() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("GATEFRAME_ENV")))
	return env == "production" || env == "prod"
}

// ValidateProduction rejects known dev defaults and short secrets in production.
func ValidateProduction(secrets map[string]string) error {
	if !IsProduction() {
		return nil
	}
	devDefaults := map[string]string{
		"JWT_SECRET":      DevJWTSecret,
		"INTERNAL_TOKEN":  DevInternalToken,
		"MINIO_ACCESS_KEY": "minioadmin",
		"MINIO_SECRET_KEY": "minioadmin",
	}
	for name, value := range secrets {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("%s must be set when GATEFRAME_ENV=production", name)
		}
		if def, ok := devDefaults[name]; ok && value == def {
			return fmt.Errorf("%s must not use the development default in production", name)
		}
		if name == "JWT_SECRET" || name == "INTERNAL_TOKEN" {
			if len(value) < MinSecretLength {
				return fmt.Errorf("%s must be at least %d characters in production", name, MinSecretLength)
			}
		}
	}
	return nil
}
