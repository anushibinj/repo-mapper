// Package parser dispatches discovered files to matching plugins and
// collects the entities they produce. It is fault tolerant: a plugin
// failure on one file is logged and skipped, never fatal (PRD section 23).
package parser

import (
	"os"

	"github.com/anushibinj/repo-mapper/internal/logger"
	"github.com/anushibinj/repo-mapper/internal/model"
	"github.com/anushibinj/repo-mapper/internal/plugin"
)

// Result is the outcome of parsing one file.
type Result struct {
	Path     string
	Entities []model.Entity
	Err      error
}

// Dispatch runs every plugin whose CanParse matches relPath against the
// file's content (read from absPath), returning the combined entities from
// all matching plugins. A single file may be handled by more than one
// plugin (e.g. a .java file matched by both the Java and Spring plugins).
func Dispatch(plugins []plugin.Plugin, log *logger.Logger, repoRoot, relPath, absPath string) Result {
	var matching []plugin.Plugin
	for _, p := range plugins {
		if p.CanParse(relPath) {
			matching = append(matching, p)
		}
	}
	if len(matching) == 0 {
		return Result{Path: relPath}
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return Result{Path: relPath, Err: err}
	}

	ctx := plugin.Context{RepoRoot: repoRoot, RelPath: relPath}
	var all []model.Entity
	for _, p := range matching {
		entities, err := p.Parse(ctx, content)
		if err != nil {
			if log != nil {
				log.Warn("plugin parse failed", "plugin", p.Name(), "file", relPath, "error", err)
			}
			continue
		}
		all = append(all, entities...)
	}
	return Result{Path: relPath, Entities: all}
}
