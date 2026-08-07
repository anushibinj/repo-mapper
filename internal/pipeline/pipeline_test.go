package pipeline_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anushibinj/repo-mapper/internal/config"
	"github.com/anushibinj/repo-mapper/internal/logger"
	"github.com/anushibinj/repo-mapper/internal/pipeline"

	// Blank-import every Phase 1 plugin so their init() functions register
	// them with the static plugin.Registry, exactly as cmd/repo-mapper does.
	_ "github.com/anushibinj/repo-mapper/plugins/docker"
	_ "github.com/anushibinj/repo-mapper/plugins/java"
	_ "github.com/anushibinj/repo-mapper/plugins/node"
	_ "github.com/anushibinj/repo-mapper/plugins/react"
	_ "github.com/anushibinj/repo-mapper/plugins/spring"
	_ "github.com/anushibinj/repo-mapper/plugins/sql"
	_ "github.com/anushibinj/repo-mapper/plugins/vite"
)

// TestFullScan_BillingAppExample is a regression-guarding integration test:
// it runs the whole scanner -> cache -> parser -> analyzer pipeline against
// the real example fixture (examples/billing-app) and asserts on the
// counts/shapes that manual verification confirmed were correct after
// fixing Bugs 1-9 (see project history). Any future change that breaks
// this pipeline end-to-end should fail here before it reaches a real user.
func TestFullScan_BillingAppExample(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "examples", "billing-app"))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	cfg := config.Default()
	log := logger.Nop()

	result, err := pipeline.FullScan(repoRoot, cfg, log)
	if err != nil {
		t.Fatalf("FullScan failed: %v", err)
	}
	repo := result.Repository

	kindCounts := map[string]int{}
	for _, c := range repo.Components {
		kindCounts[c.Kind]++
	}

	wantKindCounts := map[string]int{
		"controller":    2, // BillingController, AuthController
		"service":       2, // BillingService, AuthService
		"repository":    1, // BillingRepository
		"entity":        2, // Invoice, Customer
		"page":          1, // BillingPage
		"component":     1, // InvoiceList
		"hook":          1, // useBilling
		"infra-service": 3, // docker-compose backend/frontend/db
	}
	for kind, want := range wantKindCounts {
		if got := kindCounts[kind]; got != want {
			t.Errorf("component kind %q: got %d, want %d (all kinds: %+v)", kind, got, want, kindCounts)
		}
	}

	// Regression guard for Bug 3: Spring's generic @Component stereotype
	// must never share Kind "component" with React components.
	if kindCounts["bean"] > 0 {
		for _, c := range repo.Components {
			if c.Kind == "bean" && c.Language != "Java" {
				t.Errorf("expected only Java components to have Kind=bean, got %+v", c)
			}
		}
	}

	// Regression guard for Bug 1: the class-level @RequestMapping must not
	// produce a spurious extra route, and paths/methods must be correct.
	if len(repo.APIs) != 4 {
		t.Fatalf("expected 4 APIs (3 billing + 1 auth), got %d: %+v", len(repo.APIs), repo.APIs)
	}
	apiByHandler := map[string]string{}
	for _, a := range repo.APIs {
		apiByHandler[a.Handler] = a.Method + " " + a.Path
	}
	wantAPIs := map[string]string{
		"listInvoices":  "GET /billing/invoices",
		"getInvoice":    "GET /billing/invoices/{id}",
		"createInvoice": "POST /billing/invoices",
		"login":         "POST /auth/login",
	}
	for handler, want := range wantAPIs {
		if got := apiByHandler[handler]; got != want {
			t.Errorf("API for handler %q: got %q, want %q", handler, got, want)
		}
	}

	// Regression guard for Bug 8: entity-class-name relations should be
	// normalized to SQL table names, not left as duplicate spellings.
	if len(repo.Tables) != 2 {
		t.Fatalf("expected 2 tables (customers, invoices), got %d: %+v", len(repo.Tables), repo.Tables)
	}
	var invoicesTable *struct {
		relatesTo []string
	}
	for _, tbl := range repo.Tables {
		if tbl.Name == "invoices" {
			invoicesTable = &struct{ relatesTo []string }{tbl.RelatesTo}
		}
	}
	if invoicesTable == nil {
		t.Fatalf("expected invoices table, got %+v", repo.Tables)
	}
	if len(invoicesTable.relatesTo) != 1 || invoicesTable.relatesTo[0] != "customers" {
		t.Errorf("expected invoices.RelatesTo == [customers] (deduped), got %v", invoicesTable.relatesTo)
	}

	// Regression guard for Bugs 5 & 6: useBilling must be detected as a
	// react-hook (not missed due to filename heuristics) and must carry
	// its own API calls.
	var hook *struct {
		calls []string
	}
	for _, c := range repo.Components {
		if c.Kind == "hook" && c.Name == "useBilling" {
			hook = &struct{ calls []string }{c.Calls}
		}
	}
	if hook == nil {
		t.Fatalf("expected a useBilling hook component, got %+v", repo.Components)
	}
	if len(hook.calls) == 0 {
		t.Errorf("expected useBilling to have captured API calls, got none")
	}

	// Sanity: a Billing feature should have been formed spanning all layers.
	var billingFeature *struct {
		frontendLen int
		backendLen  int
		apisLen     int
	}
	for _, f := range repo.Features {
		if f.Name == "Billing" {
			billingFeature = &struct {
				frontendLen int
				backendLen  int
				apisLen     int
			}{len(f.Frontend), len(f.Backend), len(f.APIs)}
		}
	}
	if billingFeature == nil {
		t.Fatalf("expected a Billing feature, got %+v", repo.Features)
	}
	if billingFeature.frontendLen == 0 || billingFeature.backendLen == 0 || billingFeature.apisLen == 0 {
		t.Errorf("expected Billing feature to span frontend/backend/apis, got %+v", billingFeature)
	}
}

// TestFullScan_AutomaticallyGitignoresCacheDirectory verifies that running
// a scan on a fresh repository (with no .gitignore yet) automatically
// creates one covering the cache directory, so nobody has to remember to
// exclude .cache/ from version control by hand.
func TestFullScan_AutomaticallyGitignoresCacheDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	if _, err := pipeline.FullScan(dir, cfg, logger.Nop()); err != nil {
		t.Fatalf("FullScan failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore to be created automatically: %v", err)
	}
	if !strings.Contains(string(data), ".cache/") {
		t.Errorf("expected auto-created .gitignore to cover .cache/, got:\n%s", data)
	}
}
