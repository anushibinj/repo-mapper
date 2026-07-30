package sql

import (
	"testing"

	"github.com/anushibinj/repo-mapper/internal/plugin"
)

type invoiceEntity struct {
	attrs map[string]string
	refs  []string
}

func TestParse_CreateTableWithForeignKey(t *testing.T) {
	src := `
CREATE TABLE customers (
    id BIGINT PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE TABLE invoices (
    id BIGINT PRIMARY KEY,
    customer_id BIGINT,
    amount DECIMAL(10, 2),
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);
`
	p := New()
	entities, err := p.Parse(plugin.Context{RelPath: "schema.sql"}, []byte(src))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(entities) != 2 {
		t.Fatalf("expected 2 tables, got %d: %+v", len(entities), entities)
	}

	var invoices *invoiceEntity
	for _, e := range entities {
		if e.Name == "invoices" {
			invoices = &invoiceEntity{e.Attributes, e.Refs}
		}
	}
	if invoices == nil {
		t.Fatalf("expected invoices table entity, got %+v", entities)
	}
	if len(invoices.refs) != 1 || invoices.refs[0] != "customers" {
		t.Errorf("expected refs=[customers], got %v", invoices.refs)
	}
	if invoices.attrs["column:id"] == "" {
		t.Errorf("expected column:id attribute to be present, got %+v", invoices.attrs)
	}
}

func TestCanParse(t *testing.T) {
	p := New()
	if !p.CanParse("db/schema.sql") {
		t.Error("expected CanParse(*.sql) = true")
	}
	if p.CanParse("db/schema.txt") {
		t.Error("expected CanParse(*.txt) = false")
	}
}
