package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/anushibinj/repo-mapper/internal/logger"
)

// initTestRepoWithOutput creates a real git repository in a temp dir with
// two commits, and writes a repo-map.json under outputDir/repo-map.json
// recording the first commit's hash — simulating "docs were generated at
// commit A, then commit B landed afterward without regenerating docs".
// Returns the repo dir and the first commit's hash.
func initTestRepoWithOutput(t *testing.T) (repoDir, outputDir, firstCommit string) {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
		return string(out)
	}

	run("init", "-q", "-b", "main")
	run("-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "--allow-empty", "-q", "-m", "first commit")
	head := run("rev-parse", "HEAD")
	firstCommit = trimNewline(head)

	outputDir = filepath.Join(dir, ".repo-mapper")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	repoMap := `{"name":"demo","git":{"branch":"main","commitHash":"` + firstCommit + `"}}`
	if err := os.WriteFile(filepath.Join(outputDir, "repo-map.json"), []byte(repoMap), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-q", "-m", "commit generated docs")

	// A second, unrelated commit that lands after the docs were generated —
	// this is the state a CI checkout would be in: clean working tree, but
	// HEAD has moved past what repo-map.json recorded.
	if err := os.WriteFile(filepath.Join(dir, "app.go"), []byte("package app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "app.go")
	run("-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-q", "-m", "add app.go")

	return dir, outputDir, firstCommit
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func TestResolveUpdateBase_ExplicitBaseIsUsedAsIs(t *testing.T) {
	repoDir, outputDir, _ := initTestRepoWithOutput(t)
	base, auto := resolveUpdateBase("some-explicit-ref", repoDir, outputDir, logger.Nop())
	if base != "some-explicit-ref" || auto {
		t.Fatalf("expected explicit base to pass through unchanged, got base=%q auto=%v", base, auto)
	}
}

func TestResolveUpdateBase_CleanCheckoutAutoDetectsLastGeneratedCommit(t *testing.T) {
	repoDir, outputDir, firstCommit := initTestRepoWithOutput(t)
	base, auto := resolveUpdateBase("", repoDir, outputDir, logger.Nop())
	if !auto {
		t.Fatalf("expected auto-detection to trigger on a clean checkout with a stale recorded commit")
	}
	if base != firstCommit {
		t.Fatalf("expected auto-detected base %q, got %q", firstCommit, base)
	}
}

func TestResolveUpdateBase_DirtyWorkingTreeSkipsAutoDetection(t *testing.T) {
	repoDir, outputDir, _ := initTestRepoWithOutput(t)
	if err := os.WriteFile(filepath.Join(repoDir, "app.go"), []byte("package app\n\n// dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	base, auto := resolveUpdateBase("", repoDir, outputDir, logger.Nop())
	if auto || base != "" {
		t.Fatalf("expected no auto-detection with uncommitted changes present, got base=%q auto=%v", base, auto)
	}
}

func TestResolveUpdateBase_NotARepoSkipsAutoDetection(t *testing.T) {
	dir := t.TempDir()
	base, auto := resolveUpdateBase("", dir, filepath.Join(dir, ".repo-mapper"), logger.Nop())
	if auto || base != "" {
		t.Fatalf("expected no auto-detection outside a git repo, got base=%q auto=%v", base, auto)
	}
}
