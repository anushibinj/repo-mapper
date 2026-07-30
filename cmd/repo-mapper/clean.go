package main

import (
	"fmt"

	"github.com/anushibinj/repo-mapper/internal/cache"
)

func runClean(args []string) error {
	fs, root := newFlagSet("clean")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := resolveRepoRoot(*root)
	if err != nil {
		return err
	}

	dir := cache.Dir(repoRoot)
	if err := cache.Clean(dir); err != nil {
		return fmt.Errorf("clean: %w", err)
	}
	fmt.Printf("Removed cache at %s\n", dir)
	return nil
}
