package docker

import (
	"testing"

	"github.com/anushibinj/repo-mapper/internal/plugin"
)

func TestParse_Dockerfile(t *testing.T) {
	src := `
FROM eclipse-temurin:21-jre
COPY target/app.jar /app.jar
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app.jar"]
`
	p := New()
	entities, err := p.Parse(plugin.Context{RelPath: "backend/Dockerfile"}, []byte(src))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d: %+v", len(entities), entities)
	}
	e := entities[0]
	if e.Kind != "docker-image" {
		t.Errorf("expected Kind=docker-image, got %q", e.Kind)
	}
	if e.Attributes["baseImage"] != "eclipse-temurin:21-jre" {
		t.Errorf("expected baseImage=eclipse-temurin:21-jre, got %q", e.Attributes["baseImage"])
	}
	if e.Attributes["exposedPorts"] != "8080" {
		t.Errorf("expected exposedPorts=8080, got %q", e.Attributes["exposedPorts"])
	}
}

func TestParse_ComposeFile(t *testing.T) {
	src := `
version: "3.9"
services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    depends_on:
      - db
  db:
    image: postgres:16
    ports:
      - "5432:5432"
`
	p := New()
	entities, err := p.Parse(plugin.Context{RelPath: "docker-compose.yml"}, []byte(src))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	byName := map[string]struct {
		image string
		ports string
		refs  []string
	}{}
	for _, e := range entities {
		byName[e.Name] = struct {
			image string
			ports string
			refs  []string
		}{e.Attributes["image"], e.Attributes["ports"], e.Refs}
	}
	if len(byName) != 2 {
		t.Fatalf("expected 2 services, got %d: %+v", len(byName), entities)
	}
	backend, ok := byName["backend"]
	if !ok {
		t.Fatalf("expected a backend service, got %+v", byName)
	}
	if len(backend.refs) != 1 || backend.refs[0] != "db" {
		t.Errorf("expected backend depends_on=[db], got %v", backend.refs)
	}
	db, ok := byName["db"]
	if !ok || db.image != "postgres:16" {
		t.Errorf("expected db service with image postgres:16, got %+v", db)
	}
}

func TestCanParse(t *testing.T) {
	p := New()
	cases := map[string]bool{
		"backend/Dockerfile":     true,
		"backend/Dockerfile.dev": true,
		"docker-compose.yml":     true,
		"compose.yaml":           true,
		"README.md":              false,
	}
	for path, want := range cases {
		if got := p.CanParse(path); got != want {
			t.Errorf("CanParse(%q) = %v, want %v", path, got, want)
		}
	}
}
