package node

import (
	"testing"

	"github.com/anushibinj/repo-mapper/internal/plugin"
)

func TestParse_PackageJSON(t *testing.T) {
	src := `{
  "name": "billing-frontend",
  "scripts": { "dev": "vite", "build": "vite build" },
  "dependencies": { "react": "^18.2.0", "axios": "^1.6.0" },
  "devDependencies": { "vite": "^5.0.0" }
}`
	p := New()
	entities, err := p.Parse(plugin.Context{RelPath: "frontend/package.json"}, []byte(src))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	e := entities[0]
	if e.Name != "billing-frontend" {
		t.Errorf("expected Name=billing-frontend, got %q", e.Name)
	}
	if len(e.Refs) != 2 || e.Refs[0] != "axios" || e.Refs[1] != "react" {
		t.Errorf("expected sorted refs=[axios,react], got %v", e.Refs)
	}
	if e.Attributes["scripts"] != "build,dev" {
		t.Errorf("expected sorted scripts=build,dev, got %q", e.Attributes["scripts"])
	}
}

func TestParse_PackageJSON_FallsBackToDirNameWhenNameMissing(t *testing.T) {
	p := New()
	entities, err := p.Parse(plugin.Context{RelPath: "frontend/package.json"}, []byte(`{}`))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if entities[0].Name != "frontend" {
		t.Errorf("expected Name=frontend (dir fallback), got %q", entities[0].Name)
	}
}

func TestCanParse(t *testing.T) {
	p := New()
	if !p.CanParse("frontend/package.json") {
		t.Error("expected CanParse(package.json) = true")
	}
	if p.CanParse("frontend/package-lock.json") {
		t.Error("expected CanParse(package-lock.json) = false")
	}
}
