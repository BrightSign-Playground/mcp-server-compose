// Package testutil provides shared helpers for tests.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempFile writes content to a temporary file and registers cleanup.
// Returns the file path.
func TempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("testutil.TempFile: %v", err)
	}
	return path
}

// AssertNoError fails the test if err is non-nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
}
