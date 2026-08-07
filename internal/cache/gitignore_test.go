package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureIgnored_CreatesGitignoreInsideOutputDir(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureIgnored(dir, ".repo-mapper"); err != nil {
		t.Fatalf("EnsureIgnored failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".repo-mapper", ".gitignore"))
	if err != nil {
		t.Fatalf("expected .repo-mapper/.gitignore to be created: %v", err)
	}
	if !alreadyIgnores(string(data), gitignoreEntry) {
		t.Errorf("expected created .gitignore to cover %q, got:\n%s", gitignoreEntry, data)
	}
}

func TestEnsureIgnored_AppendsToExistingGitignoreWithoutDuplicating(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, ".repo-mapper")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitignorePath := filepath.Join(outDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("# existing\n*.tmp\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureIgnored(dir, ".repo-mapper"); err != nil {
		t.Fatalf("EnsureIgnored failed: %v", err)
	}
	if err := EnsureIgnored(dir, ".repo-mapper"); err != nil {
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
	if !strings.Contains(content, "*.tmp") {
		t.Errorf("expected original *.tmp entry to survive, got:\n%s", content)
	}

	// Idempotency: running twice must not add a second cache/ line.
	count := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == "cache/" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 cache/ line after two runs, got %d in:\n%s", count, content)
	}
}

func TestEnsureIgnored_NoOpWhenAlreadyCoveredWithVariantSlashes(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, ".repo-mapper")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitignorePath := filepath.Join(outDir, ".gitignore")
	original := "build/\ncache\n"
	if err := os.WriteFile(gitignorePath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureIgnored(dir, ".repo-mapper"); err != nil {
		t.Fatalf("EnsureIgnored failed: %v", err)
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}
	if string(data) != original {
		t.Errorf("expected .gitignore to be left untouched when already covered, got:\n%s", string(data))
	}
}
