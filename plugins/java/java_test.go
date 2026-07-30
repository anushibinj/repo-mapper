package java

import (
	"testing"

	"github.com/anushibinj/repo-mapper/internal/plugin"
)

func TestParse_ClassWithExtendsAndImplements(t *testing.T) {
	src := `
package com.example;

import java.util.List;
import java.io.Serializable;

public class Invoice extends BaseEntity implements Serializable, Comparable<Invoice> {
}
`
	p := New()
	entities, err := p.Parse(plugin.Context{RelPath: "Invoice.java"}, []byte(src))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d: %+v", len(entities), entities)
	}
	e := entities[0]
	if e.Name != "Invoice" {
		t.Errorf("expected Name=Invoice, got %q", e.Name)
	}
	if e.Package != "com.example" {
		t.Errorf("expected Package=com.example, got %q", e.Package)
	}
	if e.Attributes["declKind"] != "class" {
		t.Errorf("expected declKind=class, got %q", e.Attributes["declKind"])
	}
	if e.Attributes["extends"] != "BaseEntity" {
		t.Errorf("expected extends=BaseEntity, got %q", e.Attributes["extends"])
	}
	if e.Attributes["implements"] != "Serializable, Comparable<Invoice>" {
		t.Errorf("expected implements list, got %q", e.Attributes["implements"])
	}
}

func TestParse_Interface(t *testing.T) {
	src := `
package com.example;

public interface Repository {
}
`
	p := New()
	entities, err := p.Parse(plugin.Context{RelPath: "Repository.java"}, []byte(src))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(entities) != 1 || entities[0].Attributes["declKind"] != "interface" {
		t.Fatalf("expected 1 interface entity, got %+v", entities)
	}
}

func TestCanParse(t *testing.T) {
	p := New()
	if !p.CanParse("src/Main.java") {
		t.Error("expected CanParse(*.java) = true")
	}
	if p.CanParse("src/Main.ts") {
		t.Error("expected CanParse(*.ts) = false")
	}
}
