package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPasswordMatchesAdminSeed(t *testing.T) {
	// Hash used in migrations/001_init.up.sql for admin / Admin@123456
	const seedHash = "$2a$10$rdLZIa3luMiNClizZTbZX.S3W3AW3Pk9I084s8iIQ70SkLsqfiExa"
	err := bcrypt.CompareHashAndPassword([]byte(seedHash), []byte("Admin@123456"))
	require.NoError(t, err)
}

func TestHashPasswordRoundtrip(t *testing.T) {
	hash, err := HashPassword("TestPass@123")
	require.NoError(t, err)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("TestPass@123")))
}
