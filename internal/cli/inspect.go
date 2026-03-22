package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func inspectCmd(progName string, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no run path specified\n")
		fmt.Fprintf(os.Stderr, "Usage: %s inspect <run-path>\n", progName)
		fmt.Fprintf(os.Stderr, "  e.g. %s inspect .jgattractorlogs/init_rest_app\n", progName)
		os.Exit(1)
	}

	runPath := args[0]
	info, err := os.Stat(runPath)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %q is not a valid run directory\n", runPath)
		os.Exit(1)
	}

	printManifest(runPath)
	printStages(runPath)
	printCheckpointSummary(runPath)
}

func printManifest(runPath string) {
	data, err := os.ReadFile(filepath.Join(runPath, "manifest.json"))
	if err != nil {
		return
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		return
	}

	fmt.Println("  Pipeline Run")
	fmt.Println("  " + strings.Repeat("─", 50))
	for _, key := range []string{"name", "goal", "started_at", "node_count", "edge_count"} {
		if v, ok := manifest[key]; ok {
			fmt.Printf("  %-14s %v\n", key+":", v)
		}
	}
	fmt.Println()
}

func printStages(runPath string) {
	entries, err := os.ReadDir(runPath)
	if err != nil {
		return
	}

	var stages []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			stages = append(stages, e)
		}
	}
	sort.Slice(stages, func(i, j int) bool { return stages[i].Name() < stages[j].Name() })

	if len(stages) == 0 {
		return
	}

	fmt.Println("  Stages")
	fmt.Println("  " + strings.Repeat("─", 50))
	for _, stage := range stages {
		outcome := readStageOutcome(filepath.Join(runPath, stage.Name()))
		symbol := symbolForOutcome(outcome)
		fmt.Printf("  %s %s  [%s]\n", symbol, stage.Name(), outcome)
	}
	fmt.Println()
}

func readStageOutcome(stageDir string) string {
	data, err := os.ReadFile(filepath.Join(stageDir, "status.json"))
	if err != nil {
		return "unknown"
	}
	var status struct {
		Outcome string `json:"outcome"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return "unknown"
	}
	if status.Outcome == "" {
		return "unknown"
	}
	return status.Outcome
}

func symbolForOutcome(outcome string) string {
	switch outcome {
	case "success":
		return "✓"
	case "partial_success":
		return "◐"
	case "fail":
		return "✗"
	case "retry":
		return "↻"
	case "skipped":
		return "─"
	default:
		return "?"
	}
}

func printCheckpointSummary(runPath string) {
	data, err := os.ReadFile(filepath.Join(runPath, "checkpoint.json"))
	if err != nil {
		return
	}
	var cp struct {
		CurrentNode    string   `json:"current_node"`
		CompletedNodes []string `json:"completed_nodes"`
		Timestamp      string   `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &cp); err != nil {
		return
	}

	fmt.Println("  Checkpoint")
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Printf("  last_node:     %s\n", cp.CurrentNode)
	fmt.Printf("  completed:     %d stages\n", len(cp.CompletedNodes))
	fmt.Printf("  timestamp:     %s\n", cp.Timestamp)
	fmt.Println()
}
