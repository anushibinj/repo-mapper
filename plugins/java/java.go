// Package java implements a lightweight, regex/heuristic-based Java plugin.
// It extracts package declarations, top-level type declarations (classes,
// interfaces, enums, records), their supertypes, and imports — enough
// structural signal for the analyzer to build a symbol graph without a full
// Java grammar/AST (see PRD section 11, "Parsing Strategy").
package java

import (
	"regexp"
	"strings"

	"github.com/anushibinj/repo-mapper/internal/model"
	"github.com/anushibinj/repo-mapper/internal/plugin"
)

func init() {
	plugin.Register(New())
}

// Plugin is the Java language plugin.
type Plugin struct{}

// New constructs a Java Plugin.
func New() *Plugin { return &Plugin{} }

// Name implements plugin.Plugin.
func (p *Plugin) Name() string { return "java" }

// CanParse implements plugin.Plugin.
func (p *Plugin) CanParse(file string) bool {
	return strings.HasSuffix(file, ".java")
}

var (
	packageRe = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)\s*;`)
	importRe  = regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?([\w.]+(?:\.\*)?)\s*;`)
	// Matches: [modifiers] class|interface|enum|record Name [<Generic>] [extends X] [implements Y, Z]
	typeDeclRe   = regexp.MustCompile(`(?m)(?:^|\n)((?:@\w+(?:\([^)]*\))?\s*)*)(?:public\s+|private\s+|protected\s+|final\s+|abstract\s+|static\s+)*\b(class|interface|enum|record)\s+(\w+)(?:\s*<[^>{]*>)?(?:\s+extends\s+([\w.<>, ]+?))?(?:\s+implements\s+([\w.<>, ]+?))?\s*[{(]`)
	annotationRe = regexp.MustCompile(`@(\w+)(?:\(([^)]*)\))?`)
)

// Parse implements plugin.Plugin.
func (p *Plugin) Parse(ctx plugin.Context, content []byte) ([]model.Entity, error) {
	src := string(content)

	pkg := ""
	if m := packageRe.FindStringSubmatch(src); m != nil {
		pkg = m[1]
	}

	var imports []string
	for _, m := range importRe.FindAllStringSubmatch(src, -1) {
		imports = append(imports, m[1])
	}

	var entities []model.Entity
	for _, m := range typeDeclRe.FindAllStringSubmatch(src, -1) {
		annotationsBlock := m[1]
		kind := m[2] // class|interface|enum|record
		name := m[3]
		extends := strings.TrimSpace(m[4])
		implements := strings.TrimSpace(m[5])

		var annotations []string
		for _, am := range annotationRe.FindAllStringSubmatch(annotationsBlock, -1) {
			annotations = append(annotations, am[1])
		}

		refs := append([]string{}, imports...)
		if extends != "" {
			refs = append(refs, splitTypeList(extends)...)
		}
		if implements != "" {
			refs = append(refs, splitTypeList(implements)...)
		}

		entities = append(entities, model.Entity{
			Kind:     "java-type",
			Name:     name,
			File:     ctx.RelPath,
			Package:  pkg,
			Language: "Java",
			Attributes: map[string]string{
				"declKind":    kind, // class, interface, enum, record
				"annotations": strings.Join(annotations, ","),
				"extends":     extends,
				"implements":  implements,
			},
			Refs: refs,
		})
	}

	return entities, nil
}

func splitTypeList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if idx := strings.IndexAny(part, "<"); idx >= 0 {
			part = part[:idx]
		}
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
