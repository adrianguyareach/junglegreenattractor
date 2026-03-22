package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/engine"
)

const responseTruncationLimit = 200

// CodergenHandler calls the LLM backend or runs in simulation mode.
type CodergenHandler struct {
	Backend CodergenBackend
}

func (h *CodergenHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	prompt := resolvePrompt(node, graph)
	stageDir := logsRoot

	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return nil, fmt.Errorf("create stage dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(stageDir, "prompt.md"), []byte(prompt), 0644); err != nil {
		return nil, fmt.Errorf("write prompt: %w", err)
	}

	responseText, result := h.callBackend(node, prompt, ctx, stageDir)
	if result != nil {
		return result, nil
	}

	if err := os.WriteFile(filepath.Join(stageDir, "response.md"), []byte(responseText), 0644); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}

	outcome := &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Stage completed: " + node.ID,
		ContextUpdates: map[string]string{
			"last_stage":    node.ID,
			"last_response": truncateResponse(responseText),
		},
	}

	writeStatus(stageDir, outcome)
	return outcome, nil
}

func resolvePrompt(node *dot.Node, graph *dot.Graph) string {
	prompt := node.Attr("prompt", "")
	if prompt == "" {
		prompt = node.Attr("label", node.ID)
	}
	goal := graph.GraphAttr("goal", "")
	return strings.ReplaceAll(prompt, "$goal", goal)
}

func (h *CodergenHandler) callBackend(node *dot.Node, prompt string, ctx *engine.Context, stageDir string) (string, *engine.Outcome) {
	if h.Backend == nil {
		return fmt.Sprintf("[Simulated] Response for stage: %s", node.ID), nil
	}

	result, err := h.Backend.Run(node, prompt, ctx)
	if err != nil {
		return "", &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: err.Error(),
		}
	}
	if result != nil {
		writeStatus(stageDir, result)
		return "", result
	}
	return "[Backend returned no response]", nil
}

func truncateResponse(s string) string {
	if len(s) <= responseTruncationLimit {
		return s
	}
	return s[:responseTruncationLimit]
}

// writeStageSummary creates a single markdown file that aggregates the prompt,
// response, and status for a stage. This makes it easier to point external
// coding agents at one file instead of three separate ones.
func writeStageSummary(stageDir string) error {
	prompt, _ := os.ReadFile(filepath.Join(stageDir, "prompt.md"))
	response, _ := os.ReadFile(filepath.Join(stageDir, "response.md"))
	status, _ := os.ReadFile(filepath.Join(stageDir, "status.json"))

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

	return os.WriteFile(filepath.Join(stageDir, "stage.md"), []byte(b.String()), 0644)
}

func writeStatus(stageDir string, outcome *engine.Outcome) {
	_ = engine.WriteStatusFile(stageDir, outcome)
}
