package main

import (
	"fmt"

	"github.com/anushibinj/repo-mapper/internal/generator"
	"github.com/anushibinj/repo-mapper/internal/pipeline"
)

// runGraph prints one Mermaid architecture diagram to stdout without
// writing any files (PRD section 6, "graph").
func runGraph(args []string) error {
	fs, root := newFlagSet("graph")
	diagram := fs.String("diagram", "system", "Diagram to print: system|backend|frontend|database|auth")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, cfg, log, err := loadContext(*root)
	if err != nil {
		return err
	}

	result, err := pipeline.FullScan(repoRoot, cfg, log)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	var out string
	switch *diagram {
	case "system":
		out = generator.SystemDiagram(result.Repository)
	case "backend":
		out = generator.BackendDiagram(result.Repository)
	case "frontend":
		out = generator.FrontendDiagram(result.Repository)
	case "database":
		out = generator.DatabaseDiagram(result.Repository)
	case "auth":
		var ok bool
		out, ok = generator.AuthDiagram(result.Repository)
		if !ok {
			return fmt.Errorf("no authentication-related components detected")
		}
	default:
		return fmt.Errorf("unknown diagram %q (expected system|backend|frontend|database|auth)", *diagram)
	}

	fmt.Println(out)
	return nil
}
