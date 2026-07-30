package scanner

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ignoreRule is one parsed line from a gitignore-style file or an explicit
// exclude entry.
type ignoreRule struct {
	pattern  string
	negate   bool
	dirOnly  bool
	anchored bool // pattern contains a "/" other than a trailing one -> relative to root
}

type ignoreMatcher struct {
	rules []ignoreRule
}

// newIgnoreMatcher builds a matcher from always-excluded names/globs,
// root .gitignore, and an optional custom ignore file, plus baked-in
// defaults for VCS metadata directories.
func newIgnoreMatcher(root string, exclude []string, ignoreFile string) *ignoreMatcher {
	m := &ignoreMatcher{}

	for _, e := range exclude {
		m.rules = append(m.rules, parseIgnoreLine(e))
	}
	m.rules = append(m.rules, parseIgnoreLine(".git"))

	if data, err := os.ReadFile(filepath.Join(root, ".gitignore")); err == nil {
		m.addLines(string(data))
	}
	if ignoreFile != "" {
		if data, err := os.ReadFile(filepath.Join(root, ignoreFile)); err == nil {
			m.addLines(string(data))
		}
	}
	return m
}

func (m *ignoreMatcher) addLines(content string) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		m.rules = append(m.rules, parseIgnoreLine(trimmed))
	}
}

func parseIgnoreLine(line string) ignoreRule {
	r := ignoreRule{}
	if strings.HasPrefix(line, "!") {
		r.negate = true
		line = line[1:]
	}
	if strings.HasSuffix(line, "/") {
		r.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}
	if strings.HasPrefix(line, "/") {
		r.anchored = true
		line = strings.TrimPrefix(line, "/")
	} else if strings.Contains(line, "/") {
		r.anchored = true
	}
	r.pattern = line
	return r
}

// matches reports whether relPath (forward-slash, relative to root) should
// be ignored. isDir indicates whether relPath is a directory.
func (m *ignoreMatcher) matches(relPath string, isDir bool) bool {
	ignored := false
	base := path.Base(relPath)

	for _, r := range m.rules {
		if r.dirOnly && !isDir {
			// A dir-only rule can still match an ancestor directory of a
			// file; that's handled by WalkDir's SkipDir short-circuit, so
			// here we only need to consider isDir matches directly.
			continue
		}

		var matched bool
		if r.anchored {
			ok, _ := path.Match(r.pattern, relPath)
			matched = ok
		} else {
			ok, _ := path.Match(r.pattern, base)
			if !ok {
				// also try matching against the full relative path so
				// patterns like "**/foo" behave reasonably without full
				// gitignore double-star semantics
				ok2, _ := path.Match(r.pattern, relPath)
				ok = ok2
			}
			matched = ok
		}

		if matched {
			ignored = !r.negate
		}
	}
	return ignored
}

// IgnoreMatcher is the exported form of ignoreMatcher, usable outside this
// package. The incremental-update pipeline needs it to filter a raw `git
// diff` file list through the same exclude/.gitignore rules a full Scan
// applies, so cache/output artifacts (.cache/, .ai/, etc.) never get
// reprocessed as if they were source changes.
type IgnoreMatcher struct {
	m *ignoreMatcher
}

// NewIgnoreMatcher builds an IgnoreMatcher from the same inputs Scan uses.
func NewIgnoreMatcher(root string, exclude []string, ignoreFile string) *IgnoreMatcher {
	return &IgnoreMatcher{m: newIgnoreMatcher(root, exclude, ignoreFile)}
}

// Matches reports whether relPath (forward-slash, relative to root) should
// be ignored.
func (i *IgnoreMatcher) Matches(relPath string, isDir bool) bool {
	return i.m.matches(relPath, isDir)
}
