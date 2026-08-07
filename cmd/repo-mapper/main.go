// Command repo-mapper is the Repo Mapper CLI entrypoint. See PRD section 6
// for the full command reference.
package main

import (
	"errors"
	"fmt"
	"os"

	// Blank-imported so each plugin's init() registers itself into the
	// static plugin.Registry (see internal/plugin's "Plugin Loading
	// Strategy" doc comment for why this is compiled-in, not dynamic).
	_ "github.com/anushibinj/repo-mapper/plugins/docker"
	_ "github.com/anushibinj/repo-mapper/plugins/java"
	_ "github.com/anushibinj/repo-mapper/plugins/node"
	_ "github.com/anushibinj/repo-mapper/plugins/react"
	_ "github.com/anushibinj/repo-mapper/plugins/spring"
	_ "github.com/anushibinj/repo-mapper/plugins/sql"
	_ "github.com/anushibinj/repo-mapper/plugins/vite"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "scan":
		err = runScan(args)
	case "update":
		err = runUpdate(args)
	case "clean":
		err = runClean(args)
	case "doctor":
		err = runDoctor(args)
	case "plugins":
		err = runPlugins(args)
	case "graph":
		err = runGraph(args)
	case "export":
		err = runExport(args)
	case "-h", "--help", "help":
		printUsage()
		return
	case "-v", "--version", "version":
		fmt.Println("repo-mapper " + version)
		return
	default:
		fmt.Fprintf(os.Stderr, "repo-mapper: unknown command %q\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		if errors.Is(err, errNoChanges) {
			// Distinct exit code so automation can tell "nothing to
			// commit" apart from a real failure without parsing stdout.
			os.Exit(3)
		}
		fmt.Fprintf(os.Stderr, "repo-mapper: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`repo-mapper — AI-first repository architecture mapper

Usage:
  repo-mapper <command> [flags]

Commands:
  scan       Perform a full repository scan and generate documentation
  update     Use Git diff to update only what changed
  clean      Delete the local cache
  doctor     Validate the installation and environment
  plugins    List registered plugins
  graph      Print an architecture diagram
  export     Export the canonical JSON model

Run "repo-mapper <command> -h" for command-specific flags.
`)
}
