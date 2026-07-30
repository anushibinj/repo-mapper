// Package model defines the canonical repository model — the single source
// of truth from which every generator (Markdown, Mermaid, JSON) renders its
// output. Nothing outside this package should be treated as authoritative.
package model

// Repository is the root of the canonical model.
type Repository struct {
	Name       string      `json:"name"`
	RootPath   string      `json:"rootPath"`
	Languages  []Language  `json:"languages"`
	Modules    []Module    `json:"modules"`
	Features   []Feature   `json:"features"`
	Components []Component `json:"components"`
	Routes     []Route     `json:"routes"`
	APIs       []API       `json:"apis"`
	Tables     []Table     `json:"tables"`
	Git        GitInfo     `json:"git,omitempty"`
}

// Language represents a detected programming language and its footprint.
type Language struct {
	Name      string `json:"name"`
	FileCount int    `json:"fileCount"`
}

// Module represents a coarse-grained unit of the repository (e.g. a Maven
// module, an npm workspace package, or a top-level directory).
type Module struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Kind string `json:"kind"` // e.g. "backend", "frontend", "infra"
}

// Component is any structural building block discovered by a plugin:
// a Java class, a Spring controller/service/repository, a React
// component/hook, a Docker service, etc. Layer/Kind disambiguate roles.
type Component struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Kind       string            `json:"kind"` // controller, service, repository, entity, component, hook, page, container, config
	Language   string            `json:"language"`
	File       string            `json:"file"`
	Package    string            `json:"package,omitempty"`
	DependsOn  []string          `json:"dependsOn,omitempty"` // IDs of other components
	Calls      []string          `json:"calls,omitempty"`     // API paths called (frontend -> backend)
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Route represents a frontend route (React Router, etc.).
type Route struct {
	Path      string `json:"path"`
	Component string `json:"component"`
	File      string `json:"file"`
}

// API represents a backend HTTP endpoint.
type API struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	Controller string `json:"controller"`
	Handler    string `json:"handler"`
	File       string `json:"file"`
}

// Table represents a database table/entity and its relationships.
type Table struct {
	Name       string   `json:"name"`
	Entity     string   `json:"entity,omitempty"`
	Columns    []Column `json:"columns,omitempty"`
	RelatesTo  []string `json:"relatesTo,omitempty"`
	SourceFile string   `json:"sourceFile"`
}

// Column represents a single database column.
type Column struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	PrimaryKey bool   `json:"primaryKey,omitempty"`
	ForeignKey string `json:"foreignKey,omitempty"`
}

// Feature groups related frontend/backend/database components together
// (see AI Manifest, PRD section 15).
type Feature struct {
	Name     string   `json:"feature"`
	Frontend []string `json:"frontend,omitempty"`
	Backend  []string `json:"backend,omitempty"`
	APIs     []string `json:"apis,omitempty"`
	Database []string `json:"database,omitempty"`
}

// GitInfo captures repository VCS metadata at time of generation.
type GitInfo struct {
	Branch     string `json:"branch,omitempty"`
	CommitHash string `json:"commitHash,omitempty"`
}

// Entity is the raw, plugin-produced unit of information before analysis
// turns it into Components/Routes/APIs/Tables/Features. Plugins only ever
// produce Entities; the Analyzer is solely responsible for interpreting
// them into the canonical model.
type Entity struct {
	Kind       string            // e.g. "java-class", "spring-controller", "react-component", "sql-table"
	Name       string            // symbol name
	File       string            // source file path (relative to repo root)
	Package    string            // package/namespace, if any
	Language   string            // language plugin that produced it
	Attributes map[string]string // free-form structured data (annotations, methods, routes, etc.)
	Refs       []string          // referenced/related symbol names (imports, calls, extends)
}
