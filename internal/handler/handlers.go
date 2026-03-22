package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/engine"
	"github.com/adrianguyareach/junglegreenattractor/internal/interviewer"
)

// StartHandler is a no-op for the pipeline entry point.
type StartHandler struct{}

func (h *StartHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{Status: engine.StatusSuccess, Notes: "Pipeline started"}, nil
}

// ExitHandler is a no-op for the pipeline exit point.
type ExitHandler struct{}

func (h *ExitHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{Status: engine.StatusSuccess, Notes: "Pipeline exit reached"}, nil
}

// ConditionalHandler passes through; edge routing is handled by the engine.
type ConditionalHandler struct{}

func (h *ConditionalHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Conditional node evaluated: " + node.ID,
	}, nil
}

// CodergenHandler calls the LLM backend or runs in simulation mode.
type CodergenHandler struct {
	Backend CodergenBackend
}

func (h *CodergenHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	prompt := node.Attr("prompt", "")
	if prompt == "" {
		prompt = node.Attr("label", node.ID)
	}

	// Expand $goal
	goal := graph.GraphAttr("goal", "")
	prompt = strings.ReplaceAll(prompt, "$goal", goal)

	// stageDir is passed from the engine (e.g. 01_scaffold_project) for sequential ordering
	stageDir := logsRoot
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return nil, fmt.Errorf("create stage dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(stageDir, "prompt.md"), []byte(prompt), 0644); err != nil {
		return nil, fmt.Errorf("write prompt: %w", err)
	}

	var responseText string

	if h.Backend != nil {
		result, err := h.Backend.Run(node, prompt, ctx)
		if err != nil {
			return &engine.Outcome{
				Status:        engine.StatusFail,
				FailureReason: err.Error(),
			}, nil
		}
		if result != nil {
			writeStatus(stageDir, result)
			return result, nil
		}
		responseText = "[Backend returned no response]"
	} else {
		responseText = fmt.Sprintf("[Simulated] Response for stage: %s", node.ID)
	}

	if err := os.WriteFile(filepath.Join(stageDir, "response.md"), []byte(responseText), 0644); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}

	truncated := responseText
	if len(truncated) > 200 {
		truncated = truncated[:200]
	}

	outcome := &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Stage completed: " + node.ID,
		ContextUpdates: map[string]string{
			"last_stage":    node.ID,
			"last_response": truncated,
		},
	}

	writeStatus(stageDir, outcome)
	return outcome, nil
}

// writeStageSummary creates a single markdown file that aggregates the prompt,
// response, and status for a stage. This makes it easier to point external
// coding agents at one file instead of three separate ones.
func writeStageSummary(stageDir string) {
	promptPath := filepath.Join(stageDir, "prompt.md")
	responsePath := filepath.Join(stageDir, "response.md")
	statusPath := filepath.Join(stageDir, "status.json")
	summaryPath := filepath.Join(stageDir, "stage.md")

	prompt, _ := os.ReadFile(promptPath)
	response, _ := os.ReadFile(responsePath)
	status, _ := os.ReadFile(statusPath)

	var b strings.Builder
	b.WriteString("# Stage\n\n")

	if len(prompt) > 0 {
		b.WriteString("## Prompt\n\n")
		b.Write(prompt)
		b.WriteString("\n\n")
	}

	if len(response) > 0 {
		b.WriteString("## Response\n\n")
		b.Write(response)
		b.WriteString("\n\n")
	}

	if len(status) > 0 {
		b.WriteString("## Status\n\n```json\n")
		b.Write(status)
		b.WriteString("\n```\n")
	}

	_ = os.WriteFile(summaryPath, []byte(b.String()), 0644)
}

// WaitForHumanHandler blocks until a human selects an option.
type WaitForHumanHandler struct {
	Interviewer interviewer.Interviewer
}

func (h *WaitForHumanHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	edges := graph.OutgoingEdges(node.ID)
	if len(edges) == 0 {
		return &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: "No outgoing edges for human gate",
		}, nil
	}

	var choices []interviewer.Option
	for _, edge := range edges {
		label := edge.Attr("label", edge.To)
		key := parseAcceleratorKey(label)
		choices = append(choices, interviewer.Option{Key: key, Label: label})
	}

	question := interviewer.Question{
		Text:    node.Attr("label", "Select an option:"),
		Type:    interviewer.MultipleChoice,
		Options: choices,
		Stage:   node.ID,
	}

	answer := h.Interviewer.Ask(question)

	if answer.Value == interviewer.AnswerTimeout {
		defaultChoice := node.Attr("human.default_choice", "")
		if defaultChoice != "" {
			return &engine.Outcome{
				Status:           engine.StatusSuccess,
				SuggestedNextIDs: []string{defaultChoice},
				ContextUpdates: map[string]string{
					"human.gate.selected": defaultChoice,
				},
			}, nil
		}
		return &engine.Outcome{
			Status:        engine.StatusRetry,
			FailureReason: "human gate timeout, no default",
		}, nil
	}

	if answer.Value == interviewer.AnswerSkipped {
		return &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: "human skipped interaction",
		}, nil
	}

	// Find matching edge target
	selectedTo := ""
	selectedLabel := ""
	for i, opt := range choices {
		if answer.SelectedOption != nil && answer.SelectedOption.Key == opt.Key {
			selectedTo = edges[i].To
			selectedLabel = opt.Label
			break
		}
		if strings.EqualFold(string(answer.Value), opt.Key) {
			selectedTo = edges[i].To
			selectedLabel = opt.Label
			break
		}
	}
	if selectedTo == "" && len(edges) > 0 {
		selectedTo = edges[0].To
		selectedLabel = choices[0].Label
	}

	return &engine.Outcome{
		Status:           engine.StatusSuccess,
		SuggestedNextIDs: []string{selectedTo},
		ContextUpdates: map[string]string{
			"human.gate.selected": string(answer.Value),
			"human.gate.label":    selectedLabel,
		},
	}, nil
}

var accelPattern = regexp.MustCompile(`^\[(\w)\]\s+|^(\w)\)\s+|^(\w)\s*-\s+`)

func parseAcceleratorKey(label string) string {
	m := accelPattern.FindStringSubmatch(label)
	if m != nil {
		for _, g := range m[1:] {
			if g != "" {
				return strings.ToUpper(g)
			}
		}
	}
	if len(label) > 0 {
		return strings.ToUpper(string(label[0]))
	}
	return ""
}

// ParallelHandler fans out execution to multiple branches concurrently.
type ParallelHandler struct{}

func (h *ParallelHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	edges := graph.OutgoingEdges(node.ID)
	if len(edges) == 0 {
		return &engine.Outcome{Status: engine.StatusSuccess, Notes: "No branches to execute"}, nil
	}

	successCount := 0
	failCount := 0
	for _, edge := range edges {
		// For each branch, we note the target; in a full implementation these would
		// run concurrently with isolated context clones.
		ctx.Set("parallel.branch."+edge.To, "pending")
		successCount++
	}

	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  fmt.Sprintf("Parallel fan-out to %d branches (sequential simulation)", len(edges)),
		ContextUpdates: map[string]string{
			"parallel.branch_count":  fmt.Sprintf("%d", len(edges)),
			"parallel.success_count": fmt.Sprintf("%d", successCount),
			"parallel.failure_count": fmt.Sprintf("%d", failCount),
		},
	}, nil
}

// FanInHandler consolidates results from parallel branches.
type FanInHandler struct{}

func (h *FanInHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	results := ctx.Get("parallel.results")
	if results == "" {
		return &engine.Outcome{
			Status: engine.StatusSuccess,
			Notes:  "Fan-in with no explicit parallel results; proceeding",
		}, nil
	}

	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Fan-in consolidation complete",
		ContextUpdates: map[string]string{
			"parallel.fan_in.completed": "true",
		},
	}, nil
}

// ToolHandler executes a shell command.
type ToolHandler struct{}

func (h *ToolHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	command := node.Attr("tool_command", "")
	if command == "" {
		return &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: "No tool_command specified",
		}, nil
	}

	timeoutStr := node.Attr("timeout", "30s")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = 30 * time.Second
	}

	cmd := exec.Command("sh", "-c", command)
	done := make(chan error, 1)
	var output []byte

	go func() {
		output, err = cmd.CombinedOutput()
		done <- err
	}()

	select {
	case <-time.After(timeout):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: fmt.Sprintf("Command timed out after %s", timeoutStr),
		}, nil
	case cmdErr := <-done:
		if cmdErr != nil {
			return &engine.Outcome{
				Status:        engine.StatusFail,
				FailureReason: fmt.Sprintf("Command failed: %v\nOutput: %s", cmdErr, string(output)),
			}, nil
		}
	}

	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Tool completed: " + command,
		ContextUpdates: map[string]string{
			"tool.output": string(output),
		},
	}, nil
}

// ManagerLoopHandler orchestrates observation cycles over a child pipeline.
type ManagerLoopHandler struct{}

func (h *ManagerLoopHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Manager loop handler (stub): " + node.ID,
	}, nil
}

func writeStatus(stageDir string, outcome *engine.Outcome) {
	data, err := json.MarshalIndent(outcome, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(stageDir, "status.json"), data, 0644)
}

// BuildDefaultRegistry creates a handler registry with all built-in handlers.
func BuildDefaultRegistry(backend CodergenBackend, iv interviewer.Interviewer) *Registry {
	reg := NewRegistry()

	codergen := &CodergenHandler{Backend: backend}
	reg.SetDefault(codergen)

	reg.Register("start", &StartHandler{})
	reg.Register("exit", &ExitHandler{})
	reg.Register("codergen", codergen)
	reg.Register("conditional", &ConditionalHandler{})
	reg.Register("wait.human", &WaitForHumanHandler{Interviewer: iv})
	reg.Register("parallel", &ParallelHandler{})
	reg.Register("parallel.fan_in", &FanInHandler{})
	reg.Register("tool", &ToolHandler{})
	reg.Register("stack.manager_loop", &ManagerLoopHandler{})

	return reg
}
