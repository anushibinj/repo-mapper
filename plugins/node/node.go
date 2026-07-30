// Package node parses package.json into module metadata: name, scripts,
// and dependencies. This gives the analyzer a Node/npm module boundary and
// its declared dependency set.
package node

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anushibinj/repo-mapper/internal/model"
	"github.com/anushibinj/repo-mapper/internal/plugin"
)

func init() {
	plugin.Register(New())
}

// Plugin is the Node/npm plugin.
type Plugin struct{}

// New constructs a Node Plugin.
func New() *Plugin { return &Plugin{} }

// Name implements plugin.Plugin.
func (p *Plugin) Name() string { return "node" }

// CanParse implements plugin.Plugin.
func (p *Plugin) CanParse(file string) bool {
	return strings.EqualFold(filepath.Base(file), "package.json")
}

type packageJSON struct {
	Name            string            `json:"name"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// Parse implements plugin.Plugin.
func (p *Plugin) Parse(ctx plugin.Context, content []byte) ([]model.Entity, error) {
	var pkg packageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, err
	}

	name := pkg.Name
	if name == "" {
		name = filepath.Base(filepath.Dir(ctx.RelPath))
		if name == "." || name == "" {
			name = "root"
		}
	}

	deps := make([]string, 0, len(pkg.Dependencies))
	for d := range pkg.Dependencies {
		deps = append(deps, d)
	}
	sort.Strings(deps)

	scripts := make([]string, 0, len(pkg.Scripts))
	for s := range pkg.Scripts {
		scripts = append(scripts, s)
	}
	sort.Strings(scripts)

	return []model.Entity{{
		Kind:     "node-module",
		Name:     name,
		File:     ctx.RelPath,
		Language: "Node",
		Attributes: map[string]string{
			"scripts": strings.Join(scripts, ","),
		},
		Refs: deps,
	}}, nil
}
