// Package pipeline wires together the scanner, cache, parser, and analyzer
// into the two top-level workflows the CLI exposes: a full scan and a
// git-diff-driven incremental update (PRD section 7 & 17).
package pipeline

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/anushibinj/repo-mapper/internal/analyzer"
	"github.com/anushibinj/repo-mapper/internal/cache"
	"github.com/anushibinj/repo-mapper/internal/config"
	gitmod "github.com/anushibinj/repo-mapper/internal/git"
	"github.com/anushibinj/repo-mapper/internal/logger"
	"github.com/anushibinj/repo-mapper/internal/model"
	"github.com/anushibinj/repo-mapper/internal/parser"
	"github.com/anushibinj/repo-mapper/internal/plugin"
	"github.com/anushibinj/repo-mapper/internal/scanner"
)

// ErrGitDiffFailed wraps any failure from the underlying `git diff` used to
// discover changed files. Callers (namely the `update` command) use
// errors.Is to detect this specific failure mode — e.g. when an
// auto-detected base ref no longer exists in a shallow CI checkout — and
// can choose to fall back to a full scan instead of failing outright.
var ErrGitDiffFailed = errors.New("git diff failed")

// Result carries the built model plus run statistics useful for CLI output.
type Result struct {
	Repository   *model.Repository
	FilesScanned int
	FilesParsed  int
	CacheHits    int
}

// FullScan walks the entire repository, parses every file matched by a
// registered plugin (skipping re-parse when the cached hash still matches),
// and builds the canonical Repository model.
func FullScan(repoRoot string, cfg *config.Config, log *logger.Logger) (*Result, error) {
	// Ensure the cache directory is gitignored before anything writes to it.
	// This is a convenience, not a hard requirement, so a failure here is
	// logged and never fails the scan.
	if err := cache.EnsureIgnored(repoRoot); err != nil && log != nil {
		log.Warn("failed to update .gitignore for cache directory", "error", err)
	}

	files, err := scanner.Scan(scanner.Options{
		RepoRoot:   repoRoot,
		Exclude:    cfg.Scan.Exclude,
		IgnoreFile: cfg.Scan.IgnoreFile,
		Workers:    cfg.Scan.Workers,
	})
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	c, err := cache.Load(cache.Dir(repoRoot))
	if err != nil {
		return nil, fmt.Errorf("load cache: %w", err)
	}

	plugins := enabledPlugins(cfg)

	var entities []model.Entity
	parsedCount := 0
	cacheHits := 0
	seenPaths := map[string]bool{}

	for _, f := range files {
		seenPaths[f.Path] = true
		if f.Binary {
			continue
		}

		if c.Unchanged(f.Path, f.Hash) {
			cached, ok := c.Get(f.Path)
			if ok {
				entities = append(entities, cached...)
				cacheHits++
				continue
			}
		}

		res := parser.Dispatch(plugins, log, repoRoot, f.Path, f.AbsPath)
		if res.Err != nil {
			if log != nil {
				log.Warn("failed to read file", "file", f.Path, "error", res.Err)
			}
			continue
		}
		c.Put(f.Path, f.Hash, res.Entities)
		entities = append(entities, res.Entities...)
		parsedCount++
	}

	// Drop cache entries for files that no longer exist.
	for path := range c.Hashes {
		if !seenPaths[path] {
			c.Forget(path)
		}
	}

	if err := c.Save(); err != nil {
		return nil, fmt.Errorf("save cache: %w", err)
	}

	repo := buildRepository(repoRoot, files, entities)
	return &Result{Repository: repo, FilesScanned: len(files), FilesParsed: parsedCount, CacheHits: cacheHits}, nil
}

// IncrementalUpdate uses `git diff` to discover changed files and only
// parses those, reusing cached entities for every other previously-known
// file. Unlike FullScan, it does not walk the entire repository tree,
// which is the primary source of speedup for large repositories.
func IncrementalUpdate(repoRoot string, cfg *config.Config, log *logger.Logger, baseRef string) (*Result, error) {
	repo := gitmod.New(repoRoot)
	if !repo.IsRepo() {
		return nil, fmt.Errorf("update: %s is not a git repository", repoRoot)
	}

	// Ensure the cache directory is gitignored before anything writes to it.
	// This is a convenience, not a hard requirement, so a failure here is
	// logged and never fails the update.
	if err := cache.EnsureIgnored(repoRoot); err != nil && log != nil {
		log.Warn("failed to update .gitignore for cache directory", "error", err)
	}

	var changed []string
	var err error
	if baseRef != "" {
		changed, err = repo.ChangedFiles(baseRef, "")
	} else {
		changed, err = repo.WorkingChanges()
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGitDiffFailed, err)
	}

	c, err := cache.Load(cache.Dir(repoRoot))
	if err != nil {
		return nil, fmt.Errorf("load cache: %w", err)
	}

	plugins := enabledPlugins(cfg)
	ignoreMatcher := scanner.NewIgnoreMatcher(repoRoot, cfg.Scan.Exclude, cfg.Scan.IgnoreFile)

	parsedCount := 0
	for _, relPath := range changed {
		relPath = filepath.ToSlash(relPath)

		// A raw `git diff` file list bypasses Scan's exclude/.gitignore
		// rules entirely. Without this check, Repo Mapper's own output
		// (.repo-mapper/) and cache (.cache/) files — modified by the very last run
		// — would show up as "changed" on the next `update`, wastefully
		// reprocessing them (and, since they contain hashes that change
		// every run, causing them to reappear as "changed" forever).
		if ignoreMatcher.Matches(relPath, false) {
			continue
		}

		absPath := filepath.Join(repoRoot, filepath.FromSlash(relPath))

		info, statErr := os.Stat(absPath)
		if statErr != nil || info.IsDir() {
			// File deleted or is a directory entry in the diff.
			c.Forget(relPath)
			continue
		}

		hash, herr := scanner.HashFile(absPath)
		if herr != nil {
			continue
		}
		if c.Unchanged(relPath, hash) {
			continue
		}

		res := parser.Dispatch(plugins, log, repoRoot, relPath, absPath)
		if res.Err != nil {
			if log != nil {
				log.Warn("failed to read file", "file", relPath, "error", res.Err)
			}
			continue
		}
		c.Put(relPath, hash, res.Entities)
		parsedCount++
	}

	if err := c.Save(); err != nil {
		return nil, fmt.Errorf("save cache: %w", err)
	}

	// Rebuild the full model from the (now up-to-date) cache. Language
	// stats are recomputed from cached file paths since we didn't walk the
	// tree.
	var entities []model.Entity
	var files []scanner.File
	paths := make([]string, 0, len(c.Entities))
	for p := range c.Entities {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		entities = append(entities, c.Entities[p]...)
		files = append(files, scanner.File{Path: p, Language: scanner.ClassifyLanguage(p)})
	}

	repoModel := buildRepository(repoRoot, files, entities)
	return &Result{Repository: repoModel, FilesScanned: len(files), FilesParsed: parsedCount}, nil
}

func enabledPlugins(cfg *config.Config) []plugin.Plugin {
	all := plugin.All()
	var out []plugin.Plugin
	for _, p := range all {
		if cfg.PluginEnabled(p.Name()) {
			out = append(out, p)
		}
	}
	return out
}

func buildRepository(repoRoot string, files []scanner.File, entities []model.Entity) *model.Repository {
	langCounts := map[string]int{}
	for _, f := range files {
		if f.Language == "" {
			continue
		}
		langCounts[f.Language]++
	}
	langNames := make([]string, 0, len(langCounts))
	for name := range langCounts {
		langNames = append(langNames, name)
	}
	sort.Strings(langNames)
	languages := make([]model.Language, 0, len(langNames))
	for _, name := range langNames {
		languages = append(languages, model.Language{Name: name, FileCount: langCounts[name]})
	}

	gitInfo := model.GitInfo{}
	repo := gitmod.New(repoRoot)
	if repo.IsRepo() {
		if branch, err := repo.Branch(); err == nil {
			gitInfo.Branch = branch
		}
		if hash, err := repo.CommitHash(); err == nil {
			gitInfo.CommitHash = hash
		}
	}

	name := filepath.Base(repoRoot)
	return analyzer.Build(name, repoRoot, languages, entities, gitInfo)
}
