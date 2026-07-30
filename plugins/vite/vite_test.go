package vite

import (
	"testing"

	"github.com/anushibinj/repo-mapper/internal/plugin"
)

func TestParse_ViteConfig(t *testing.T) {
	src := `
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
});
`
	p := New()
	entities, err := p.Parse(plugin.Context{RelPath: "frontend/vite.config.ts"}, []byte(src))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(entities) != 1 || entities[0].Kind != "vite-config" {
		t.Fatalf("expected 1 vite-config entity, got %+v", entities)
	}
}

func TestCanParse(t *testing.T) {
	p := New()
	cases := map[string]bool{
		"frontend/vite.config.ts":    true,
		"frontend/vite.config.js":    true,
		"frontend/webpack.config.js": false,
	}
	for path, want := range cases {
		if got := p.CanParse(path); got != want {
			t.Errorf("CanParse(%q) = %v, want %v", path, got, want)
		}
	}
}
