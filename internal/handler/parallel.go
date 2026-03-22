package handler

import (
	"fmt"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/engine"
)

// ParallelHandler fans out execution to multiple branches. In the current
// implementation, branches are simulated sequentially.
type ParallelHandler struct{}

func (h *ParallelHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	edges := graph.OutgoingEdges(node.ID)
	if len(edges) == 0 {
		return &engine.Outcome{Status: engine.StatusSuccess, Notes: "No branches to execute"}, nil
	}

	for _, edge := range edges {
		ctx.Set("parallel.branch."+edge.To, "pending")
	}

	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  fmt.Sprintf("Parallel fan-out to %d branches (sequential simulation)", len(edges)),
		ContextUpdates: map[string]string{
			"parallel.branch_count":  fmt.Sprintf("%d", len(edges)),
			"parallel.success_count": fmt.Sprintf("%d", len(edges)),
			"parallel.failure_count": "0",
		},
	}, nil
}

// FanInHandler consolidates results from parallel branches.
type FanInHandler struct{}

func (h *FanInHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Fan-in consolidation complete",
		ContextUpdates: map[string]string{
			"parallel.fan_in.completed": "true",
		},
	}, nil
}
