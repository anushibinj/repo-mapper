// Package spring is a heuristic plugin layered on top of Java source,
// recognising common Spring Framework / Spring Boot annotations to surface
// controllers, services, repositories, entities, and their HTTP routes and
// dependencies (PRD section 12, "Spring: @RestController -> Endpoints").
package spring

import (
	"regexp"
	"strings"

	"github.com/anushibinj/repo-mapper/internal/model"
	"github.com/anushibinj/repo-mapper/internal/plugin"
)

func init() {
	plugin.Register(New())
}

// Plugin is the Spring framework plugin. It only handles .java files that
// contain at least one recognised Spring annotation.
type Plugin struct{}

// New constructs a Spring Plugin.
func New() *Plugin { return &Plugin{} }

// Name implements plugin.Plugin.
func (p *Plugin) Name() string { return "spring" }

// CanParse implements plugin.Plugin. Spring re-parses the same .java files
// the java plugin handles; the parser dispatch layer runs every matching
// plugin per file, so both sets of entities are produced independently.
func (p *Plugin) CanParse(file string) bool {
	return strings.HasSuffix(file, ".java")
}

var (
	classAnnotatedRe = regexp.MustCompile(
		`@(RestController|Controller|Service|Repository|Component|Entity)\b(?:\(([^)]*)\))?[^;{]*?\b(?:class|interface)\s+(\w+)`)
	classRequestMappingRe = regexp.MustCompile(`@RequestMapping\(([^)]*)\)\s*(?:@\w+(?:\([^)]*\))?\s*)*(?:public\s+)?(?:class|interface)\s+(\w+)`)
	methodMappingRe       = regexp.MustCompile(`@(GetMapping|PostMapping|PutMapping|DeleteMapping|PatchMapping|RequestMapping)\(([^)]*)\)[\s\S]{0,200}?\b(\w+)\s*\(`)
	autowiredFieldRe      = regexp.MustCompile(`@Autowired[\s\S]{0,80}?\b(?:private|public|protected|final|\s)*([\w.]+(?:<[\w.,\s]+>)?)\s+(\w+)\s*;`)
	// Constructor-injection heuristic: a field declaration of a type ending
	// in Repository/Service assigned via constructor parameter of same name.
	fieldDeclRe      = regexp.MustCompile(`(?m)^\s*(?:private|public|protected|final|\s)+([\w.]+(?:<[\w.,\s]+>)?)\s+(\w+)\s*;`)
	valueAttrRe      = regexp.MustCompile(`(?:value\s*=\s*)?"([^"]*)"`)
	nameAttrRe       = regexp.MustCompile(`name\s*=\s*"([^"]*)"`)
	tableAnnoRe      = regexp.MustCompile(`@Table\(([^)]*)\)`)
	columnRe         = regexp.MustCompile(`@Column\(([^)]*)\)\s*(?:private|public|protected|\s)*([\w.]+(?:<[\w.,\s]+>)?)\s+(\w+)\s*;`)
	idFieldRe        = regexp.MustCompile(`@Id\s*(?:@\w+(?:\([^)]*\))?\s*)*(?:private|public|protected|\s)*([\w.]+)\s+(\w+)\s*;`)
	classBodyStartRe = regexp.MustCompile(`(?:class|interface)\s+\w+[^{]*\{`)
)

// Parse implements plugin.Plugin.
func (p *Plugin) Parse(ctx plugin.Context, content []byte) ([]model.Entity, error) {
	src := string(content)
	if !strings.Contains(src, "org.springframework") && !strings.ContainsAny(src, "@") {
		return nil, nil
	}

	var entities []model.Entity

	stereotype, className, classArgs := detectStereotype(src)
	if stereotype == "" {
		return nil, nil
	}

	attrs := map[string]string{
		"stereotype": stereotype,
	}

	var deps []string
	for _, m := range autowiredFieldRe.FindAllStringSubmatch(src, -1) {
		deps = append(deps, m[2]) // field name; analyzer resolves to component by type/name heuristics
		attrs["dep:"+m[2]] = m[1]
	}

	switch stereotype {
	case "RestController", "Controller":
		basePath := ""
		if classArgs != "" {
			basePath = firstMatch(valueAttrRe, classArgs)
		} else if m := classRequestMappingRe.FindStringSubmatch(src); m != nil && m[2] == className {
			basePath = firstMatch(valueAttrRe, m[1])
		}
		attrs["basePath"] = basePath

		// Search for method-level mapping annotations only within the class
		// body — searching the whole file would let a class-level
		// @RequestMapping("...") (used for the base path above) get
		// mis-matched as a method mapping, since RequestMapping is valid at
		// both levels.
		classBody := src
		if idx := classBodyStartRe.FindStringIndex(src); idx != nil {
			classBody = src[idx[1]:]
		}

		for _, m := range methodMappingRe.FindAllStringSubmatch(classBody, -1) {
			httpMethod := mappingToHTTPMethod(m[1])
			path := firstMatch(valueAttrRe, m[2])
			handler := m[3]
			fullPath := joinPath(basePath, path)
			entities = append(entities, model.Entity{
				Kind:     "spring-route",
				Name:     handler,
				File:     ctx.RelPath,
				Language: "Java",
				Attributes: map[string]string{
					"method":     httpMethod,
					"path":       fullPath,
					"controller": className,
					"handler":    handler,
				},
			})
		}

	case "Entity":
		tableName := className
		if m := tableAnnoRe.FindStringSubmatch(src); m != nil {
			if n := firstMatch(nameAttrRe, m[1]); n != "" {
				tableName = n
			}
		}
		attrs["table"] = tableName

		var relatesTo []string
		for _, m := range fieldDeclRe.FindAllStringSubmatch(src, -1) {
			fieldType := m[1]
			if looksLikeEntityRef(fieldType) {
				relatesTo = append(relatesTo, baseTypeName(fieldType))
			}
		}
		attrs["relatesTo"] = strings.Join(relatesTo, ",")
	}

	entities = append(entities, model.Entity{
		Kind:       "spring-" + strings.ToLower(stereotype),
		Name:       className,
		File:       ctx.RelPath,
		Language:   "Java",
		Attributes: attrs,
		Refs:       deps,
	})

	return entities, nil
}

func detectStereotype(src string) (stereotype, className, classArgs string) {
	m := classAnnotatedRe.FindStringSubmatch(src)
	if m == nil {
		return "", "", ""
	}
	return m[1], m[3], m[2]
}

func mappingToHTTPMethod(annotation string) string {
	switch annotation {
	case "GetMapping":
		return "GET"
	case "PostMapping":
		return "POST"
	case "PutMapping":
		return "PUT"
	case "DeleteMapping":
		return "DELETE"
	case "PatchMapping":
		return "PATCH"
	default:
		return "GET"
	}
}

func firstMatch(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

func joinPath(base, sub string) string {
	base = strings.TrimSuffix(base, "/")
	if sub == "" {
		if base == "" {
			return "/"
		}
		return base
	}
	if !strings.HasPrefix(sub, "/") {
		sub = "/" + sub
	}
	return base + sub
}

func looksLikeEntityRef(typeName string) bool {
	t := baseTypeName(typeName)
	return len(t) > 0 && t[0] >= 'A' && t[0] <= 'Z' &&
		!strings.HasPrefix(typeName, "String") &&
		!strings.Contains(typeName, "Long") &&
		!strings.Contains(typeName, "Integer") &&
		!strings.Contains(typeName, "Boolean") &&
		!strings.Contains(typeName, "Double") &&
		!strings.Contains(typeName, "BigDecimal") &&
		!strings.Contains(typeName, "Date") &&
		!strings.Contains(typeName, "Instant") &&
		!strings.Contains(typeName, "LocalDate")
}

func baseTypeName(typeName string) string {
	// Unwrap generics like List<Order> -> Order; leaves plain names as-is.
	if idx := strings.Index(typeName, "<"); idx >= 0 {
		inner := strings.TrimSuffix(typeName[idx+1:], ">")
		inner = strings.TrimSpace(strings.TrimSuffix(inner, ">"))
		return inner
	}
	return typeName
}
