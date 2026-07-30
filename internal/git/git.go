// Package git wraps the git CLI to provide changed-file detection, branch
// and commit metadata, and merge-base computation needed for incremental
// updates (PRD sections 10 & 17). Shelling out to the system git binary
// keeps this dependency-free and behaviourally identical to what a
// developer or CI runner already has installed.
package git

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

// Repo represents a git repository rooted at Dir.
type Repo struct {
	Dir string
}

// New returns a Repo bound to dir. It does not verify dir is a git repo;
// use IsRepo for that.
func New(dir string) *Repo {
	return &Repo{Dir: dir}
}

func (r *Repo) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// IsRepo reports whether Dir is inside a git working tree.
func (r *Repo) IsRepo() bool {
	out, err := r.run("rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// Branch returns the current branch name (empty if detached).
func (r *Repo) Branch() (string, error) {
	return r.run("rev-parse", "--abbrev-ref", "HEAD")
}

// CommitHash returns the current HEAD commit hash.
func (r *Repo) CommitHash() (string, error) {
	return r.run("rev-parse", "HEAD")
}

// MergeBase returns the merge base commit between HEAD and base (e.g.
// "origin/main"), used to compute a pull-request-scoped diff.
func (r *Repo) MergeBase(base string) (string, error) {
	return r.run("merge-base", "HEAD", base)
}

// ChangedFiles returns file paths (relative to Dir) changed between two
// refs. If to is empty, it defaults to the working tree (i.e. `git diff
// --name-only from`).
func (r *Repo) ChangedFiles(from, to string) ([]string, error) {
	args := []string{"diff", "--name-only"}
	if from != "" {
		if to != "" {
			args = append(args, from+".."+to)
		} else {
			args = append(args, from)
		}
	}
	out, err := r.run(args...)
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

// StagedFiles returns file paths staged for commit.
func (r *Repo) StagedFiles() ([]string, error) {
	out, err := r.run("diff", "--name-only", "--cached")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

// UntrackedFiles returns files not yet tracked by git.
func (r *Repo) UntrackedFiles() ([]string, error) {
	out, err := r.run("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

// WorkingChanges returns the union of unstaged, staged, and untracked
// changes since HEAD — the practical "what changed" set used by
// `repo-mapper update` when no explicit base ref is given.
func (r *Repo) WorkingChanges() ([]string, error) {
	seen := map[string]struct{}{}
	var all []string

	add := func(files []string, err error) error {
		if err != nil {
			return err
		}
		for _, f := range files {
			if _, ok := seen[f]; !ok && f != "" {
				seen[f] = struct{}{}
				all = append(all, f)
			}
		}
		return nil
	}

	if err := add(r.ChangedFiles("HEAD", "")); err != nil {
		return nil, err
	}
	if err := add(r.StagedFiles()); err != nil {
		return nil, err
	}
	if err := add(r.UntrackedFiles()); err != nil {
		return nil, err
	}
	return all, nil
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}
