// Package vite detects Vite-based frontend projects via their config file.
package vite

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/anushibinj/repo-mapper/internal/model"
	"github.com/anushibinj/repo-mapper/internal/plugin"
)

func init() {
	plugin.Register(New())
}

// Plugin is the Vite bundler plugin.
type Plugin struct{}

// New constructs a Vite Plugin.
func New() *Plugin { return &Plugin{} }

// Name implements plugin.Plugin.
func (p *Plugin) Name() string { return "vite" }

// CanParse implements plugin.Plugin.
func (p *Plugin) CanParse(file string) bool {
	base := strings.ToLower(filepath.Base(file))
	return strings.HasPrefix(base, "vite.config.")
}

var pluginUsageRe = regexp.MustCompile(`plugins\s*:\s*\[([^\]]*)\]`)

// Parse implements plugin.Plugin.
func (p *Plugin) Parse(ctx plugin.Context, content []byte) ([]model.Entity, error) {
	src := string(content)
	attrs := map[string]string{}
	if m := pluginUsageRe.FindStringSubmatch(src); m != nil {
		attrs["plugins"] = strings.Join(strings.Fields(strings.ReplaceAll(m[1], ",", " ")), " ")
	}
	return []model.Entity{{
		Kind:       "vite-config",
		Name:       "vite",
		File:       ctx.RelPath,
		Language:   "Config",
		Attributes: attrs,
	}}, nil
}
