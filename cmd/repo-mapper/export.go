package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/anushibinj/repo-mapper/internal/pipeline"
)

// runExport prints (or writes) the canonical JSON model (PRD section 6,
// "export"). Defaults to stdout so it composes with other tools.
func runExport(args []string) error {
	fs, root := newFlagSet("export")
	out := fs.String("out", "", "Write to this file instead of stdout")
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

	data, err := json.MarshalIndent(result.Repository, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal model: %w", err)
	}
	data = append(data, '\n')

	if *out == "" {
		_, err = os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(*out, data, 0o644)
}
