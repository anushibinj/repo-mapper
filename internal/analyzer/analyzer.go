// Package analyzer consumes the flat, plugin-produced model.Entity list and
// produces the canonical model.Repository: Components, Routes, APIs,
// Tables, and Features (PRD section 12). This is the only place semantic
// interpretation happens — plugins stay dumb and deterministic, the
// analyzer is where cross-entity relationships get resolved.
package analyzer

import (
	"sort"
	"strings"

	"github.com/anushibinj/repo-mapper/internal/model"
)

// Build turns a flat entity list plus scan-level metadata into the
// canonical Repository model.
func Build(name, rootPath string, languages []model.Language, entities []model.Entity, gitInfo model.GitInfo) *model.Repository {
	repo := &model.Repository{
		Name:      name,
		RootPath:  rootPath,
		Languages: languages,
		Git:       gitInfo,
	}

	componentsByKey := map[string]*model.Component{}
	var componentOrder []string

	tablesByName := map[string]*model.Table{}
	var tableOrder []string

	var routes []model.Route
	var apis []model.API
	var modules []model.Module
	seenModules := map[string]bool{}

	upsertComponent := func(key string, c model.Component) *model.Component {
		if existing, ok := componentsByKey[key]; ok {
			// Prefer more specific stereotypes (spring-*) over the generic
			// "java-type" entity for the same file+name. Deliberately not
			// merging DependsOn across ranks: java-type's DependsOn comes
			// from raw imports/extends/implements, which is a different
			// kind of signal than a Spring stereotype's @Autowired field
			// names, and mixing them pollutes the dependency graph (e.g.
			// "Autowired" or "*" showing up as a fake dependency).
			if rank(c.Kind) > rank(existing.Kind) {
				stored := c
				*existing = stored
			}
			return existing
		}
		stored := c
		componentsByKey[key] = &stored
		componentOrder = append(componentOrder, key)
		return &stored
	}

	for _, e := range entities {
		switch e.Kind {
		case "java-type":
			key := componentKey(e.File, e.Name)
			upsertComponent(key, model.Component{
				ID:       key,
				Name:     e.Name,
				Kind:     e.Attributes["declKind"],
				Language: "Java",
				File:     e.File,
				Package:  e.Package,
				// DependsOn intentionally left empty here: raw
				// imports/extends/implements are framework/JDK noise, not a
				// meaningful component dependency graph. Spring stereotypes
				// (which rank higher, see upsertComponent) supply the real
				// @Autowired-based dependency signal.
				Attributes: map[string]string{"annotations": e.Attributes["annotations"]},
			})

		case "spring-restcontroller", "spring-controller":
			key := componentKey(e.File, e.Name)
			upsertComponent(key, model.Component{
				ID:        key,
				Name:      e.Name,
				Kind:      "controller",
				Language:  "Java",
				File:      e.File,
				DependsOn: fieldNamesToGuess(e.Refs),
				Attributes: map[string]string{
					"basePath": e.Attributes["basePath"],
				},
			})

		case "spring-service":
			key := componentKey(e.File, e.Name)
			upsertComponent(key, model.Component{
				ID:        key,
				Name:      e.Name,
				Kind:      "service",
				Language:  "Java",
				File:      e.File,
				DependsOn: fieldNamesToGuess(e.Refs),
			})

		case "spring-repository":
			key := componentKey(e.File, e.Name)
			upsertComponent(key, model.Component{
				ID:       key,
				Name:     e.Name,
				Kind:     "repository",
				Language: "Java",
				File:     e.File,
			})

		case "spring-component":
			// Kind is "bean", not "component": the generic Spring @Component
			// stereotype would otherwise collide with the frontend/React
			// "component" concept used throughout this tool (PRD sections
			// 12, 14, 15), corrupting backend.md/frontend.md sections and
			// diagram filters that key off Kind == "component".
			key := componentKey(e.File, e.Name)
			upsertComponent(key, model.Component{
				ID:        key,
				Name:      e.Name,
				Kind:      "bean",
				Language:  "Java",
				File:      e.File,
				DependsOn: fieldNamesToGuess(e.Refs),
			})

		case "spring-entity":
			key := componentKey(e.File, e.Name)
			upsertComponent(key, model.Component{
				ID:       key,
				Name:     e.Name,
				Kind:     "entity",
				Language: "Java",
				File:     e.File,
				Attributes: map[string]string{
					"table": e.Attributes["table"],
				},
			})

			tableName := e.Attributes["table"]
			if tableName == "" {
				tableName = e.Name
			}
			t := getOrCreateTable(tablesByName, &tableOrder, tableName, e.File)
			relatesTo := splitCSV(e.Attributes["relatesTo"])
			t.RelatesTo = mergeUnique(t.RelatesTo, relatesTo)
			t.Entity = e.Name

		case "spring-route":
			apis = append(apis, model.API{
				Method:     e.Attributes["method"],
				Path:       e.Attributes["path"],
				Controller: e.Attributes["controller"],
				Handler:    e.Attributes["handler"],
				File:       e.File,
			})

		case "react-component":
			key := componentKey(e.File, e.Name)
			upsertComponent(key, model.Component{
				ID:       key,
				Name:     e.Name,
				Kind:     classifyReactComponent(e.File),
				Language: "TypeScript/JavaScript",
				File:     e.File,
				Calls:    splitCSV(e.Attributes["apiCalls"]),
				Attributes: map[string]string{
					"hooks": e.Attributes["hooks"],
				},
			})

		case "react-hook":
			key := componentKey(e.File, e.Name)
			upsertComponent(key, model.Component{
				ID:       key,
				Name:     e.Name,
				Kind:     "hook",
				Language: "TypeScript/JavaScript",
				File:     e.File,
				Calls:    splitCSV(e.Attributes["apiCalls"]),
			})

		case "react-route":
			routes = append(routes, model.Route{
				Path:      e.Attributes["path"],
				Component: e.Attributes["component"],
				File:      e.File,
			})

		case "sql-table":
			t := getOrCreateTable(tablesByName, &tableOrder, e.Name, e.File)
			t.RelatesTo = mergeUnique(t.RelatesTo, e.Refs)
			t.Columns = mergeColumns(t.Columns, extractColumns(e.Attributes))

		case "node-module":
			if !seenModules[e.File] {
				seenModules[e.File] = true
				modules = append(modules, model.Module{
					Name: e.Name,
					Path: dirOf(e.File),
					Kind: classifyNodeModule(e.Refs),
				})
			}

		case "docker-service":
			key := componentKey(e.File, e.Name)
			upsertComponent(key, model.Component{
				ID:       key,
				Name:     e.Name,
				Kind:     "infra-service",
				Language: "Docker",
				File:     e.File,
				Attributes: map[string]string{
					"image": e.Attributes["image"],
					"ports": e.Attributes["ports"],
				},
				DependsOn: e.Refs,
			})

		case "docker-image":
			key := componentKey(e.File, e.Name)
			upsertComponent(key, model.Component{
				ID:       key,
				Name:     e.Name,
				Kind:     "docker-image",
				Language: "Docker",
				File:     e.File,
				Attributes: map[string]string{
					"baseImage":    e.Attributes["baseImage"],
					"exposedPorts": e.Attributes["exposedPorts"],
				},
			})

		case "vite-config":
			// Surfaced via Modules/Attributes only; no standalone component needed.
		}
	}

	components := make([]model.Component, 0, len(componentOrder))
	for _, key := range componentOrder {
		components = append(components, *componentsByKey[key])
	}
	sort.Slice(components, func(i, j int) bool { return components[i].ID < components[j].ID })

	tables := make([]model.Table, 0, len(tableOrder))
	for _, name := range tableOrder {
		tables = append(tables, *tablesByName[name])
	}
	resolveEntityNamesToTableNames(components, tables)
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })

	sort.Slice(routes, func(i, j int) bool { return routes[i].Path < routes[j].Path })
	sort.Slice(apis, func(i, j int) bool {
		if apis[i].Path != apis[j].Path {
			return apis[i].Path < apis[j].Path
		}
		return apis[i].Method < apis[j].Method
	})
	sort.Slice(modules, func(i, j int) bool { return modules[i].Path < modules[j].Path })

	repo.Components = components
	repo.Tables = tables
	repo.Routes = routes
	repo.APIs = apis
	repo.Modules = modules
	repo.Features = buildFeatures(components, apis, tables)

	return repo
}

// rank gives spring stereotypes precedence over the generic "class" kind
// discovered by the plain Java plugin, so a class annotated
// @RestController shows up as "controller", not "class".
func rank(kind string) int {
	switch kind {
	case "controller", "service", "repository", "entity", "bean":
		return 2
	case "class", "interface", "enum", "record":
		return 1
	default:
		return 0
	}
}

func componentKey(file, name string) string {
	return file + "#" + name
}

func mergeUnique(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range append(append([]string{}, a...), b...) {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func splitCSV(s string) []string {
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

// resolveEntityNamesToTableNames normalises each Table's RelatesTo list: a
// spring-entity's field-based relation (e.g. "Customer", the Java class
// name) and a sql-table's FK-based relation (e.g. "customers", the SQL
// table name) may both describe the same real relationship. Where an
// "entity" component's name matches a RelatesTo entry, it is replaced with
// that entity's mapped table name so callers see one deduplicated relation
// instead of two spellings of the same thing.
func resolveEntityNamesToTableNames(components []model.Component, tables []model.Table) {
	entityToTable := map[string]string{}
	for _, c := range components {
		if c.Kind != "entity" {
			continue
		}
		table := c.Attributes["table"]
		if table == "" {
			table = c.Name
		}
		entityToTable[strings.ToLower(c.Name)] = table
	}
	if len(entityToTable) == 0 {
		return
	}

	for i := range tables {
		if len(tables[i].RelatesTo) == 0 {
			continue
		}
		resolved := make([]string, 0, len(tables[i].RelatesTo))
		for _, rel := range tables[i].RelatesTo {
			if table, ok := entityToTable[strings.ToLower(rel)]; ok {
				resolved = append(resolved, table)
			} else {
				resolved = append(resolved, rel)
			}
		}
		tables[i].RelatesTo = mergeUnique(resolved, nil)
	}
}

// fieldNamesToGuess turns @Autowired field names (e.g. "orderService") into
// a best-effort dependency reference.
func fieldNamesToGuess(refs []string) []string {
	return mergeUnique(refs, nil)
}

func classifyReactComponent(file string) string {
	lower := strings.ToLower(file)
	switch {
	case strings.Contains(lower, "page"):
		return "page"
	case strings.Contains(lower, "container"):
		return "container"
	default:
		return "component"
	}
}

func classifyNodeModule(deps []string) string {
	for _, d := range deps {
		if d == "react" || d == "vite" || d == "next" {
			return "frontend"
		}
		if d == "express" || d == "fastify" || d == "koa" {
			return "backend"
		}
	}
	return "node"
}

func dirOf(file string) string {
	idx := strings.LastIndex(file, "/")
	if idx < 0 {
		return "."
	}
	return file[:idx]
}

func getOrCreateTable(m map[string]*model.Table, order *[]string, name, file string) *model.Table {
	if t, ok := m[name]; ok {
		return t
	}
	t := &model.Table{Name: name, SourceFile: file}
	m[name] = t
	*order = append(*order, name)
	return t
}

func extractColumns(attrs map[string]string) []model.Column {
	var cols []model.Column
	for k, v := range attrs {
		if !strings.HasPrefix(k, "column:") {
			continue
		}
		name := strings.TrimPrefix(k, "column:")
		parts := strings.SplitN(v, ":", 3)
		col := model.Column{Name: name}
		if len(parts) > 0 {
			col.Type = parts[0]
		}
		if len(parts) > 1 {
			col.PrimaryKey = parts[1] == "1"
		}
		if len(parts) > 2 {
			col.ForeignKey = parts[2]
		}
		cols = append(cols, col)
	}
	sort.Slice(cols, func(i, j int) bool { return cols[i].Name < cols[j].Name })
	return cols
}

func mergeColumns(a, b []model.Column) []model.Column {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	byName := map[string]model.Column{}
	for _, c := range a {
		byName[c.Name] = c
	}
	for _, c := range b {
		byName[c.Name] = c
	}
	out := make([]model.Column, 0, len(byName))
	for _, c := range byName {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
