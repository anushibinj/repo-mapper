package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/anushibinj/repo-mapper/internal/generator"
	"github.com/anushibinj/repo-mapper/internal/pipeline"
)

func runScan(args []string) error {
	fs, root := newFlagSet("scan")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, cfg, log, err := loadContext(*root)
	if err != nil {
		return err
	}

	start := time.Now()
	result, err := pipeline.FullScan(repoRoot, cfg, log)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	outputDir := filepath.Join(repoRoot, cfg.Output.Directory)

	existing, err := loadExistingRepository(outputDir)
	if err != nil {
		return fmt.Errorf("read existing output: %w", err)
	}
	if existing != nil && result.Repository.EqualIgnoringGit(existing) {
		fmt.Println("No architectural changes detected since last run (only the git commit metadata would differ).")
		fmt.Println("Skipping documentation rewrite — nothing to commit.")

		// Still run autohandlers so that missing .github files (skill,
		// copilot-instructions) are created even when the repo map is
		// already up to date.
		runAutohandlers(repoRoot, cfg, log)
		return errNoChanges
	}

	if err := generator.WriteAll(result.Repository, outputDir); err != nil {
		return fmt.Errorf("generate output: %w", err)
	}

	runAutohandlers(repoRoot, cfg, log)

	elapsed := time.Since(start)
	fmt.Printf("Scanned %d files (%d parsed, %d cache hits) in %s\n",
		result.FilesScanned, result.FilesParsed, result.CacheHits, elapsed.Round(time.Millisecond))
	fmt.Printf("Wrote documentation to %s\n", outputDir)
	return nil
}
