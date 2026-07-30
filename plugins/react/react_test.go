package react

import (
	"testing"

	"github.com/anushibinj/repo-mapper/internal/plugin"
)

func TestParse_FunctionComponentWithHookAndApiCall(t *testing.T) {
	src := `
import { useBilling } from '../hooks/useBilling';

export function BillingPage() {
  const { invoices } = useBilling();
  return <div>{invoices.length}</div>;
}
`
	p := New()
	entities, err := p.Parse(plugin.Context{RelPath: "BillingPage.tsx"}, []byte(src))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var comp *struct {
		name  string
		hooks string
	}
	for _, e := range entities {
		if e.Kind == "react-component" && e.Name == "BillingPage" {
			comp = &struct {
				name  string
				hooks string
			}{e.Name, e.Attributes["hooks"]}
		}
	}
	if comp == nil {
		t.Fatalf("expected react-component BillingPage, got %+v", entities)
	}
	if comp.hooks != "useBilling" {
		t.Errorf("expected hooks=useBilling, got %q", comp.hooks)
	}
}

func TestParse_CustomHookWithAxiosCall(t *testing.T) {
	src := `
import axios from 'axios';

export function useBilling() {
  axios.get('/billing/invoices');
  return {};
}
`
	p := New()
	entities, err := p.Parse(plugin.Context{RelPath: "useBilling.ts"}, []byte(src))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	found := false
	for _, e := range entities {
		if e.Kind == "react-hook" && e.Name == "useBilling" {
			found = true
			if e.Attributes["apiCalls"] != "/billing/invoices" {
				t.Errorf("expected apiCalls=/billing/invoices, got %q", e.Attributes["apiCalls"])
			}
		}
	}
	if !found {
		t.Fatalf("expected react-hook useBilling, got %+v", entities)
	}
}

func TestCanParse(t *testing.T) {
	p := New()
	cases := map[string]bool{
		"src/App.tsx":             true,
		"src/App.jsx":             true,
		"src/hooks/useBilling.ts": true,
		"src/pages/HomePage.js":   true,
		"src/utils/math.ts":       false,
		"src/index.css":           false,
	}
	for path, want := range cases {
		if got := p.CanParse(path); got != want {
			t.Errorf("CanParse(%q) = %v, want %v", path, got, want)
		}
	}
}
