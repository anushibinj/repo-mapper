package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/anushibinj/repo-mapper/internal/config"
	"github.com/anushibinj/repo-mapper/internal/plugin"
)

// runDoctor validates the local installation and environment: git
// availability, repo root accessibility, config loading, output directory
// writability, and registered plugins (PRD section 6, "doctor").
func runDoctor(args []string) error {
	fs, root := newFlagSet("doctor")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ok := true
	check := func(label string, pass bool, detail string) {
		status := "OK"
		if !pass {
			status = "FAIL"
			ok = false
		}
		fmt.Printf("[%s] %-28s %s\n", status, label, detail)
	}

	repoRoot, err := resolveRepoRoot(*root)
	if err != nil {
		check("repository root", false, err.Error())
	} else if info, statErr := os.Stat(repoRoot); statErr != nil || !info.IsDir() {
		check("repository root", false, "not found: "+repoRoot)
	} else {
		check("repository root", true, repoRoot)
	}

	if _, err := exec.LookPath("git"); err != nil {
		check("git binary", false, "git not found on PATH (required for `update`)")
	} else {
		check("git binary", true, "found on PATH")
	}

	var cfg *config.Config
	if repoRoot != "" {
		cfg, err = config.Load(repoRoot)
		if err != nil {
			check("configuration", false, err.Error())
		} else {
			configPath := filepath.Join(repoRoot, config.DefaultFileName)
			if _, statErr := os.Stat(configPath); statErr == nil {
				check("configuration", true, configPath)
			} else {
				check("configuration", true, "using defaults (no "+config.DefaultFileName+" found)")
			}
		}
	}

	if cfg != nil {
		outDir := filepath.Join(repoRoot, cfg.Output.Directory)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			check("output directory writable", false, err.Error())
		} else {
			probe := filepath.Join(outDir, ".repo-mapper-write-test")
			if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
				check("output directory writable", false, err.Error())
			} else {
				os.Remove(probe)
				check("output directory writable", true, outDir)
			}
		}
	}

	names := plugin.Names()
	check("plugins registered", len(names) > 0, fmt.Sprintf("%d plugins", len(names)))

	if !ok {
		return fmt.Errorf("doctor found problems")
	}
	fmt.Println("\nAll checks passed.")
	return nil
}
