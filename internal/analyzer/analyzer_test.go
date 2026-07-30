package analyzer

import (
	"testing"

	"github.com/anushibinj/repo-mapper/internal/model"
)

func findComponent(components []model.Component, name string) *model.Component {
	for i := range components {
		if components[i].Name == name {
			return &components[i]
		}
	}
	return nil
}

// TestBuild_SpringComponentDoesNotCollideWithReactComponent is a regression
// test: the generic Spring @Component stereotype must not share the Kind
// "component" with frontend React components, or backend.md/frontend.md
// and diagram filters mix the two up.
func TestBuild_SpringComponentDoesNotCollideWithReactComponent(t *testing.T) {
	entities := []model.Entity{
		{Kind: "spring-component", Name: "CacheWarmer", File: "CacheWarmer.java", Language: "Java"},
		{Kind: "react-component", Name: "InvoiceList", File: "InvoiceList.tsx", Language: "TypeScript/JavaScript"},
	}
	repo := Build("test", "/repo", nil, entities, model.GitInfo{})

	backendComp := findComponent(repo.Components, "CacheWarmer")
	frontendComp := findComponent(repo.Components, "InvoiceList")
	if backendComp == nil || frontendComp == nil {
		t.Fatalf("expected both components present, got %+v", repo.Components)
	}
	if backendComp.Kind == frontendComp.Kind {
		t.Errorf("expected different Kinds, both got %q", backendComp.Kind)
	}
	if backendComp.Kind != "bean" {
		t.Errorf("expected Spring @Component to have Kind=bean, got %q", backendComp.Kind)
	}
	if frontendComp.Kind != "component" {
		t.Errorf("expected React component to have Kind=component, got %q", frontendComp.Kind)
	}
}

// TestBuild_SpringStereotypeOverridesGenericJavaType is a regression test:
// a @RestController-annotated class must show up as Kind "controller", not
// the generic "class" kind the plain Java plugin also emits for it, and its
// DependsOn must come only from @Autowired fields, not from unrelated
// import names.
func TestBuild_SpringStereotypeOverridesGenericJavaType(t *testing.T) {
	entities := []model.Entity{
		{
			Kind: "java-type", Name: "BillingController", File: "BillingController.java", Language: "Java",
			Attributes: map[string]string{"declKind": "class"},
			Refs:       []string{"org.springframework.web.bind.annotation.RestController", "java.util.List"},
		},
		{
			Kind: "spring-restcontroller", Name: "BillingController", File: "BillingController.java", Language: "Java",
			Attributes: map[string]string{"basePath": "/billing"},
			Refs:       []string{"billingService"},
		},
	}
	repo := Build("test", "/repo", nil, entities, model.GitInfo{})

	c := findComponent(repo.Components, "BillingController")
	if c == nil {
		t.Fatalf("expected BillingController component")
	}
	if c.Kind != "controller" {
		t.Errorf("expected Kind=controller, got %q", c.Kind)
	}
	if len(c.DependsOn) != 1 || c.DependsOn[0] != "billingService" {
		t.Errorf("expected DependsOn=[billingService] (no import noise), got %v", c.DependsOn)
	}
}

func TestBuild_APIsAndTablesFromEntities(t *testing.T) {
	entities := []model.Entity{
		{
			Kind: "spring-route", Name: "listInvoices", File: "BillingController.java", Language: "Java",
			Attributes: map[string]string{"method": "GET", "path": "/billing/invoices", "controller": "BillingController", "handler": "listInvoices"},
		},
		{
			Kind: "sql-table", Name: "invoices", File: "schema.sql", Language: "SQL",
			Attributes: map[string]string{"column:id": "BIGINT:1:", "column:amount": "DECIMAL:0:"},
		},
	}
	repo := Build("test", "/repo", nil, entities, model.GitInfo{})

	if len(repo.APIs) != 1 || repo.APIs[0].Path != "/billing/invoices" {
		t.Errorf("expected 1 API with path /billing/invoices, got %+v", repo.APIs)
	}
	if len(repo.Tables) != 1 || repo.Tables[0].Name != "invoices" {
		t.Fatalf("expected 1 table named invoices, got %+v", repo.Tables)
	}
	if len(repo.Tables[0].Columns) != 2 {
		t.Errorf("expected 2 columns, got %+v", repo.Tables[0].Columns)
	}
}

func TestBuild_FeatureGroupingByNamingStem(t *testing.T) {
	entities := []model.Entity{
		{Kind: "spring-restcontroller", Name: "BillingController", File: "BillingController.java", Language: "Java"},
		{Kind: "spring-service", Name: "BillingService", File: "BillingService.java", Language: "Java"},
		{
			Kind: "spring-route", Name: "listInvoices", File: "BillingController.java", Language: "Java",
			Attributes: map[string]string{"method": "GET", "path": "/billing/invoices", "controller": "BillingController", "handler": "listInvoices"},
		},
		{Kind: "react-component", Name: "BillingPage", File: "BillingPage.tsx", Language: "TypeScript/JavaScript"},
	}
	repo := Build("test", "/repo", nil, entities, model.GitInfo{})

	var billing *model.Feature
	for i := range repo.Features {
		if repo.Features[i].Name == "Billing" {
			billing = &repo.Features[i]
		}
	}
	if billing == nil {
		t.Fatalf("expected a Billing feature, got %+v", repo.Features)
	}
	if len(billing.Backend) != 2 {
		t.Errorf("expected 2 backend members, got %v", billing.Backend)
	}
	if len(billing.Frontend) != 1 || billing.Frontend[0] != "BillingPage" {
		t.Errorf("expected frontend=[BillingPage], got %v", billing.Frontend)
	}
	if len(billing.APIs) != 1 || billing.APIs[0] != "/billing/invoices" {
		t.Errorf("expected apis=[/billing/invoices], got %v", billing.APIs)
	}
}

func TestBuild_EntityRelatesToDedupedAgainstTableName(t *testing.T) {
	entities := []model.Entity{
		{
			Kind: "spring-entity", Name: "Invoice", File: "Invoice.java", Language: "Java",
			Attributes: map[string]string{"table": "invoices", "relatesTo": "Customer"},
		},
		{
			Kind: "spring-entity", Name: "Customer", File: "Customer.java", Language: "Java",
			Attributes: map[string]string{"table": "customers"},
		},
		{
			Kind: "sql-table", Name: "invoices", File: "schema.sql", Language: "SQL",
			Refs: []string{"customers"},
		},
	}
	repo := Build("test", "/repo", nil, entities, model.GitInfo{})

	var invoices *model.Table
	for i := range repo.Tables {
		if repo.Tables[i].Name == "invoices" {
			invoices = &repo.Tables[i]
		}
	}
	if invoices == nil {
		t.Fatalf("expected invoices table, got %+v", repo.Tables)
	}
	if len(invoices.RelatesTo) != 1 || invoices.RelatesTo[0] != "customers" {
		t.Errorf("expected deduped RelatesTo=[customers], got %v", invoices.RelatesTo)
	}
}

// TestBuild_Deterministic verifies that building the same entity set twice
// produces byte-identical output ordering (Copilot Coding Guideline #9).
func TestBuild_Deterministic(t *testing.T) {
	entities := []model.Entity{
		{Kind: "spring-restcontroller", Name: "ZController", File: "Z.java", Language: "Java"},
		{Kind: "spring-restcontroller", Name: "AController", File: "A.java", Language: "Java"},
	}
	r1 := Build("test", "/repo", nil, entities, model.GitInfo{})
	r2 := Build("test", "/repo", nil, entities, model.GitInfo{})

	if len(r1.Components) != len(r2.Components) {
		t.Fatalf("component count mismatch")
	}
	for i := range r1.Components {
		if r1.Components[i].ID != r2.Components[i].ID {
			t.Errorf("component order mismatch at %d: %q vs %q", i, r1.Components[i].ID, r2.Components[i].ID)
		}
	}
	if r1.Components[0].Name != "AController" {
		t.Errorf("expected sorted order (AController first), got %q", r1.Components[0].Name)
	}
}
