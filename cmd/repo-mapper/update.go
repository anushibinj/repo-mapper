package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/anushibinj/repo-mapper/internal/generator"
	gitmod "github.com/anushibinj/repo-mapper/internal/git"
	"github.com/anushibinj/repo-mapper/internal/logger"
	"github.com/anushibinj/repo-mapper/internal/pipeline"
)

func runUpdate(args []string) error {
	fs, root := newFlagSet("update")
	base := fs.String("base", "", "Base ref to diff against (default: auto-detect — working tree changes, "+
		"or if the checkout is clean, the commit recorded in the last generated repo-map.json)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, cfg, log, err := loadContext(*root)
	if err != nil {
		return err
	}

	outputDir := filepath.Join(repoRoot, cfg.Output.Directory)

	effectiveBase, autoDetected := resolveUpdateBase(*base, repoRoot, outputDir, log)

	start := time.Now()
	result, err := pipeline.IncrementalUpdate(repoRoot, cfg, log, effectiveBase)
	if err != nil {
		// A base ref we auto-detected ourselves (from the last generated
		// repo-map.json) may no longer exist — e.g. a shallow CI checkout,
		// a rebased/force-pushed history, or a squashed merge. Rather than
		// fail the whole run (and thus never recover without manual
		// intervention), fall back to a full scan. An explicitly
		// user-supplied -base failing is still a real error and is
		// surfaced as-is.
		if autoDetected && errors.Is(err, pipeline.ErrGitDiffFailed) {
			if log != nil {
				log.Warn("auto-detected base ref is no longer valid for git diff; falling back to a full scan", "error", err)
			}
			fmt.Println("Auto-detected base commit is no longer reachable (e.g. shallow clone or rewritten history) — falling back to a full scan.")
			result, err = pipeline.FullScan(repoRoot, cfg, log)
		}
		if err != nil {
			return fmt.Errorf("update: %w", err)
		}
	}

	existing, err := loadExistingRepository(outputDir)
	if err != nil {
		return fmt.Errorf("read existing output: %w", err)
	}
	if existing != nil && result.Repository.EqualIgnoringGit(existing) {
		fmt.Println("No architectural changes detected since last run (only the git commit metadata would differ).")
		fmt.Println("Skipping documentation rewrite — nothing to commit.")
		return errNoChanges
	}

	if err := generator.WriteAll(result.Repository, outputDir); err != nil {
		return fmt.Errorf("generate output: %w", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("Re-parsed %d changed files (%d files known total) in %s\n",
		result.FilesParsed, result.FilesScanned, elapsed.Round(time.Millisecond))
	fmt.Printf("Wrote documentation to %s\n", outputDir)
	return nil
}

// resolveUpdateBase determines the base ref IncrementalUpdate should diff
// against, and whether that ref was auto-detected (as opposed to explicitly
// passed via -base).
//
// A CI checkout is typically clean (no uncommitted changes), so the
// default "working tree changes since HEAD" behavior finds nothing to do
// even though many commits may have landed since the last successful
// `update`/`scan`. To make `update` work correctly with zero flags in that
// environment, when no -base is given and the working tree is clean, we
// fall back to the commit hash recorded in the last generated
// repo-map.json (repo.Git.CommitHash) and diff from there to HEAD instead.
func resolveUpdateBase(explicitBase, repoRoot, outputDir string, log *logger.Logger) (base string, autoDetected bool) {
	if explicitBase != "" {
		return explicitBase, false
	}

	repo := gitmod.New(repoRoot)
	if !repo.IsRepo() {
		return "", false
	}

	working, err := repo.WorkingChanges()
	if err != nil || len(working) > 0 {
		// Either we can't tell, or there are real uncommitted changes to
		// diff against — let IncrementalUpdate use its normal working-tree
		// behavior.
		return "", false
	}

	existing, err := loadExistingRepository(outputDir)
	if err != nil || existing == nil || existing.Git.CommitHash == "" {
		return "", false
	}

	head, err := repo.CommitHash()
	if err != nil || head == existing.Git.CommitHash {
		// Already up to date with the last generation, or HEAD couldn't be
		// determined — nothing to auto-detect.
		return "", false
	}

	if log != nil {
		log.Info("clean checkout detected; auto-detected base ref from last generated repo-map.json",
			"base", existing.Git.CommitHash)
	}
	return existing.Git.CommitHash, true
}
