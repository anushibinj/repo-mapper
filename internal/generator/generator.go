// Package generator renders the canonical model.Repository into the three
// AI/human-facing output formats: JSON, Markdown, and Mermaid diagrams
// (PRD sections 13-15). Every generator function is a pure function of the
// Repository model, guaranteeing deterministic output for unchanged input
// (Copilot Coding Guideline #9).
package generator

import (
	"os"
	"path/filepath"

	"github.com/anushibinj/repo-mapper/internal/model"
)

// WriteAll renders every JSON, Markdown, and Mermaid artifact into
// outputDir (conventionally ".ai"), creating subdirectories as needed.
func WriteAll(repo *model.Repository, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "diagrams"), 0o755); err != nil {
		return err
	}

	if err := WriteJSON(repo, outputDir); err != nil {
		return err
	}
	if err := WriteMarkdown(repo, outputDir); err != nil {
		return err
	}
	if err := WriteMermaid(repo, outputDir); err != nil {
		return err
	}
	return nil
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
