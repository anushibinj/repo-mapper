package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureIgnored_CreatesGitignoreWhenMissing(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureIgnored(dir); err != nil {
		t.Fatalf("EnsureIgnored failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore to be created: %v", err)
	}
	if !alreadyIgnores(string(data), gitignoreEntry) {
		t.Errorf("expected created .gitignore to cover %q, got:\n%s", gitignoreEntry, data)
	}
}

func TestEnsureIgnored_AppendsToExistingGitignoreWithoutDuplicating(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("node_modules/\n*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureIgnored(dir); err != nil {
		t.Fatalf("EnsureIgnored failed: %v", err)
	}
	if err := EnsureIgnored(dir); err != nil {
		t.Fatalf("second EnsureIgnored call failed: %v", err)
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}
	content := string(data)

	if !alreadyIgnores(content, gitignoreEntry) {
		t.Errorf("expected .gitignore to cover %q, got:\n%s", gitignoreEntry, content)
	}
	// Original entries must be preserved.
	if !alreadyIgnores(content, "node_modules/") {
		t.Errorf("expected original node_modules/ entry to survive, got:\n%s", content)
	}

	// Idempotency: running twice must not add a second .cache/ line.
	count := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == ".cache/" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 .cache/ line after two runs, got %d in:\n%s", count, content)
	}
}

func TestEnsureIgnored_NoOpWhenAlreadyCoveredWithVariantSlashes(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")
	original := "build/\n.cache\n"
	if err := os.WriteFile(gitignorePath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureIgnored(dir); err != nil {
		t.Fatalf("EnsureIgnored failed: %v", err)
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}
	if string(data) != original {
		t.Errorf("expected .gitignore to be left untouched when already covered (no trailing slash variant), got:\n%s", data)
	}
}
