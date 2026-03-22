package handler

import (
	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/engine"
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

// ManagerLoopHandler orchestrates observation cycles over a child pipeline.
type ManagerLoopHandler struct{}

func (h *ManagerLoopHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Manager loop handler (stub): " + node.ID,
	}, nil
}
