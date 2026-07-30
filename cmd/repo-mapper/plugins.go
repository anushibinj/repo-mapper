package main

import (
	"fmt"

	"github.com/anushibinj/repo-mapper/internal/config"
	"github.com/anushibinj/repo-mapper/internal/plugin"
)

// runPlugins lists every registered plugin and whether it's enabled per the
// resolved configuration (PRD section 6, "plugins").
func runPlugins(args []string) error {
	fs, root := newFlagSet("plugins")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := resolveRepoRoot(*root)
	if err != nil {
		return err
	}
	cfg, err := config.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	names := plugin.Names()
	if len(names) == 0 {
		fmt.Println("No plugins registered.")
		return nil
	}

	fmt.Printf("%-12s %s\n", "PLUGIN", "STATUS")
	for _, name := range names {
		status := "enabled"
		if !cfg.PluginEnabled(name) {
			status = "disabled"
		}
		fmt.Printf("%-12s %s\n", name, status)
	}
	return nil
}
