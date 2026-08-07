package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anushibinj/repo-mapper/internal/config"
	"github.com/anushibinj/repo-mapper/internal/logger"
	"github.com/anushibinj/repo-mapper/internal/model"
)

// errNoChanges is a sentinel returned by scan/update when the freshly built
// model is identical to the previously written repo-map.json apart from Git
// metadata. Callers (main) translate it into a distinct exit code so
// automation can tell "nothing to commit" apart from a real failure, instead
// of parsing stdout.
var errNoChanges = errors.New("no architectural changes since last run (only git metadata would differ)")

// loadExistingRepository reads outputDir/repo-map.json from a previous run,
// if present. A missing file is not an error — it just means there is
// nothing to compare against yet (e.g. first run).
func loadExistingRepository(outputDir string) (*model.Repository, error) {
	data, err := os.ReadFile(filepath.Join(outputDir, "repo-map.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var repo model.Repository
	if err := json.Unmarshal(data, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

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
