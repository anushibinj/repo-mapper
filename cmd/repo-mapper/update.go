package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/anushibinj/repo-mapper/internal/generator"
	"github.com/anushibinj/repo-mapper/internal/pipeline"
)

func runUpdate(args []string) error {
	fs, root := newFlagSet("update")
	base := fs.String("base", "", "Base ref to diff against (default: working tree changes since HEAD)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, cfg, log, err := loadContext(*root)
	if err != nil {
		return err
	}

	start := time.Now()
	result, err := pipeline.IncrementalUpdate(repoRoot, cfg, log, *base)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	outputDir := filepath.Join(repoRoot, cfg.Output.Directory)
	if err := generator.WriteAll(result.Repository, outputDir); err != nil {
		return fmt.Errorf("generate output: %w", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("Re-parsed %d changed files (%d files known total) in %s\n",
		result.FilesParsed, result.FilesScanned, elapsed.Round(time.Millisecond))
	fmt.Printf("Wrote documentation to %s\n", outputDir)
	return nil
}
