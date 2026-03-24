package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/engine"
	"github.com/adrianguyareach/junglegreenattractor/internal/event"
	"github.com/adrianguyareach/junglegreenattractor/internal/handler"
	"github.com/adrianguyareach/junglegreenattractor/internal/interviewer"
	"github.com/adrianguyareach/junglegreenattractor/internal/transform"
	"github.com/adrianguyareach/junglegreenattractor/internal/validate"
)

type varFlags []string

func (v *varFlags) String() string { return strings.Join(*v, ", ") }
func (v *varFlags) Set(s string) error {
	*v = append(*v, s)
	return nil
}

func runCmd(progName string, args []string) {
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
	logDir := fs.String("log", ".jgattractorlogs", "Log directory")
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
		fmt.Fprintf(os.Stderr, "Usage: %s run <pipeline.dot> [flags]\n", progName)
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
	fmt.Printf("Graph: %s (%d nodes, %d edges)\n", graph.Name, len(graph.Nodes), len(graph.Edges))

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
	logsRoot := filepath.Join(*logDir, baseName)
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
