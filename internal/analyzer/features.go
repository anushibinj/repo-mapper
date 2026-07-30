package analyzer

import (
	"sort"
	"strings"

	"github.com/anushibinj/repo-mapper/internal/model"
)

// componentStemSuffixes are stripped from component names to derive a
// feature "stem" used for cross-layer grouping (e.g. "BillingController",
// "BillingService", and "BillingPage" all stem to "billing").
var componentStemSuffixes = []string{
	"Controller", "RestController", "Service", "ServiceImpl", "Repository",
	"Entity", "Page", "Container", "Component", "View", "Hook",
}

func stemOf(name string) string {
	for _, suf := range componentStemSuffixes {
		if strings.HasSuffix(name, suf) && len(name) > len(suf) {
			return strings.ToLower(strings.TrimSuffix(name, suf))
		}
	}
	return strings.ToLower(name)
}

// buildFeatures groups components, APIs, and tables into Features by naming
// stem, giving the AI Manifest (PRD section 15) its feature-oriented view.
// This is a heuristic, not a guarantee — features with entirely unrelated
// naming conventions across layers won't be grouped, which is an accepted
// Phase 1 limitation (see Roadmap for AST/semantic upgrades).
func buildFeatures(components []model.Component, apis []model.API, tables []model.Table) []model.Feature {
	type bucket struct {
		frontend map[string]bool
		backend  map[string]bool
		apis     map[string]bool
		database map[string]bool
	}
	buckets := map[string]*bucket{}
	var order []string

	getBucket := func(stem string) *bucket {
		if b, ok := buckets[stem]; ok {
			return b
		}
		b := &bucket{
			frontend: map[string]bool{},
			backend:  map[string]bool{},
			apis:     map[string]bool{},
			database: map[string]bool{},
		}
		buckets[stem] = b
		order = append(order, stem)
		return b
	}

	controllerStems := map[string]string{} // controllerName -> stem, for API attribution

	for _, c := range components {
		stem := stemOf(c.Name)
		if stem == "" {
			continue
		}
		b := getBucket(stem)
		switch c.Kind {
		case "controller", "service", "repository", "entity", "bean":
			b.backend[c.Name] = true
			if c.Kind == "controller" {
				controllerStems[c.Name] = stem
			}
		case "page", "container", "component", "hook":
			b.frontend[c.Name] = true
		}
	}

	for _, a := range apis {
		stem, ok := controllerStems[a.Controller]
		if !ok {
			stem = stemOf(a.Controller)
		}
		if stem == "" {
			continue
		}
		b := getBucket(stem)
		b.apis[a.Path] = true
	}

	for _, t := range tables {
		name := t.Entity
		if name == "" {
			name = t.Name
		}
		stem := stemOf(name)
		if stem == "" {
			continue
		}
		b := getBucket(stem)
		b.database[t.Name] = true
	}

	sort.Strings(order)

	var features []model.Feature
	for _, stem := range order {
		b := buckets[stem]
		// Skip stems that only ever matched a single, isolated component —
		// not enough signal to call it a cross-cutting "feature".
		total := len(b.frontend) + len(b.backend) + len(b.apis) + len(b.database)
		if total < 1 {
			continue
		}
		features = append(features, model.Feature{
			Name:     titleCase(stem),
			Frontend: setToSortedSlice(b.frontend),
			Backend:  setToSortedSlice(b.backend),
			APIs:     setToSortedSlice(b.apis),
			Database: setToSortedSlice(b.database),
		})
	}
	return features
}

// titleCase upper-cases the first rune only. Avoids the deprecated
// strings.Title for this simple, ASCII-oriented use case.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func setToSortedSlice(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
