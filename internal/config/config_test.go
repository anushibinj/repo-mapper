package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault_HasSaneDefaults(t *testing.T) {
	cfg := Default()
	if cfg.Output.Directory != ".ai" {
		t.Errorf("expected default output directory .ai, got %q", cfg.Output.Directory)
	}
	if cfg.LLM.Enabled {
		t.Error("expected LLM disabled by default")
	}
	for _, name := range []string{"java", "spring", "react", "vite", "node", "docker", "sql"} {
		if !cfg.PluginEnabled(name) {
			t.Errorf("expected plugin %q enabled by default", name)
		}
	}
	if !cfg.PluginEnabled("some-unknown-plugin") {
		t.Error("expected unmentioned plugins to default to enabled")
	}
	found := false
	for _, e := range cfg.Scan.Exclude {
		if e == "node_modules" {
			found = true
		}
	}
	if !found {
		t.Error("expected node_modules in default excludes")
	}
}

func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed on missing config: %v", err)
	}
	if cfg.Output.Directory != ".ai" {
		t.Errorf("expected default output directory, got %q", cfg.Output.Directory)
	}
}

func TestLoad_MergesFileOverDefaults(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `
output:
  directory: custom-ai
plugins:
  sql: false
`
	if err := os.WriteFile(filepath.Join(dir, DefaultFileName), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Output.Directory != "custom-ai" {
		t.Errorf("expected overridden output directory, got %q", cfg.Output.Directory)
	}
	if cfg.PluginEnabled("sql") {
		t.Error("expected sql plugin disabled by config override")
	}
	// Plugins not mentioned in the file should still resolve via defaults.
	if !cfg.PluginEnabled("java") {
		t.Error("expected java plugin still enabled")
	}
}
