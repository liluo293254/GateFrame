package validation

import "testing"

func TestIsAllowedExtension(t *testing.T) {
	t.Parallel()

	allowed := []string{"doc.pdf", "photo.PNG", "notes.md", "data.json", "archive.zip"}
	for _, name := range allowed {
		if !IsAllowedExtension(name) {
			t.Fatalf("expected allowed: %s", name)
		}
	}

	blocked := []string{"malware.exe", "script.sh", "binary", ".exe"}
	for _, name := range blocked {
		if IsAllowedExtension(name) {
			t.Fatalf("expected blocked: %s", name)
		}
	}
}

func TestValidateUploadRejectsExe(t *testing.T) {
	t.Parallel()

	err := ValidateUpload("payload.exe", 4, DefaultMaxFileBytes)
	if err == nil {
		t.Fatal("expected error for .exe upload")
	}
	if err.Error() != "file type not allowed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateUploadRejectsOversizedFile(t *testing.T) {
	t.Parallel()

	err := ValidateUpload("large.txt", DefaultMaxFileBytes+1, DefaultMaxFileBytes)
	if err == nil {
		t.Fatal("expected size error")
	}
}

func TestValidateUploadAcceptsWithinLimit(t *testing.T) {
	t.Parallel()

	if err := ValidateUpload("ok.txt", 1024, DefaultMaxFileBytes); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
