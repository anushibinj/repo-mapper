package generator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/anushibinj/repo-mapper/internal/model"
)

// WriteMermaid renders system.mmd, backend.mmd, frontend.mmd, auth.mmd, and
// database.mmd under diagrams/ (PRD section 14). Each diagram is small and
// focused — never one giant graph.
func WriteMermaid(repo *model.Repository, outputDir string) error {
	dir := filepath.Join(outputDir, "diagrams")

	if err := writeFile(filepath.Join(dir, "system.mmd"), renderSystemDiagram(repo)); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(dir, "backend.mmd"), renderBackendDiagram(repo)); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(dir, "frontend.mmd"), renderFrontendDiagram(repo)); err != nil {
		return err
	}
	if diagram, ok := renderAuthDiagram(repo); ok {
		if err := writeFile(filepath.Join(dir, "auth.mmd"), diagram); err != nil {
			return err
		}
	}
	if err := writeFile(filepath.Join(dir, "database.mmd"), renderDatabaseDiagram(repo)); err != nil {
		return err
	}
	return nil
}

// SystemDiagram renders the system-level Mermaid diagram without writing
// it to disk. Used by the `graph` CLI command and available for future
// integrations (e.g. an MCP server) that want diagram text directly.
func SystemDiagram(repo *model.Repository) string { return renderSystemDiagram(repo) }

// BackendDiagram renders the backend-layer Mermaid diagram.
func BackendDiagram(repo *model.Repository) string { return renderBackendDiagram(repo) }

// FrontendDiagram renders the frontend-layer Mermaid diagram.
func FrontendDiagram(repo *model.Repository) string { return renderFrontendDiagram(repo) }

// DatabaseDiagram renders the database ER Mermaid diagram.
func DatabaseDiagram(repo *model.Repository) string { return renderDatabaseDiagram(repo) }

// AuthDiagram renders the authentication flow Mermaid diagram, if any
// auth-related components were detected.
func AuthDiagram(repo *model.Repository) (string, bool) { return renderAuthDiagram(repo) }

func renderSystemDiagram(repo *model.Repository) string {
	var b strings.Builder
	b.WriteString("flowchart TD\n")

	hasFrontend := hasAnyComponent(repo.Components, "page", "container", "component", "hook")
	hasBackend := hasAnyComponent(repo.Components, "controller", "service", "repository")
	hasDB := len(repo.Tables) > 0

	if hasFrontend {
		b.WriteString("    Frontend[Frontend]\n")
	}
	if hasBackend {
		b.WriteString("    Backend[Backend]\n")
	}
	if hasDB {
		b.WriteString("    Database[(Database)]\n")
	}
	if hasFrontend && hasBackend {
		b.WriteString("    Frontend --> Backend\n")
	}
	if hasBackend && hasDB {
		b.WriteString("    Backend --> Database\n")
	}
	for _, m := range repo.Modules {
		if m.Kind == "infra" {
			fmt.Fprintf(&b, "    %s[%s]\n", mermaidID(m.Name), m.Name)
		}
	}
	if !hasFrontend && !hasBackend && !hasDB {
		b.WriteString("    Repository[Repository]\n")
	}
	return b.String()
}

func renderBackendDiagram(repo *model.Repository) string {
	var b strings.Builder
	b.WriteString("flowchart TD\n")

	controllers := filterComponents(repo.Components, "controller")
	services := filterComponents(repo.Components, "service")
	repos := filterComponents(repo.Components, "repository")

	if len(controllers)+len(services)+len(repos) == 0 {
		b.WriteString("    NoBackend[No backend components detected]\n")
		return b.String()
	}

	byName := map[string]model.Component{}
	for _, c := range append(append(append([]model.Component{}, controllers...), services...), repos...) {
		byName[c.Name] = c
	}

	var edges []string
	seenEdge := map[string]bool{}
	for _, c := range controllers {
		for _, depField := range c.DependsOn {
			if target := resolveByFieldName(byName, depField); target != "" {
				edge := fmt.Sprintf("    %s --> %s", mermaidID(c.Name), mermaidID(target))
				if !seenEdge[edge] {
					seenEdge[edge] = true
					edges = append(edges, edge)
				}
			}
		}
	}
	for _, c := range services {
		for _, depField := range c.DependsOn {
			if target := resolveByFieldName(byName, depField); target != "" {
				edge := fmt.Sprintf("    %s --> %s", mermaidID(c.Name), mermaidID(target))
				if !seenEdge[edge] {
					seenEdge[edge] = true
					edges = append(edges, edge)
				}
			}
		}
	}

	for _, c := range controllers {
		fmt.Fprintf(&b, "    %s[%s]\n", mermaidID(c.Name), c.Name)
	}
	for _, c := range services {
		fmt.Fprintf(&b, "    %s(%s)\n", mermaidID(c.Name), c.Name)
	}
	for _, c := range repos {
		fmt.Fprintf(&b, "    %s[(%s)]\n", mermaidID(c.Name), c.Name)
	}
	sort.Strings(edges)
	for _, e := range edges {
		b.WriteString(e + "\n")
	}
	return b.String()
}

func renderFrontendDiagram(repo *model.Repository) string {
	var b strings.Builder
	b.WriteString("flowchart TD\n")

	pages := filterComponents(repo.Components, "page")
	containers := filterComponents(repo.Components, "container")
	components := filterComponents(repo.Components, "component")
	hooks := filterComponents(repo.Components, "hook")

	if len(pages)+len(containers)+len(components)+len(hooks) == 0 {
		b.WriteString("    NoFrontend[No frontend components detected]\n")
		return b.String()
	}

	for _, c := range pages {
		fmt.Fprintf(&b, "    %s[%s]\n", mermaidID(c.Name), c.Name)
	}
	for _, c := range containers {
		fmt.Fprintf(&b, "    %s(%s)\n", mermaidID(c.Name), c.Name)
	}
	for _, c := range components {
		fmt.Fprintf(&b, "    %s(%s)\n", mermaidID(c.Name), c.Name)
	}
	for _, c := range hooks {
		fmt.Fprintf(&b, "    %s{{%s}}\n", mermaidID(c.Name), c.Name)
	}

	b.WriteString("    REST[[REST API]]\n")

	var edges []string
	all := append(append(append(append([]model.Component{}, pages...), containers...), components...), hooks...)
	for _, c := range all {
		for _, hook := range splitCSVLocal(c.Attributes["hooks"]) {
			edges = append(edges, fmt.Sprintf("    %s --> %s", mermaidID(c.Name), mermaidID(hook)))
		}
		for range c.Calls {
			edges = append(edges, fmt.Sprintf("    %s --> REST", mermaidID(c.Name)))
			break // one edge to REST is enough per component
		}
	}
	sort.Strings(edges)
	seen := map[string]bool{}
	for _, e := range edges {
		if seen[e] {
			continue
		}
		seen[e] = true
		b.WriteString(e + "\n")
	}
	return b.String()
}

// renderAuthDiagram only produces output when auth-related components are
// detected by name heuristic, matching PRD section 14's "Authentication —
// Flow Diagram" as an example diagram, not a mandatory one.
func renderAuthDiagram(repo *model.Repository) (string, bool) {
	authRe := regexp.MustCompile(`(?i)auth|login|security|jwt|token`)
	var authComponents []model.Component
	for _, c := range repo.Components {
		if authRe.MatchString(c.Name) {
			authComponents = append(authComponents, c)
		}
	}
	if len(authComponents) == 0 {
		return "", false
	}

	var b strings.Builder
	b.WriteString("flowchart LR\n")
	b.WriteString("    Client[Client]\n")
	for _, c := range authComponents {
		fmt.Fprintf(&b, "    %s[%s]\n", mermaidID(c.Name), c.Name)
	}
	prev := "Client"
	for _, c := range authComponents {
		fmt.Fprintf(&b, "    %s --> %s\n", prev, mermaidID(c.Name))
		prev = mermaidID(c.Name)
	}
	return b.String(), true
}

func renderDatabaseDiagram(repo *model.Repository) string {
	var b strings.Builder
	b.WriteString("erDiagram\n")
	if len(repo.Tables) == 0 {
		b.WriteString("    %% No tables detected\n")
		return b.String()
	}

	for _, t := range repo.Tables {
		id := mermaidID(t.Name)
		if len(t.Columns) == 0 {
			fmt.Fprintf(&b, "    %s {\n        string id\n    }\n", id)
			continue
		}
		fmt.Fprintf(&b, "    %s {\n", id)
		for _, c := range t.Columns {
			colType := c.Type
			if colType == "" {
				colType = "string"
			}
			fmt.Fprintf(&b, "        %s %s\n", mermaidType(colType), c.Name)
		}
		b.WriteString("    }\n")
	}

	seenRel := map[string]bool{}
	for _, t := range repo.Tables {
		for _, related := range t.RelatesTo {
			relID := mermaidID(related)
			key := t.Name + "->" + related
			if seenRel[key] {
				continue
			}
			seenRel[key] = true
			fmt.Fprintf(&b, "    %s ||--o{ %s : \"references\"\n", mermaidID(t.Name), relID)
		}
	}
	return b.String()
}

// mermaidID sanitises a name into a Mermaid-safe node identifier.
var idSanitizeRe = regexp.MustCompile(`[^A-Za-z0-9_]`)

func mermaidID(name string) string {
	if name == "" {
		return "Unknown"
	}
	return idSanitizeRe.ReplaceAllString(name, "_")
}

// mermaidType collapses SQL/Java type spellings to simple ER-diagram type
// tokens Mermaid expects (no spaces/parens).
func mermaidType(t string) string {
	t = strings.SplitN(t, "(", 2)[0]
	t = strings.TrimSpace(t)
	t = strings.ReplaceAll(t, " ", "_")
	if t == "" {
		return "string"
	}
	return t
}

func resolveByFieldName(byName map[string]model.Component, fieldName string) string {
	// Heuristic: an @Autowired field like "orderService" of type OrderService
	// should resolve to the component literally named "OrderService". Since
	// the analyzer only has the field name (not always the declared type),
	// try a case-insensitive suffix/prefix match against known component
	// names as a best effort. Names are sorted before matching so the result
	// is deterministic regardless of Go's randomised map iteration order.
	lowerField := strings.ToLower(fieldName)
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if strings.EqualFold(name, fieldName) {
			return name
		}
	}
	for _, name := range names {
		lowerName := strings.ToLower(name)
		if strings.Contains(lowerName, lowerField) || strings.Contains(lowerField, lowerName) {
			return name
		}
	}
	return ""
}

func splitCSVLocal(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
