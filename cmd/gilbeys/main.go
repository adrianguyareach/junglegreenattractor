package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrianguyareach/gilbeys/internal/dot"
	"github.com/adrianguyareach/gilbeys/internal/engine"
	"github.com/adrianguyareach/gilbeys/internal/event"
	"github.com/adrianguyareach/gilbeys/internal/handler"
	"github.com/adrianguyareach/gilbeys/internal/interviewer"
	"github.com/adrianguyareach/gilbeys/internal/transform"
	"github.com/adrianguyareach/gilbeys/internal/validate"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "validate":
		validateCmd(os.Args[2:])
	case "version":
		fmt.Println("gilbeys", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`gilbeys - A DOT-based pipeline runner for AI workflows

Usage:
  gilbeys run <pipeline.dot> [flags]    Run a pipeline
  gilbeys validate <pipeline.dot>       Validate a pipeline without running
  gilbeys version                       Print version
  gilbeys help                          Show this help

Run Flags:
  -log <dir>            Log directory (default: .attractorlogs)
  -name <name>          Run name for the log folder (default: derived from .dot filename)
  -var key=value        Set pipeline variable (repeatable)
  -auto-approve         Auto-approve all human gates
  -simulate             Run in simulation mode (no LLM backend)

Build:
  go build -o gilbeys ./cmd/gilbeys/

Examples:
  # Run a pipeline (logs default to .attractorlogs/<dot-filename>_<timestamp>/)
  gilbeys run pipeline.dot

  # Custom log directory
  gilbeys run pipeline.dot -log logs/.attractorlogs

  # With variables and auto-approve (for CI/CD)
  gilbeys run init_rest_app.dot -auto-approve \
    -var module_name="github.com/acme/api" \
    -var first_module="user"

  # Multiple variables
  gilbeys run add_module.dot \
    -var module_name=product \
    -var entity_fields="ID string, Name string, Price int64"

  # Custom run name
  gilbeys run full_feature.dot -name "feature-search-v2"

  # Validate only (no execution)
  gilbeys validate pipeline.dot`)
}

type varFlags []string

func (v *varFlags) String() string { return strings.Join(*v, ", ") }
func (v *varFlags) Set(s string) error {
	*v = append(*v, s)
	return nil
}

func runCmd(args []string) {
	// Extract the .dot file from args before flag parsing, since Go's flag
	// package stops at the first positional argument.
	var dotFile string
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			// Boolean flags don't consume the next arg; all others do.
			boolFlags := map[string]bool{"-auto-approve": true, "-simulate": true}
			if i+1 < len(args) && !strings.Contains(arg, "=") && !boolFlags[arg] {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else if dotFile == "" {
			dotFile = arg
		}
	}

	fs := flag.NewFlagSet("run", flag.ExitOnError)
	logDir := fs.String("log", ".attractorlogs", "Log directory")
	runName := fs.String("name", "", "Run name for the log folder (default: derived from .dot filename)")
	autoApprove := fs.Bool("auto-approve", false, "Auto-approve all human gates")
	simulate := fs.Bool("simulate", true, "Run in simulation mode (no LLM backend)")
	var vars varFlags
	fs.Var(&vars, "var", "Pipeline variable (key=value, repeatable)")

	if err := fs.Parse(flagArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if dotFile == "" {
		fmt.Fprintln(os.Stderr, "Error: no pipeline file specified")
		fmt.Fprintln(os.Stderr, "Usage: gilbeys run <pipeline.dot> [flags]")
		os.Exit(1)
	}

	// Parse variables
	varMap := make(map[string]string)
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Error: invalid variable format %q (expected key=value)\n", v)
			os.Exit(1)
		}
		varMap[parts[0]] = parts[1]
	}

	// Read DOT source
	source, err := os.ReadFile(dotFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", dotFile, err)
		os.Exit(1)
	}

	// Parse
	fmt.Printf("  Parsing %s...\n", dotFile)
	graph, err := dot.Parse(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Graph: %s (%d nodes, %d edges)\n", graph.Name, len(graph.Nodes), len(graph.Edges))

	// Apply transforms
	transforms := []transform.Transform{
		&transform.CustomVariableExpansion{Vars: varMap},
		&transform.VariableExpansion{},
		&transform.StylesheetApplication{},
	}
	transform.ApplyAll(graph, transforms)

	// Validate
	fmt.Println("  Validating...")
	diags, err := validate.ValidateOrRaise(graph)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Validation error:\n")
		for _, d := range diags {
			if d.Severity == validate.SeverityError {
				fmt.Fprintf(os.Stderr, "  %s\n", d)
			}
		}
		os.Exit(1)
	}
	for _, d := range diags {
		if d.Severity == validate.SeverityWarning {
			fmt.Printf("  WARNING: %s\n", d.Message)
		}
	}

	// Derive a human-readable run directory name from the .dot filename
	baseName := *runName
	if baseName == "" {
		baseName = strings.TrimSuffix(filepath.Base(dotFile), filepath.Ext(dotFile))
	}
	logsRoot := filepath.Join(*logDir, fmt.Sprintf("%s_%s", baseName, time.Now().Format("20060102_150405")))
	if err := os.MkdirAll(logsRoot, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log directory: %v\n", err)
		os.Exit(1)
	}

	// Set up interviewer
	var iv interviewer.Interviewer
	if *autoApprove {
		iv = interviewer.NewAutoApproveInterviewer()
	} else {
		iv = interviewer.NewConsoleInterviewer()
	}

	// Set up backend
	var backend handler.CodergenBackend
	if !*simulate {
		fmt.Println("  Note: LLM backend not configured; running in simulation mode")
	}
	_ = backend // nil = simulation mode

	// Build handler registry
	reg := handler.BuildDefaultRegistry(backend, iv)
	resolver := &handler.RegistryAdapter{Reg: reg}

	// Set up event emitter
	emitter := event.NewEmitter()
	emitter.On(func(evt event.Event) {
		switch evt.Kind {
		case event.PipelineStarted:
			goal, _ := evt.Data["goal"].(string)
			fmt.Printf("\n  Pipeline started")
			if goal != "" {
				fmt.Printf(" — goal: %s", truncate(goal, 80))
			}
			fmt.Println()
		case event.StageStarted:
			label, _ := evt.Data["label"].(string)
			if label == "" {
				label = evt.NodeID
			}
			fmt.Printf("  ► [%s] %s\n", evt.NodeID, label)
		case event.StageCompleted:
			fmt.Printf("    ✓ completed\n")
		case event.StageFailed:
			fmt.Printf("    ✗ failed: %s\n", evt.Message)
		case event.StageRetrying:
			attempt, _ := evt.Data["attempt"].(int)
			fmt.Printf("    ↻ retrying (attempt %d)\n", attempt)
		case event.PipelineCompleted:
			fmt.Printf("\n  Pipeline completed successfully.\n")
		case event.PipelineFailed:
			fmt.Printf("\n  Pipeline FAILED: %s\n", evt.Message)
		case event.CheckpointSaved:
			// silent
		}
	})

	// Run
	config := engine.Config{
		LogsRoot: logsRoot,
		Vars:     varMap,
	}

	runner := engine.NewRunner(graph, config, resolver, emitter)
	fmt.Printf("  Logs: %s\n", logsRoot)

	outcome, err := runner.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Outcome: %s\n", outcome.Status)
	if outcome.Notes != "" {
		fmt.Printf("  Notes: %s\n", outcome.Notes)
	}
	fmt.Printf("  Logs written to: %s\n", logsRoot)
}

func validateCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no pipeline file specified")
		fmt.Fprintln(os.Stderr, "Usage: gilbeys validate <pipeline.dot>")
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
