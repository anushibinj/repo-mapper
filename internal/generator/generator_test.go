package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/anushibinj/repo-mapper/internal/model"
)

func sampleRepo() *model.Repository {
	return &model.Repository{
		Name:      "sample",
		RootPath:  "/repo",
		Languages: []model.Language{{Name: "Java", FileCount: 2}, {Name: "TypeScript/JavaScript", FileCount: 1}},
		Components: []model.Component{
			{ID: "BillingController.java#BillingController", Name: "BillingController", Kind: "controller", Language: "Java", File: "BillingController.java", DependsOn: []string{"billingService"}},
			{ID: "BillingService.java#BillingService", Name: "BillingService", Kind: "service", Language: "Java", File: "BillingService.java"},
			{ID: "Invoice.java#Invoice", Name: "Invoice", Kind: "entity", Language: "Java", File: "Invoice.java"},
			{ID: "BillingPage.tsx#BillingPage", Name: "BillingPage", Kind: "page", Language: "TypeScript/JavaScript", File: "BillingPage.tsx", Calls: []string{"/billing/invoices"}},
		},
		APIs: []model.API{
			{Method: "GET", Path: "/billing/invoices", Controller: "BillingController", Handler: "listInvoices", File: "BillingController.java"},
		},
		Tables: []model.Table{
			{Name: "invoices", Entity: "Invoice", SourceFile: "schema.sql", Columns: []model.Column{{Name: "id", Type: "BIGINT", PrimaryKey: true}}},
		},
		Features: []model.Feature{
			{Name: "Billing", Backend: []string{"BillingController", "BillingService"}, Frontend: []string{"BillingPage"}, APIs: []string{"/billing/invoices"}, Database: []string{"invoices"}},
		},
		Git: model.GitInfo{Branch: "main", CommitHash: "abc123"},
	}
}

func TestWriteAll_CreatesAllExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	repo := sampleRepo()

	if err := WriteAll(repo, dir); err != nil {
		t.Fatalf("WriteAll failed: %v", err)
	}

	expected := []string{
		"README.md", "backend.md", "frontend.md", "architecture.md", "database.md", "features.md",
		"repo-map.json", "feature-map.json", "api-map.json", "ownership-map.json", "routes.json",
		filepath.Join("diagrams", "system.mmd"),
		filepath.Join("diagrams", "backend.mmd"),
		filepath.Join("diagrams", "frontend.mmd"),
		filepath.Join("diagrams", "database.mmd"),
	}
	for _, name := range expected {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected file %s to exist: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("expected file %s to be non-empty", name)
		}
	}
}

func TestWriteJSON_RepoMapRoundTrips(t *testing.T) {
	dir := t.TempDir()
	repo := sampleRepo()

	if err := WriteJSON(repo, dir); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "repo-map.json"))
	if err != nil {
		t.Fatalf("failed to read repo-map.json: %v", err)
	}
	var got model.Repository
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal repo-map.json: %v", err)
	}
	if got.Name != repo.Name || len(got.Components) != len(repo.Components) {
		t.Errorf("repo-map.json round-trip mismatch: %+v", got)
	}
}

func TestWriteAll_Deterministic(t *testing.T) {
	repo := sampleRepo()
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	if err := WriteAll(repo, dir1); err != nil {
		t.Fatalf("WriteAll (1) failed: %v", err)
	}
	if err := WriteAll(repo, dir2); err != nil {
		t.Fatalf("WriteAll (2) failed: %v", err)
	}

	names := []string{"README.md", "repo-map.json", filepath.Join("diagrams", "system.mmd"), filepath.Join("diagrams", "backend.mmd")}
	for _, name := range names {
		a, err := os.ReadFile(filepath.Join(dir1, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		b, err := os.ReadFile(filepath.Join(dir2, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(a) != string(b) {
			t.Errorf("expected %s to be byte-identical across runs", name)
		}
	}
}

func TestPluralize(t *testing.T) {
	cases := map[string]string{
		"repository": "Repositories",
		"entity":     "Entities",
		"controller": "Controllers",
		"service":    "Services",
		"bean":       "Beans",
		"key":        "Keys", // 'e' precedes 'y', so simple +s, not "-ies"
	}
	for kind, want := range cases {
		if got := pluralize(kind); got != want {
			t.Errorf("pluralize(%q) = %q, want %q", kind, got, want)
		}
	}
}
