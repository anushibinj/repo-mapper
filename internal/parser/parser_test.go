package parser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/anushibinj/repo-mapper/internal/model"
	"github.com/anushibinj/repo-mapper/internal/plugin"
)

type fakePlugin struct {
	name      string
	canParse  func(string) bool
	entities  []model.Entity
	err       error
	callCount *int
}

func (f fakePlugin) Name() string              { return f.name }
func (f fakePlugin) CanParse(file string) bool { return f.canParse(file) }
func (f fakePlugin) Parse(ctx plugin.Context, content []byte) ([]model.Entity, error) {
	if f.callCount != nil {
		*f.callCount++
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.entities, nil
}

func TestDispatch_CombinesEntitiesFromMultipleMatchingPlugins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Foo.java")
	if err := os.WriteFile(path, []byte("class Foo {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	javaCalls := 0
	springCalls := 0
	plugins := []plugin.Plugin{
		fakePlugin{
			name:      "java",
			canParse:  func(string) bool { return true },
			entities:  []model.Entity{{Kind: "java-type", Name: "Foo"}},
			callCount: &javaCalls,
		},
		fakePlugin{
			name:      "spring",
			canParse:  func(string) bool { return true },
			entities:  []model.Entity{{Kind: "spring-service", Name: "Foo"}},
			callCount: &springCalls,
		},
	}

	result := Dispatch(plugins, nil, dir, "Foo.java", path)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Entities) != 2 {
		t.Fatalf("expected 2 combined entities, got %d: %+v", len(result.Entities), result.Entities)
	}
	if javaCalls != 1 || springCalls != 1 {
		t.Errorf("expected each matching plugin called exactly once, got java=%d spring=%d", javaCalls, springCalls)
	}
}

func TestDispatch_SkipsNonMatchingPlugins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Foo.java")
	if err := os.WriteFile(path, []byte("class Foo {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	tsCalls := 0
	plugins := []plugin.Plugin{
		fakePlugin{name: "typescript", canParse: func(f string) bool { return filepath.Ext(f) == ".ts" }, callCount: &tsCalls},
	}

	result := Dispatch(plugins, nil, dir, "Foo.java", path)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Entities) != 0 {
		t.Errorf("expected no entities, got %+v", result.Entities)
	}
	if tsCalls != 0 {
		t.Errorf("expected non-matching plugin never called, got %d calls", tsCalls)
	}
}

// TestDispatch_FaultTolerant verifies a plugin's parse error is swallowed
// (logged, not fatal) and does not prevent other plugins' results from
// being returned (PRD section 23: fault tolerance).
func TestDispatch_FaultTolerant(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Foo.java")
	if err := os.WriteFile(path, []byte("class Foo {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	plugins := []plugin.Plugin{
		fakePlugin{name: "broken", canParse: func(string) bool { return true }, err: errors.New("boom")},
		fakePlugin{name: "ok", canParse: func(string) bool { return true }, entities: []model.Entity{{Kind: "java-type", Name: "Foo"}}},
	}

	result := Dispatch(plugins, nil, dir, "Foo.java", path)
	if result.Err != nil {
		t.Fatalf("expected Dispatch to swallow the plugin's own error, got %v", result.Err)
	}
	if len(result.Entities) != 1 || result.Entities[0].Name != "Foo" {
		t.Errorf("expected the working plugin's entity to still be returned, got %+v", result.Entities)
	}
}

func TestDispatch_ReadErrorPropagates(t *testing.T) {
	dir := t.TempDir()
	missingPath := filepath.Join(dir, "does-not-exist.java")

	plugins := []plugin.Plugin{
		fakePlugin{name: "java", canParse: func(string) bool { return true }},
	}

	result := Dispatch(plugins, nil, dir, "does-not-exist.java", missingPath)
	if result.Err == nil {
		t.Error("expected Dispatch to return an error when the file cannot be read")
	}
}
