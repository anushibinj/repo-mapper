package cache

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// gitignoreEntry is the line Repo Mapper ensures is present in the
// repository's .gitignore. The cache directory is a performance-only
// artifact (file hashes and previously-parsed entities) with no value to
// commit — unlike the output directory (conventionally .repo-mapper/),
// which the PRD's CI/CD workflow (section 20) expects to be committed as
// living documentation. Nobody should have to remember to gitignore the
// cache by hand, so every scan/update run ensures it automatically.
const gitignoreEntry = ".cache/"

// EnsureIgnored makes sure repoRoot/.gitignore excludes the cache
// directory, creating .gitignore if it doesn't exist, or appending to it
// if it exists but doesn't already cover the cache dir. It is idempotent:
// calling it repeatedly never adds a duplicate entry.
func EnsureIgnored(repoRoot string) error {
	path := filepath.Join(repoRoot, ".gitignore")

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if alreadyIgnores(string(data), gitignoreEntry) {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	var b strings.Builder
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		b.WriteString("\n")
	}
	b.WriteString("\n# Repo Mapper cache (performance-only, not meant to be versioned)\n")
	b.WriteString(gitignoreEntry + "\n")
	_, err = f.WriteString(b.String())
	return err
}

// alreadyIgnores reports whether any non-comment line in gitignore content
// already covers entry, tolerating cosmetic variations such as a missing
// or present leading/trailing slash (".cache", ".cache/", "/.cache/" are
// all treated as equivalent).
func alreadyIgnores(content, entry string) bool {
	target := strings.Trim(entry, "/")
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Trim(line, "/") == target {
			return true
		}
	}
	return false
}
