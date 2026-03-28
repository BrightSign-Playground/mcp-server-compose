package ingest

import (
	"testing"
)

func TestValidateDir_empty(t *testing.T) {
	_, err := validateDir("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestValidateDir_nullByte(t *testing.T) {
	_, err := validateDir("/tmp/foo\x00bar")
	if err == nil {
		t.Fatal("expected error for null byte in path")
	}
}

func TestValidateDir_notExist(t *testing.T) {
	_, err := validateDir("/definitely/does/not/exist/abc123")
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

func TestValidateDir_notDir(t *testing.T) {
	// /etc/hosts is a file, not a directory.
	_, err := validateDir("/etc/hosts")
	if err == nil {
		t.Fatal("expected error when path is not a directory")
	}
}

func TestValidateDir_valid(t *testing.T) {
	path, err := validateDir("/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}
