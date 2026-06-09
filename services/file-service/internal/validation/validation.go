package validation

import (
	"fmt"
	"path/filepath"
	"strings"
)

// AllowedExtensions mirrors web/src/config/file-upload.ts (keep in sync).
var AllowedExtensions = map[string]struct{}{
	"pdf":  {},
	"png":  {},
	"jpg":  {},
	"jpeg": {},
	"gif":  {},
	"webp": {},
	"md":   {},
	"txt":  {},
	"csv":  {},
	"json": {},
	"zip":  {},
}

const DefaultMaxFileBytes int64 = 50 * 1024 * 1024

// DefaultMaxRequestBodyBytes fits base64-encoded 50 MiB plus JSON envelope.
const DefaultMaxRequestBodyBytes int64 = 70 * 1024 * 1024

func Extension(filename string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(strings.TrimSpace(filename)), "."))
	return ext
}

func IsAllowedExtension(filename string) bool {
	_, ok := AllowedExtensions[Extension(filename)]
	return ok
}

func ValidateUpload(filename string, contentSize int64, maxFileBytes int64) error {
	name := strings.TrimSpace(filename)
	if name == "" {
		return fmt.Errorf("filename is required")
	}
	if !IsAllowedExtension(name) {
		return fmt.Errorf("file type not allowed")
	}
	if contentSize <= 0 {
		return fmt.Errorf("content cannot be empty")
	}
	if maxFileBytes > 0 && contentSize > maxFileBytes {
		return fmt.Errorf("file exceeds maximum size of %d bytes", maxFileBytes)
	}
	return nil
}
