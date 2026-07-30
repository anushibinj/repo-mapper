package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTestRepo creates a real git repository in a temp dir with one commit,
// so tests exercise the actual `git` CLI behavior this package wraps.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	run("init", "-q", "-b", "main")
	run("-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "--allow-empty", "-q", "-m", "initial commit")

	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "tracked.txt")
	run("-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-q", "-m", "add tracked.txt")

	return dir
}

func TestRepo_IsRepo(t *testing.T) {
	dir := initTestRepo(t)
	r := New(dir)
	if !r.IsRepo() {
		t.Error("expected IsRepo()=true for an initialized git repo")
	}

	notRepo := New(t.TempDir())
	if notRepo.IsRepo() {
		t.Error("expected IsRepo()=false for a non-git directory")
	}
}

func TestRepo_BranchAndCommitHash(t *testing.T) {
	dir := initTestRepo(t)
	r := New(dir)

	branch, err := r.Branch()
	if err != nil {
		t.Fatalf("Branch failed: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected branch=main, got %q", branch)
	}

	hash, err := r.CommitHash()
	if err != nil {
		t.Fatalf("CommitHash failed: %v", err)
	}
	if len(hash) != 40 {
		t.Errorf("expected a 40-char commit hash, got %q", hash)
	}
}

func TestRepo_ChangedFilesAndWorkingChanges(t *testing.T) {
	dir := initTestRepo(t)
	r := New(dir)

	// Unstaged modification.
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Untracked new file.
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := r.ChangedFiles("HEAD", "")
	if err != nil {
		t.Fatalf("ChangedFiles failed: %v", err)
	}
	if len(changed) != 1 || changed[0] != "tracked.txt" {
		t.Errorf("expected ChangedFiles=[tracked.txt], got %v", changed)
	}

	untracked, err := r.UntrackedFiles()
	if err != nil {
		t.Fatalf("UntrackedFiles failed: %v", err)
	}
	if len(untracked) != 1 || untracked[0] != "new.txt" {
		t.Errorf("expected UntrackedFiles=[new.txt], got %v", untracked)
	}

	all, err := r.WorkingChanges()
	if err != nil {
		t.Fatalf("WorkingChanges failed: %v", err)
	}
	found := map[string]bool{}
	for _, f := range all {
		found[f] = true
	}
	if !found["tracked.txt"] || !found["new.txt"] {
		t.Errorf("expected WorkingChanges to include both tracked.txt and new.txt, got %v", all)
	}
}

func TestRepo_StagedFiles(t *testing.T) {
	dir := initTestRepo(t)
	r := New(dir)

	if err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "staged.txt")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}

	staged, err := r.StagedFiles()
	if err != nil {
		t.Fatalf("StagedFiles failed: %v", err)
	}
	if len(staged) != 1 || staged[0] != "staged.txt" {
		t.Errorf("expected StagedFiles=[staged.txt], got %v", staged)
	}
}
