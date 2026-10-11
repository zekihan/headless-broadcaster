package environment

import (
	"os"
	"path/filepath"
	"testing"
)

func createTempEnvFile(t *testing.T, filename, content string) {
	t.Helper()

	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create temp env file: %v", err)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		t.Fatalf("failed to write to temp env file: %v", err)
	}
}

func removeFile(filename string) {
	_ = os.Remove(filename)
}

func TestLoadEnvironmentFile(t *testing.T) {
	// Setup
	createTempEnvFile(t, ".env", "FOO=bar\n")
	createTempEnvFile(t, ".override.env", "BAR=baz\n")

	defer removeFile(".env")
	defer removeFile(".override.env")

	os.Unsetenv("FOO")
	os.Unsetenv("BAR")

	err := LoadEnvironmentFile()
	if err != nil {
		t.Fatalf("LoadEnvironmentFile failed: %v", err)
	}

	if os.Getenv("FOO") != "bar" {
		t.Errorf("expected FOO=bar, got %s", os.Getenv("FOO"))
	}

	if os.Getenv("BAR") != "baz" {
		t.Errorf("expected BAR=baz, got %s", os.Getenv("BAR"))
	}
}

func TestLoadEnvironmentFileWithOther(t *testing.T) {
	createTempEnvFile(t, ".env", "FOO=bar\n")
	createTempEnvFile(t, ".override.env", "BAR=baz\n")
	createTempEnvFile(t, "custom.env", "BAZ=qux\n")

	defer removeFile(".env")
	defer removeFile(".override.env")
	defer removeFile("custom.env")

	os.Unsetenv("BAZ")

	err := LoadEnvironmentFile("custom.env")
	if err != nil {
		t.Fatalf("LoadEnvironmentFile with custom.env failed: %v", err)
	}

	if os.Getenv("BAZ") != "qux" {
		t.Errorf("expected BAZ=qux, got %s", os.Getenv("BAZ"))
	}
}

func TestLoadEnvironmentFileMissingFiles(t *testing.T) {
	removeFile(".env")
	removeFile(".override.env")

	err := LoadEnvironmentFile()
	if err != nil {
		t.Errorf("expected no error when env files are missing, got %v", err)
	}
}

func TestLoadFileDirectory(t *testing.T) {
	dir := t.TempDir()

	err := loadFile(dir)
	if err != nil {
		t.Errorf("expected nil for directory path, got %v", err)
	}
}

func TestLoadFileStatError(t *testing.T) {
	// Traversing through a regular file yields a non-NotExist stat error
	// (ENOTDIR). loadFile must surface it instead of dereferencing a nil
	// FileInfo and panicking.
	dir := t.TempDir()

	regular := filepath.Join(dir, "file")

	err := os.WriteFile(regular, []byte("x"), 0o600)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	bogus := filepath.Join(regular, "sub")

	err = loadFile(bogus)
	if err == nil {
		t.Errorf("expected error for stat failure, got nil")
	}
}
