package autohandler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anushibinj/repo-mapper/internal/autohandler"
)

func TestUpdateCopilotInstructions_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	if err := autohandler.UpdateCopilotInstructions(dir, ".repo-mapper"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, ".github", "copilot-instructions.md"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	content := string(got)
	if !strings.Contains(content, ".repo-mapper/README.md") {
		t.Errorf("expected entrypoint link; got:\n%s", content)
	}
	if !strings.Contains(content, "<!-- repo-mapper:begin -->") {
		t.Error("expected begin marker")
	}
	if !strings.Contains(content, "<!-- repo-mapper:end -->") {
		t.Error("expected end marker")
	}
}

func TestUpdateCopilotInstructions_UpdatesExistingBlock(t *testing.T) {
	dir := t.TempDir()
	githubDir := filepath.Join(dir, ".github")
	if err := os.MkdirAll(githubDir, 0o755); err != nil {
		t.Fatal(err)
	}

	initial := "# My custom instructions\n\nDo something useful.\n" +
		"<!-- repo-mapper:begin -->\nOLD CONTENT\n<!-- repo-mapper:end -->\n" +
		"\nKeep this footer.\n"
	if err := os.WriteFile(filepath.Join(githubDir, "copilot-instructions.md"), []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := autohandler.UpdateCopilotInstructions(dir, ".repo-mapper"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(githubDir, "copilot-instructions.md"))
	content := string(got)

	if strings.Contains(content, "OLD CONTENT") {
		t.Error("managed block was not replaced")
	}
	if !strings.Contains(content, "# My custom instructions") {
		t.Error("custom content before block was removed")
	}
	if !strings.Contains(content, "Keep this footer.") {
		t.Error("custom content after block was removed")
	}
	if !strings.Contains(content, ".repo-mapper/README.md") {
		t.Error("new entrypoint link missing")
	}
}

func TestUpdateCopilotInstructions_AppendsWhenNoBlock(t *testing.T) {
	dir := t.TempDir()
	githubDir := filepath.Join(dir, ".github")
	if err := os.MkdirAll(githubDir, 0o755); err != nil {
		t.Fatal(err)
	}

	initial := "# Existing instructions\n\nDo something.\n"
	if err := os.WriteFile(filepath.Join(githubDir, "copilot-instructions.md"), []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := autohandler.UpdateCopilotInstructions(dir, ".repo-mapper"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(githubDir, "copilot-instructions.md"))
	content := string(got)

	if !strings.Contains(content, "# Existing instructions") {
		t.Error("existing content was removed")
	}
	if !strings.Contains(content, ".repo-mapper/README.md") {
		t.Error("new entrypoint link missing")
	}
}
