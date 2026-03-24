package cli

import (
	"fmt"
	"os"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/validate"
)

func validateCmd(progName string, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no pipeline file specified")
		fmt.Fprintf(os.Stderr, "Usage: %s validate <pipeline.dot>\n", progName)
		os.Exit(1)
	}

	source, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", args[0], err)
		os.Exit(1)
	}

	graph, err := dot.Parse(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	diags := validate.Validate(graph)

	hasErrors := false
	for _, d := range diags {
		fmt.Println(" ", d)
		if d.Severity == validate.SeverityError {
			hasErrors = true
		}
	}

	if hasErrors {
		fmt.Println("\n  Validation FAILED")
		os.Exit(1)
	}

	if len(diags) == 0 {
		fmt.Println("  ✓ Pipeline is valid")
	} else {
		fmt.Println("\n  Pipeline is valid (with warnings)")
	}
}
