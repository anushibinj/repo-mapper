package spring

import (
	"testing"

	"github.com/anushibinj/repo-mapper/internal/plugin"
)

type parsedEntity struct {
	kind string
	name string
	attr map[string]string
}

func parse(t *testing.T, src string) []parsedEntity {
	t.Helper()
	p := New()
	entities, err := p.Parse(plugin.Context{RelPath: "Test.java"}, []byte(src))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	out := make([]parsedEntity, 0, len(entities))
	for _, e := range entities {
		out = append(out, parsedEntity{e.Kind, e.Name, e.Attributes})
	}
	return out
}

// TestControllerRoutes_ClassLevelRequestMappingNotMisdetectedAsRoute is a
// regression test: a class-level @RequestMapping("/auth") must not be
// mistaken for a method-level route mapping (it previously caused routes
// like "/auth/auth" with the wrong HTTP method and handler name).
func TestControllerRoutes_ClassLevelRequestMappingNotMisdetectedAsRoute(t *testing.T) {
	src := `
package com.example;

import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/auth")
public class AuthController {

    @PostMapping("/login")
    public String login(@RequestBody String credentials) {
        return "token";
    }
}
`
	entities := parse(t, src)

	var routes []parsedEntity
	for _, e := range entities {
		if e.kind == "spring-route" {
			routes = append(routes, e)
		}
	}

	if len(routes) != 1 {
		t.Fatalf("expected exactly 1 route, got %d: %+v", len(routes), routes)
	}
	r := routes[0]
	if r.attr["method"] != "POST" {
		t.Errorf("expected method=POST, got %q", r.attr["method"])
	}
	if r.attr["path"] != "/auth/login" {
		t.Errorf("expected path=/auth/login, got %q", r.attr["path"])
	}
	if r.attr["handler"] != "login" {
		t.Errorf("expected handler=login, got %q", r.attr["handler"])
	}
}

func TestControllerRoutes_MultipleMethods(t *testing.T) {
	src := `
package com.example;

import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/billing")
public class BillingController {

    @GetMapping("/invoices")
    public String list() { return ""; }

    @PostMapping("/invoices")
    public String create() { return ""; }

    @GetMapping("/invoices/{id}")
    public String get() { return ""; }
}
`
	entities := parse(t, src)
	count := 0
	paths := map[string]string{}
	for _, e := range entities {
		if e.kind == "spring-route" {
			count++
			paths[e.attr["handler"]] = e.attr["method"] + " " + e.attr["path"]
		}
	}
	if count != 3 {
		t.Fatalf("expected 3 routes, got %d: %+v", count, paths)
	}
	if paths["list"] != "GET /billing/invoices" {
		t.Errorf("list route mismatch: %q", paths["list"])
	}
	if paths["create"] != "POST /billing/invoices" {
		t.Errorf("create route mismatch: %q", paths["create"])
	}
	if paths["get"] != "GET /billing/invoices/{id}" {
		t.Errorf("get route mismatch: %q", paths["get"])
	}
}

func TestServiceStereotype(t *testing.T) {
	src := `
package com.example;

import org.springframework.stereotype.Service;

@Service
public class BillingService {
}
`
	entities := parse(t, src)
	found := false
	for _, e := range entities {
		if e.kind == "spring-service" && e.name == "BillingService" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected spring-service entity for BillingService, got %+v", entities)
	}
}

func TestEntityStereotype_TableAndRelations(t *testing.T) {
	src := `
package com.example;

import javax.persistence.*;

@Entity
@Table(name = "invoices")
public class Invoice {
    @Id
    private Long id;

    private Customer customer;
}
`
	entities := parse(t, src)
	var entity *parsedEntity
	for i := range entities {
		if entities[i].kind == "spring-entity" {
			entity = &entities[i]
		}
	}
	if entity == nil {
		t.Fatalf("expected a spring-entity, got %+v", entities)
	}
	if entity.attr["table"] != "invoices" {
		t.Errorf("expected table=invoices, got %q", entity.attr["table"])
	}
	if entity.attr["relatesTo"] != "Customer" {
		t.Errorf("expected relatesTo=Customer, got %q", entity.attr["relatesTo"])
	}
}

func TestNonSpringJavaFile_ProducesNoEntities(t *testing.T) {
	src := `
package com.example;

public class PlainOldJavaObject {
    private String name;
}
`
	entities := parse(t, src)
	if len(entities) != 0 {
		t.Errorf("expected no spring entities for a plain class, got %+v", entities)
	}
}
