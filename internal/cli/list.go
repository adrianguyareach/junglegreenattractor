package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const defaultLogDir = ".jgattractorlogs"

func listCmd(progName string, args []string) {
	logDir := defaultLogDir
	if len(args) > 0 {
		logDir = args[0]
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", logDir, err)
		fmt.Fprintf(os.Stderr, "  (No pipeline runs found. Run a pipeline first.)\n")
		os.Exit(1)
	}

	var runs []runSummary
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		runs = append(runs, summarizeRun(filepath.Join(logDir, e.Name()), e.Name()))
	}

	if len(runs) == 0 {
		fmt.Println("  No pipeline runs found in", logDir)
		return
	}

	sort.Slice(runs, func(i, j int) bool { return runs[i].startedAt > runs[j].startedAt })

	fmt.Printf("  Pipeline runs in %s\n", logDir)
	fmt.Println("  " + strings.Repeat("─", 60))
	for _, r := range runs {
		fmt.Printf("  %-30s  %-12s  %s\n", r.name, r.status, r.startedAt)
	}
	fmt.Println()
}

type runSummary struct {
	name      string
	status    string
	startedAt string
}

func summarizeRun(runPath, name string) runSummary {
	summary := runSummary{name: name, status: "unknown", startedAt: ""}

	if data, err := os.ReadFile(filepath.Join(runPath, "manifest.json")); err == nil {
		var m map[string]any
		if json.Unmarshal(data, &m) == nil {
			if v, ok := m["started_at"].(string); ok {
				summary.startedAt = v
			}
		}
	}

	if data, err := os.ReadFile(filepath.Join(runPath, "checkpoint.json")); err == nil {
		var cp struct {
			CompletedNodes []string `json:"completed_nodes"`
		}
		if json.Unmarshal(data, &cp) == nil && len(cp.CompletedNodes) > 0 {
			summary.status = fmt.Sprintf("%d stages", len(cp.CompletedNodes))
		}
	}

	return summary
}
