// Package sql parses SQL DDL files (CREATE TABLE statements) into Table
// entities with column and foreign-key relationship information.
package sql

import (
	"regexp"
	"strings"

	"github.com/anushibinj/repo-mapper/internal/model"
	"github.com/anushibinj/repo-mapper/internal/plugin"
)

func init() {
	plugin.Register(New())
}

// Plugin is the SQL DDL plugin.
type Plugin struct{}

// New constructs a SQL Plugin.
func New() *Plugin { return &Plugin{} }

// Name implements plugin.Plugin.
func (p *Plugin) Name() string { return "sql" }

// CanParse implements plugin.Plugin.
func (p *Plugin) CanParse(file string) bool {
	return strings.HasSuffix(strings.ToLower(file), ".sql")
}

var (
	createTableRe = regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?[` + "`\"" + `\[]?([\w.]+)[` + "`\"" + `\]]?\s*\(`)
	columnLineRe  = regexp.MustCompile(`(?i)^\s*[` + "`\"" + `\[]?(\w+)[` + "`\"" + `\]]?\s+([\w]+(?:\([^)]*\))?)`)
	primaryKeyRe  = regexp.MustCompile(`(?i)PRIMARY\s+KEY`)
	foreignKeyRe  = regexp.MustCompile(`(?i)REFERENCES\s+[` + "`\"" + `\[]?([\w.]+)[` + "`\"" + `\]]?`)
)

// Parse implements plugin.Plugin.
func (p *Plugin) Parse(ctx plugin.Context, content []byte) ([]model.Entity, error) {
	src := string(content)
	var entities []model.Entity

	locs := createTableRe.FindAllStringSubmatchIndex(src, -1)
	for _, loc := range locs {
		tableName := src[loc[2]:loc[3]]
		bodyStart := loc[1] // right after the opening "("
		body := extractBalancedParens(src, bodyStart-1)

		var columns []model.Column
		var relatesTo []string
		for _, line := range splitTopLevelCommas(body) {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			upper := strings.ToUpper(line)
			if strings.HasPrefix(upper, "CONSTRAINT") || strings.HasPrefix(upper, "PRIMARY KEY") ||
				strings.HasPrefix(upper, "FOREIGN KEY") || strings.HasPrefix(upper, "UNIQUE") ||
				strings.HasPrefix(upper, "CHECK") || strings.HasPrefix(upper, "INDEX") ||
				strings.HasPrefix(upper, "KEY ") {
				if fk := foreignKeyRe.FindStringSubmatch(line); fk != nil {
					relatesTo = append(relatesTo, fk[1])
				}
				continue
			}

			m := columnLineRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			col := model.Column{
				Name:       m[1],
				Type:       m[2],
				PrimaryKey: primaryKeyRe.MatchString(line),
			}
			if fk := foreignKeyRe.FindStringSubmatch(line); fk != nil {
				col.ForeignKey = fk[1]
				relatesTo = append(relatesTo, fk[1])
			}
			columns = append(columns, col)
		}

		attrs := map[string]string{
			"columnCount": itoa(len(columns)),
		}
		for _, col := range columns {
			// Encoded as "name:type:pk:fk" so the analyzer can reconstruct
			// model.Column without needing a richer Entity schema.
			pk := "0"
			if col.PrimaryKey {
				pk = "1"
			}
			attrs["column:"+col.Name] = col.Type + ":" + pk + ":" + col.ForeignKey
		}

		entities = append(entities, model.Entity{
			Kind:       "sql-table",
			Name:       tableName,
			File:       ctx.RelPath,
			Language:   "SQL",
			Attributes: attrs,
			Refs:       relatesTo,
		})
	}

	return entities, nil
}

// extractBalancedParens returns the content between the matching pair of
// parentheses starting at openIdx (which must point at '(').
func extractBalancedParens(s string, openIdx int) string {
	depth := 0
	for i := openIdx; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[openIdx+1 : i]
			}
		}
	}
	return s[openIdx+1:]
}

// splitTopLevelCommas splits a column definition body on commas that are
// not nested inside parentheses (e.g. VARCHAR(255)).
func splitTopLevelCommas(s string) []string {
	var parts []string
	depth := 0
	last := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[last:i])
				last = i + 1
			}
		}
	}
	parts = append(parts, s[last:])
	return parts
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
