// Package plugin defines the contract that every language/framework plugin
// implements, plus a static, compile-time registry.
//
// Plugins are NOT dynamically loaded (Go's `plugin` package does not work on
// Windows, which conflicts with the cross-platform goal). Instead, each
// plugin package under plugins/ registers itself via an init() function:
//
//	func init() { plugin.Register(New()) }
//
// The core engine (scanner, parser, analyzer) only ever depends on this
// package's Plugin interface and Registry — never on a concrete plugin
// implementation.
package plugin

import "github.com/anushibinj/repo-mapper/internal/model"

// Context carries per-run information plugins may need while parsing a file.
type Context struct {
	// RepoRoot is the absolute path to the repository root.
	RepoRoot string
	// RelPath is the file path relative to RepoRoot.
	RelPath string
}

// Plugin is implemented by every language/framework analyzer.
type Plugin interface {
	// Name returns a short, stable, unique identifier (e.g. "java", "spring").
	Name() string

	// CanParse reports whether this plugin knows how to handle the given
	// file path (relative to the repository root).
	CanParse(file string) bool

	// Parse extracts entities from the given file's contents.
	Parse(ctx Context, content []byte) ([]model.Entity, error)
}

var registry = map[string]Plugin{}
var order []string

// Register adds a plugin to the global registry. Intended to be called from
// plugin package init() functions. Panics on duplicate names, which would
// indicate a programming error, not a runtime condition.
func Register(p Plugin) {
	name := p.Name()
	if _, exists := registry[name]; exists {
		panic("plugin: duplicate plugin registered: " + name)
	}
	registry[name] = p
	order = append(order, name)
}

// All returns every registered plugin, in registration order (deterministic
// given a fixed set of imported plugin packages).
func All() []Plugin {
	plugins := make([]Plugin, 0, len(order))
	for _, name := range order {
		plugins = append(plugins, registry[name])
	}
	return plugins
}

// Get returns the plugin registered under name, if any.
func Get(name string) (Plugin, bool) {
	p, ok := registry[name]
	return p, ok
}

// Names returns the names of all registered plugins, in registration order.
func Names() []string {
	out := make([]string, len(order))
	copy(out, order)
	return out
}

// Reset clears the registry. Exposed for tests only.
func Reset() {
	registry = map[string]Plugin{}
	order = nil
}
