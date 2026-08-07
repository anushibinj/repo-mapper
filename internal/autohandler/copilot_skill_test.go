package autohandler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anushibinj/repo-mapper/internal/autohandler"
)

func TestUpdateCopilotSkill_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	if err := autohandler.UpdateCopilotSkill(dir, ".repo-mapper"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	skillPath := filepath.Join(dir, ".github", "skills", "understand-repo", "SKILL.md")
	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("SKILL.md not created at expected path: %v", err)
	}
	content := string(got)

	if !strings.Contains(content, "name: understand-repo") {
		t.Error("expected 'name: understand-repo' in frontmatter")
	}
	if !strings.Contains(content, ".repo-mapper/README.md") {
		t.Error("expected entrypoint link in skill body")
	}
	if !strings.Contains(content, "---") {
		t.Error("expected YAML frontmatter delimiters")
	}
}

func TestUpdateCopilotSkill_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()

	// Pre-create the file with stale content.
	skillDir := filepath.Join(dir, ".github", "skills", "understand-repo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := "---\nname: understand-repo\ndescription: stale\n---\nOLD CONTENT\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := autohandler.UpdateCopilotSkill(dir, ".repo-mapper"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	content := string(got)

	if strings.Contains(content, "OLD CONTENT") {
		t.Error("stale content was not replaced")
	}
	if !strings.Contains(content, ".repo-mapper/README.md") {
		t.Error("new entrypoint link missing")
	}
}

func TestUpdateCopilotSkill_CustomOutputDir(t *testing.T) {
	dir := t.TempDir()
	if err := autohandler.UpdateCopilotSkill(dir, "docs/map"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, ".github", "skills", "understand-repo", "SKILL.md"))
	if err != nil {
		t.Fatalf("SKILL.md not created: %v", err)
	}
	if !strings.Contains(string(got), "docs/map/README.md") {
		t.Errorf("expected custom output dir in skill body; got:\n%s", string(got))
	}
}
