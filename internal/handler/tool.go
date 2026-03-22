package handler

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/engine"
)

const defaultToolTimeout = 30 * time.Second

// ToolHandler executes a shell command specified by the node's tool_command
// attribute. The command is run with a configurable timeout.
type ToolHandler struct{}

func (h *ToolHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	command := node.Attr("tool_command", "")
	if command == "" {
		return &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: "No tool_command specified",
		}, nil
	}

	timeout := parseToolTimeout(node)
	output, err := runWithTimeout(command, timeout)
	if err != nil {
		return &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: err.Error(),
		}, nil
	}

	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Tool completed: " + command,
		ContextUpdates: map[string]string{
			"tool.output": string(output),
		},
	}, nil
}

func parseToolTimeout(node *dot.Node) time.Duration {
	raw := node.Attr("timeout", "")
	if raw == "" {
		return defaultToolTimeout
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return defaultToolTimeout
	}
	return d
}

func runWithTimeout(command string, timeout time.Duration) ([]byte, error) {
	cmd := exec.Command("sh", "-c", command)
	done := make(chan error, 1)
	var output []byte
	var cmdErr error

	go func() {
		output, cmdErr = cmd.CombinedOutput()
		done <- cmdErr
	}()

	select {
	case <-time.After(timeout):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil, fmt.Errorf("command timed out after %s", timeout)
	case <-done:
		if cmdErr != nil {
			return nil, fmt.Errorf("command failed: %v\nOutput: %s", cmdErr, string(output))
		}
		return output, nil
	}
}
