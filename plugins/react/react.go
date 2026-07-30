// Package react is a heuristic plugin for React/JSX/TSX source, extracting
// components, hook usage, outbound API calls (fetch/axios), and
// react-router route declarations (PRD section 12, "React: Route ->
// Component -> API Calls").
package react

import (
	"regexp"
	"strings"

	"github.com/anushibinj/repo-mapper/internal/model"
	"github.com/anushibinj/repo-mapper/internal/plugin"
)

func init() {
	plugin.Register(New())
}

// Plugin is the React framework plugin.
type Plugin struct{}

// New constructs a React Plugin.
func New() *Plugin { return &Plugin{} }

// Name implements plugin.Plugin.
func (p *Plugin) Name() string { return "react" }

// CanParse implements plugin.Plugin.
func (p *Plugin) CanParse(file string) bool {
	lower := strings.ToLower(file)
	return strings.HasSuffix(lower, ".jsx") || strings.HasSuffix(lower, ".tsx") ||
		((strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".ts")) && looksLikeReactFileName(lower))
}

func looksLikeReactFileName(lower string) bool {
	base := lower
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	// Convention: hook files are named useXxx.ts/useXxx.tsx.
	isHookFile := strings.HasPrefix(base, "use") && len(base) > len("use")
	return strings.Contains(lower, "component") || strings.Contains(lower, "page") ||
		strings.Contains(lower, "hook") || strings.Contains(lower, "container") || isHookFile
}

var (
	// function Foo(...) { ... return (<JSX>) }  OR  const Foo = (...) => { ... }
	funcComponentRe  = regexp.MustCompile(`\bfunction\s+([A-Z]\w*)\s*\(`)
	constComponentRe = regexp.MustCompile(`\bconst\s+([A-Z]\w*)\s*(?::\s*React\.FC[^=]*)?=\s*(?:\([^)]*\)|[\w]+)\s*=>`)
	classComponentRe = regexp.MustCompile(`\bclass\s+(\w+)\s+extends\s+(?:React\.)?Component`)
	hookUsageRe      = regexp.MustCompile(`\b(use[A-Z]\w*)\s*\(`)
	customHookDeclRe = regexp.MustCompile(`\bfunction\s+(use[A-Z]\w*)\s*\(`)
	importRe         = regexp.MustCompile(`import\s+(?:[\w*{}, \n]+)\s+from\s+['"]([^'"]+)['"]`)
	fetchCallRe      = regexp.MustCompile(`fetch\(\s*['"\x60]([^'"\x60]+)['"\x60]`)
	axiosCallRe      = regexp.MustCompile(`axios\.(get|post|put|delete|patch)\(\s*['"\x60]([^'"\x60]+)['"\x60]`)
	routeElementRe   = regexp.MustCompile(`<Route\s+[^>]*path\s*=\s*["{]([^"}]+)["}][^>]*(?:element\s*=\s*\{?\s*<\s*(\w+)|component\s*=\s*\{?\s*(\w+))`)
)

// Parse implements plugin.Plugin.
func (p *Plugin) Parse(ctx plugin.Context, content []byte) ([]model.Entity, error) {
	src := string(content)
	var entities []model.Entity

	components := map[string]bool{}
	for _, m := range funcComponentRe.FindAllStringSubmatch(src, -1) {
		components[m[1]] = true
	}
	for _, m := range constComponentRe.FindAllStringSubmatch(src, -1) {
		components[m[1]] = true
	}
	for _, m := range classComponentRe.FindAllStringSubmatch(src, -1) {
		components[m[1]] = true
	}

	var hooks []string
	seenHooks := map[string]bool{}
	for _, m := range hookUsageRe.FindAllStringSubmatch(src, -1) {
		if !seenHooks[m[1]] {
			seenHooks[m[1]] = true
			hooks = append(hooks, m[1])
		}
	}

	var apiCalls []string
	for _, m := range fetchCallRe.FindAllStringSubmatch(src, -1) {
		apiCalls = append(apiCalls, m[1])
	}
	for _, m := range axiosCallRe.FindAllStringSubmatch(src, -1) {
		apiCalls = append(apiCalls, m[2])
	}

	var imports []string
	for _, m := range importRe.FindAllStringSubmatch(src, -1) {
		imports = append(imports, m[1])
	}

	if len(components) == 0 {
		// Might still be a custom-hook-only file.
		for _, m := range customHookDeclRe.FindAllStringSubmatch(src, -1) {
			entities = append(entities, model.Entity{
				Kind:     "react-hook",
				Name:     m[1],
				File:     ctx.RelPath,
				Language: "TypeScript/JavaScript",
				Attributes: map[string]string{
					"apiCalls": strings.Join(apiCalls, ","),
				},
				Refs: imports,
			})
		}
	}

	for name := range components {
		entities = append(entities, model.Entity{
			Kind:     "react-component",
			Name:     name,
			File:     ctx.RelPath,
			Language: "TypeScript/JavaScript",
			Attributes: map[string]string{
				"hooks":    strings.Join(hooks, ","),
				"apiCalls": strings.Join(apiCalls, ","),
			},
			Refs: imports,
		})
	}

	for _, m := range routeElementRe.FindAllStringSubmatch(src, -1) {
		routePath := m[1]
		comp := m[2]
		if comp == "" {
			comp = m[3]
		}
		entities = append(entities, model.Entity{
			Kind:     "react-route",
			Name:     comp,
			File:     ctx.RelPath,
			Language: "TypeScript/JavaScript",
			Attributes: map[string]string{
				"path":      routePath,
				"component": comp,
			},
		})
	}

	return entities, nil
}
