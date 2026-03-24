package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// Main is the CLI entry point for both jga and junglegreenattractor binaries.
func Main() {
	progName := filepath.Base(os.Args[0])
	if len(os.Args) < 2 {
		printUsage(progName)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd(progName, os.Args[2:])
	case "validate":
		validateCmd(progName, os.Args[2:])
	case "inspect":
		inspectCmd(progName, os.Args[2:])
	case "list":
		listCmd(progName, os.Args[2:])
	case "graph":
		graphCmd(progName, os.Args[2:])
	case "version":
		fmt.Println(progName, version)
	case "help", "--help", "-h":
		printUsage(progName)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage(progName)
		os.Exit(1)
	}
}
