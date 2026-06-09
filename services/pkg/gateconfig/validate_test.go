package gateconfig

import (
	"os"
	"testing"
)

func TestValidateProductionSkipsInDev(t *testing.T) {
	t.Setenv("GATEFRAME_ENV", "development")
	if err := ValidateProduction(map[string]string{
		"INTERNAL_TOKEN": DevInternalToken,
	}); err != nil {
		t.Fatalf("expected nil in dev, got %v", err)
	}
}

func TestValidateProductionRejectsDefaultToken(t *testing.T) {
	t.Setenv("GATEFRAME_ENV", "production")
	err := ValidateProduction(map[string]string{
		"INTERNAL_TOKEN": DevInternalToken,
	})
	if err == nil {
		t.Fatal("expected error for default internal token")
	}
}

func TestValidateProductionRejectsShortSecret(t *testing.T) {
	t.Setenv("GATEFRAME_ENV", "prod")
	err := ValidateProduction(map[string]string{
		"JWT_SECRET": "too-short-for-production",
	})
	if err == nil {
		t.Fatal("expected error for short jwt secret")
	}
}

func TestIsProduction(t *testing.T) {
	t.Setenv("GATEFRAME_ENV", "production")
	if !IsProduction() {
		t.Fatal("expected production")
	}
	t.Setenv("GATEFRAME_ENV", "")
	if IsProduction() {
		t.Fatal("expected non-production")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
