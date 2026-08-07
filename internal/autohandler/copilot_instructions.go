// Package autohandler contains post-generation hooks that update adjacent
// repository files (e.g. .github/copilot-instructions.md) so that AI
// assistants automatically discover the repo-mapper entrypoint instead of
// crawling the entire repository tree for context.
package autohandler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// managed block delimiters — everything between these markers is owned
	// by repo-mapper and will be replaced on each run.
	blockBegin = "<!-- repo-mapper:begin -->"
	blockEnd   = "<!-- repo-mapper:end -->"

	// defaultGithubDir is the conventional location for GitHub-specific files.
	defaultGithubDir = ".github"
	// instructionsFile is the filename Copilot reads for custom instructions.
	instructionsFile = "copilot-instructions.md"
)

// UpdateCopilotInstructions writes or updates .github/copilot-instructions.md
// so that GitHub Copilot (and other AI assistants) know to read the
// repo-mapper entrypoint at <outputDir>/README.md for repo context before
// exploring source files.
//
// If the file already exists the managed block (between the begin/end marker
// comments) is replaced in-place, leaving any surrounding custom content
// untouched. If the file does not exist it is created with the managed block
// as its only content.
//
// repoRoot is the absolute path of the repository root.
// outputDir is the relative path of the repo-mapper output directory
// (e.g. ".repo-mapper"), as written to the generated README.
func UpdateCopilotInstructions(repoRoot, outputDir string) error {
	githubDir := filepath.Join(repoRoot, defaultGithubDir)
	if err := os.MkdirAll(githubDir, 0o755); err != nil {
		return fmt.Errorf("create .github dir: %w", err)
	}

	targetPath := filepath.Join(githubDir, instructionsFile)

	managed := buildManagedBlock(outputDir)

	existing, err := os.ReadFile(targetPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read %s: %w", instructionsFile, err)
		}
		// File does not exist — create it with just the managed block.
		return writeFile(targetPath, managed+"\n")
	}

	// File exists — replace the managed block, or append it if absent.
	updated := replaceManagedBlock(string(existing), managed)
	return writeFile(targetPath, updated)
}

// buildManagedBlock returns the full repo-mapper managed section that will
// be embedded in copilot-instructions.md.
func buildManagedBlock(outputDir string) string {
	// Use forward-slash paths so the Markdown is portable across OS.
	entrypoint := filepath.ToSlash(filepath.Join(outputDir, "README.md"))

	var b strings.Builder
	b.WriteString(blockBegin)
	b.WriteString("\n")
	b.WriteString("## Understanding This Repository\n\n")
	b.WriteString("This repository is documented by [repo-mapper](https://github.com/anushibinj/repo-mapper).\n\n")
	b.WriteString("**Before exploring source files**, read the repository map to get a structured\n")
	b.WriteString("overview of the architecture, components, and file layout:\n\n")
	fmt.Fprintf(&b, "- [`%s`](%s) — repo-mapper entrypoint: languages, modules, and links to all\n", entrypoint, entrypoint)
	b.WriteString("  detailed views (backend, frontend, database, architecture, features).\n\n")
	b.WriteString("Use the linked files in that directory to understand the codebase structure.\n")
	b.WriteString("Only open individual source files when you need implementation details that\n")
	b.WriteString("the map does not cover.\n")
	b.WriteString(blockEnd)
	return b.String()
}

// replaceManagedBlock finds the managed block inside content and replaces it.
// If no managed block is present, the new block is appended.
func replaceManagedBlock(content, managed string) string {
	begin := strings.Index(content, blockBegin)
	end := strings.Index(content, blockEnd)

	if begin == -1 || end == -1 || end < begin {
		// No existing managed block — append with a blank separator line.
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + "\n" + managed + "\n"
	}

	// Replace everything from the begin marker to the end of the end marker.
	endIdx := end + len(blockEnd)
	return content[:begin] + managed + content[endIdx:]
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
