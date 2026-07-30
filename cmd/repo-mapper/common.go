package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anushibinj/repo-mapper/internal/config"
	"github.com/anushibinj/repo-mapper/internal/logger"
)

// resolveRepoRoot returns an absolute path for the given (possibly empty,
// meaning cwd) root argument.
func resolveRepoRoot(root string) (string, error) {
	if root == "" {
		root = "."
	}
	return filepath.Abs(root)
}

// loadContext resolves the repo root and configuration in one step, used
// by every command that operates on a repository.
func loadContext(root string) (repoRoot string, cfg *config.Config, log *logger.Logger, err error) {
	repoRoot, err = resolveRepoRoot(root)
	if err != nil {
		return "", nil, nil, err
	}
	if info, statErr := os.Stat(repoRoot); statErr != nil || !info.IsDir() {
		return "", nil, nil, fmt.Errorf("repository root does not exist or is not a directory: %s", repoRoot)
	}
	cfg, err = config.Load(repoRoot)
	if err != nil {
		return "", nil, nil, fmt.Errorf("load config: %w", err)
	}
	log = logger.Default()
	return repoRoot, cfg, log, nil
}

// newFlagSet builds a *flag.FlagSet pre-populated with the --root flag
// shared by every repository-scoped command.
func newFlagSet(name string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	root := fs.String("root", "", "Repository root (default: current directory)")
	return fs, root
}
