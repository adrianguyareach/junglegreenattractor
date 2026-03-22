// Package engine implements the core pipeline execution loop. It traverses
// a DOT graph node-by-node, executing handlers, managing retries, and
// persisting checkpoints and status files along the way.
package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/event"
)

// NodeHandler is the interface every node executor must satisfy.
// Defined in the engine package to avoid an import cycle with handler.
type NodeHandler interface {
	Execute(node *dot.Node, ctx *Context, graph *dot.Graph, logsRoot string) (*Outcome, error)
}

// HandlerResolver maps a graph node to its concrete handler.
type HandlerResolver interface {
	Resolve(node *dot.Node) NodeHandler
}

// Config holds pipeline execution configuration.
type Config struct {
	LogsRoot string
	Vars     map[string]string
}

// Runner is the pipeline execution engine.
type Runner struct {
	Graph    *dot.Graph
	Config   Config
	Resolver HandlerResolver
	Emitter  *event.Emitter
}

// NewRunner creates a new pipeline runner.
func NewRunner(graph *dot.Graph, config Config, resolver HandlerResolver, emitter *event.Emitter) *Runner {
	return &Runner{
		Graph:    graph,
		Config:   config,
		Resolver: resolver,
		Emitter:  emitter,
	}
}

// Run executes the pipeline from start to completion.
func (r *Runner) Run() (*Outcome, error) {
	ctx := NewContext()
	mirrorGraphAttributes(r.Graph, ctx)

	if err := os.MkdirAll(r.Config.LogsRoot, dirPermissions); err != nil {
		return nil, fmt.Errorf("create logs root: %w", err)
	}
	if err := writeManifest(r.Config.LogsRoot, r.Graph); err != nil {
		ctx.AppendLog(fmt.Sprintf("WARNING: failed to write manifest: %v", err))
	}

	r.Emitter.Emit(event.Event{
		Kind:    event.PipelineStarted,
		Message: r.Graph.GraphAttr("label", r.Graph.Name),
		Data:    map[string]any{"goal": r.Graph.GraphAttr("goal", "")},
	})

	startNode := findStartNode(r.Graph)
	if startNode == nil {
		return nil, fmt.Errorf("no start node found in graph")
	}

	return r.runLoop(ctx, startNode)
}

func (r *Runner) runLoop(ctx *Context, startNode *dot.Node) (*Outcome, error) {
	var completedNodes []string
	nodeOutcomes := make(map[string]*Outcome)
	nodeRetries := make(map[string]int)
	currentNode := startNode
	var lastOutcome *Outcome
	stepIndex := 0

	for {
		if isTerminal(currentNode) {
			if failed := r.handleGoalGates(currentNode, nodeOutcomes); failed != nil {
				return failed, nil
			}
			break
		}

		stepIndex++
		stageDir := filepath.Join(r.Config.LogsRoot, fmt.Sprintf("%03d_%s", stepIndex, currentNode.ID))
		if err := os.MkdirAll(stageDir, dirPermissions); err != nil {
			return nil, fmt.Errorf("create stage dir: %w", err)
		}

		outcome, err := r.executeStage(currentNode, ctx, stageDir, nodeRetries)
		if err != nil {
			return nil, err
		}

		completedNodes = append(completedNodes, currentNode.ID)
		nodeOutcomes[currentNode.ID] = outcome
		lastOutcome = outcome

		r.recordOutcome(ctx, currentNode, outcome, stageDir, completedNodes, nodeRetries)

		nextNode, done, err := r.advance(currentNode, outcome, ctx)
		if err != nil {
			return outcome, err
		}
		if done {
			break
		}
		currentNode = nextNode
	}

	r.Emitter.Emit(event.Event{
		Kind:    event.PipelineCompleted,
		Message: "Pipeline completed",
		Data:    map[string]any{"completed_nodes": completedNodes},
	})

	if lastOutcome == nil {
		lastOutcome = &Outcome{Status: StatusSuccess, Notes: "Pipeline completed (empty)"}
	}
	return lastOutcome, nil
}

func (r *Runner) handleGoalGates(node *dot.Node, outcomes map[string]*Outcome) *Outcome {
	gateOK, failedGate := checkGoalGates(r.Graph, outcomes)
	if gateOK || failedGate == nil {
		return nil
	}
	retryTarget := getRetryTarget(failedGate, r.Graph)
	if retryTarget != "" {
		if _, ok := r.Graph.Nodes[retryTarget]; ok {
			return nil
		}
	}
	return &Outcome{
		Status:        StatusFail,
		FailureReason: fmt.Sprintf("Goal gate unsatisfied for node %q and no retry target", failedGate.ID),
	}
}

func (r *Runner) executeStage(node *dot.Node, ctx *Context, stageDir string, nodeRetries map[string]int) (*Outcome, error) {
	r.Emitter.Emit(event.Event{
		Kind:   event.StageStarted,
		NodeID: node.ID,
		Data:   map[string]any{"label": node.Attr("label", node.ID)},
	})

	policy := buildRetryPolicy(node, r.Graph)
	outcome, err := r.executeWithRetry(node, ctx, stageDir, policy, nodeRetries)
	if err != nil {
		r.Emitter.Emit(event.Event{
			Kind:    event.PipelineFailed,
			NodeID:  node.ID,
			Message: err.Error(),
		})
		return nil, fmt.Errorf("executing node %q: %w", node.ID, err)
	}
	return outcome, nil
}

func (r *Runner) recordOutcome(ctx *Context, node *dot.Node, outcome *Outcome, stageDir string, completed []string, retries map[string]int) {
	ctx.ApplyUpdates(outcome.ContextUpdates)
	ctx.Set("outcome", string(outcome.Status))
	if outcome.PreferredLabel != "" {
		ctx.Set("preferred_label", outcome.PreferredLabel)
	}
	ctx.Set("current_node", node.ID)

	if err := WriteStatusFile(stageDir, outcome); err != nil {
		ctx.AppendLog(fmt.Sprintf("WARNING: failed to write status: %v", err))
	}

	r.emitStageResult(node, outcome)

	cp := NewCheckpoint(ctx, node.ID, completed, retries)
	if err := cp.Save(r.Config.LogsRoot); err != nil {
		ctx.AppendLog(fmt.Sprintf("WARNING: failed to save checkpoint: %v", err))
	}
	r.Emitter.Emit(event.Event{Kind: event.CheckpointSaved, NodeID: node.ID})
}

func (r *Runner) emitStageResult(node *dot.Node, outcome *Outcome) {
	if outcome.Status == StatusSuccess || outcome.Status == StatusPartialSuccess {
		r.Emitter.Emit(event.Event{
			Kind:   event.StageCompleted,
			NodeID: node.ID,
			Data:   map[string]any{"status": string(outcome.Status)},
		})
	} else {
		r.Emitter.Emit(event.Event{
			Kind:    event.StageFailed,
			NodeID:  node.ID,
			Message: outcome.FailureReason,
		})
	}
}

func (r *Runner) advance(node *dot.Node, outcome *Outcome, ctx *Context) (*dot.Node, bool, error) {
	nextEdge := selectEdge(node, outcome, ctx, r.Graph)
	if nextEdge == nil {
		if outcome.Status == StatusFail {
			return nil, false, fmt.Errorf("stage %q failed with no outgoing fail edge", node.ID)
		}
		return nil, true, nil
	}

	if nextEdge.Attr("loop_restart", "") == "true" {
		restarted, err := r.Run()
		if err != nil {
			return nil, false, err
		}
		_ = restarted
		return nil, true, nil
	}

	nextNode, ok := r.Graph.Nodes[nextEdge.To]
	if !ok {
		return nil, false, fmt.Errorf("edge target node %q not found", nextEdge.To)
	}
	return nextNode, false, nil
}

func (r *Runner) executeWithRetry(node *dot.Node, ctx *Context, stageDir string, policy retryPolicy, nodeRetries map[string]int) (*Outcome, error) {
	handler := r.Resolver.Resolve(node)
	if handler == nil {
		return &Outcome{Status: StatusFail, FailureReason: "no handler found for node " + node.ID}, nil
	}

	for attempt := 1; attempt <= policy.maxAttempts; attempt++ {
		outcome := r.safeExecute(handler, node, ctx, stageDir)

		if outcome.Status == StatusSuccess || outcome.Status == StatusPartialSuccess {
			nodeRetries[node.ID] = 0
			return outcome, nil
		}

		if outcome.Status == StatusFail {
			return outcome, nil
		}

		if attempt < policy.maxAttempts {
			nodeRetries[node.ID]++
			r.sleepWithRetryEvent(node.ID, attempt, policy)
			continue
		}

		if node.Attr("allow_partial", "") == "true" {
			return &Outcome{Status: StatusPartialSuccess, Notes: "retries exhausted, partial accepted"}, nil
		}
		return &Outcome{Status: StatusFail, FailureReason: "max retries exceeded"}, nil
	}

	return &Outcome{Status: StatusFail, FailureReason: "max retries exceeded"}, nil
}

func (r *Runner) safeExecute(handler NodeHandler, node *dot.Node, ctx *Context, stageDir string) (outcome *Outcome) {
	defer func() {
		if rec := recover(); rec != nil {
			outcome = &Outcome{Status: StatusFail, FailureReason: fmt.Sprintf("handler panic: %v", rec)}
		}
	}()
	result, err := handler.Execute(node, ctx, r.Graph, stageDir)
	if err != nil {
		return &Outcome{Status: StatusFail, FailureReason: err.Error()}
	}
	return result
}

func (r *Runner) sleepWithRetryEvent(nodeID string, attempt int, policy retryPolicy) {
	delay := policy.delayForAttempt(attempt)
	r.Emitter.Emit(event.Event{
		Kind:   event.StageRetrying,
		NodeID: nodeID,
		Data:   map[string]any{"attempt": attempt, "delay_ms": delay.Milliseconds()},
	})
	time.Sleep(delay)
}
