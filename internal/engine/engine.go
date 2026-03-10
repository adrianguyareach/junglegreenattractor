package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/adrianguyareach/gilbeys/internal/dot"
	"github.com/adrianguyareach/gilbeys/internal/event"
)

// Handler interface duplicated here to avoid import cycle.
// The actual implementation lives in handler package.
type NodeHandler interface {
	Execute(node *dot.Node, ctx *Context, graph *dot.Graph, logsRoot string) (*Outcome, error)
}

// HandlerResolver resolves a handler for a node.
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

	if err := os.MkdirAll(r.Config.LogsRoot, 0755); err != nil {
		return nil, fmt.Errorf("create logs root: %w", err)
	}

	writeManifest(r.Config.LogsRoot, r.Graph)

	r.Emitter.Emit(event.Event{
		Kind:    event.PipelineStarted,
		Message: r.Graph.GraphAttr("label", r.Graph.Name),
		Data:    map[string]any{"goal": r.Graph.GraphAttr("goal", "")},
	})

	startNode := findStartNode(r.Graph)
	if startNode == nil {
		return nil, fmt.Errorf("no start node found in graph")
	}

	var completedNodes []string
	nodeOutcomes := make(map[string]*Outcome)
	nodeRetries := make(map[string]int)

	currentNode := startNode
	var lastOutcome *Outcome
	stepIndex := 0

	for {
		node := currentNode

		// Terminal node check
		if isTerminal(node) {
			gateOK, failedGate := checkGoalGates(r.Graph, nodeOutcomes)
			if !gateOK && failedGate != nil {
				retryTarget := getRetryTarget(failedGate, r.Graph)
				if retryTarget != "" {
					if target, ok := r.Graph.Nodes[retryTarget]; ok {
						currentNode = target
						continue
					}
				}
				return &Outcome{
					Status:        StatusFail,
					FailureReason: fmt.Sprintf("Goal gate unsatisfied for node %q and no retry target", failedGate.ID),
				}, nil
			}
			break
		}

		// Sequential stage directory so folders sort in execution order (e.g. 01_start, 02_scaffold_project)
		stepIndex++
		stageDir := filepath.Join(r.Config.LogsRoot, fmt.Sprintf("%03d_%s", stepIndex, node.ID))
		if err := os.MkdirAll(stageDir, 0755); err != nil {
			return nil, fmt.Errorf("create stage dir: %w", err)
		}

		// Execute node handler with retry
		r.Emitter.Emit(event.Event{
			Kind:   event.StageStarted,
			NodeID: node.ID,
			Data:   map[string]any{"label": node.Attr("label", node.ID)},
		})

		retryPolicy := buildRetryPolicy(node, r.Graph)
		outcome, err := r.executeWithRetry(node, ctx, stageDir, retryPolicy, nodeRetries)
		if err != nil {
			r.Emitter.Emit(event.Event{
				Kind:    event.PipelineFailed,
				NodeID:  node.ID,
				Message: err.Error(),
			})
			return nil, fmt.Errorf("executing node %q: %w", node.ID, err)
		}

		completedNodes = append(completedNodes, node.ID)
		nodeOutcomes[node.ID] = outcome
		lastOutcome = outcome

		ctx.ApplyUpdates(outcome.ContextUpdates)
		ctx.Set("outcome", string(outcome.Status))
		if outcome.PreferredLabel != "" {
			ctx.Set("preferred_label", outcome.PreferredLabel)
		}
		ctx.Set("current_node", node.ID)

		// Write status to stage directory
		writeStatusFile(stageDir, outcome)

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

		// Save checkpoint
		cp := NewCheckpoint(ctx, node.ID, completedNodes, nodeRetries)
		if err := cp.Save(r.Config.LogsRoot); err != nil {
			// Non-fatal
			ctx.AppendLog(fmt.Sprintf("WARNING: failed to save checkpoint: %v", err))
		}
		r.Emitter.Emit(event.Event{Kind: event.CheckpointSaved, NodeID: node.ID})

		// Select next edge
		nextEdge := selectEdge(node, outcome, ctx, r.Graph)
		if nextEdge == nil {
			if outcome.Status == StatusFail {
				return outcome, fmt.Errorf("stage %q failed with no outgoing fail edge", node.ID)
			}
			break
		}

		// Handle loop_restart
		if nextEdge.Attr("loop_restart", "") == "true" {
			// Restart with fresh logs directory
			return r.Run()
		}

		// Advance
		nextNode, ok := r.Graph.Nodes[nextEdge.To]
		if !ok {
			return nil, fmt.Errorf("edge target node %q not found", nextEdge.To)
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

func (r *Runner) executeWithRetry(node *dot.Node, ctx *Context, stageDir string, policy retryPolicy, nodeRetries map[string]int) (*Outcome, error) {
	handler := r.Resolver.Resolve(node)
	if handler == nil {
		return &Outcome{Status: StatusFail, FailureReason: "no handler found for node " + node.ID}, nil
	}

	for attempt := 1; attempt <= policy.maxAttempts; attempt++ {
		outcome, err := func() (out *Outcome, retErr error) {
			defer func() {
				if r := recover(); r != nil {
					out = &Outcome{Status: StatusFail, FailureReason: fmt.Sprintf("handler panic: %v", r)}
				}
			}()
			return handler.Execute(node, ctx, r.Graph, stageDir)
		}()

		if err != nil {
			if attempt < policy.maxAttempts {
				delay := policy.delayForAttempt(attempt)
				r.Emitter.Emit(event.Event{
					Kind:   event.StageRetrying,
					NodeID: node.ID,
					Data:   map[string]any{"attempt": attempt, "delay_ms": delay.Milliseconds()},
				})
				time.Sleep(delay)
				continue
			}
			return &Outcome{Status: StatusFail, FailureReason: err.Error()}, nil
		}

		if outcome.Status == StatusSuccess || outcome.Status == StatusPartialSuccess {
			nodeRetries[node.ID] = 0
			return outcome, nil
		}

		if outcome.Status == StatusRetry {
			if attempt < policy.maxAttempts {
				nodeRetries[node.ID]++
				delay := policy.delayForAttempt(attempt)
				r.Emitter.Emit(event.Event{
					Kind:   event.StageRetrying,
					NodeID: node.ID,
					Data:   map[string]any{"attempt": attempt, "delay_ms": delay.Milliseconds()},
				})
				time.Sleep(delay)
				continue
			}
			if node.Attr("allow_partial", "") == "true" {
				return &Outcome{Status: StatusPartialSuccess, Notes: "retries exhausted, partial accepted"}, nil
			}
			return &Outcome{Status: StatusFail, FailureReason: "max retries exceeded"}, nil
		}

		if outcome.Status == StatusFail {
			return outcome, nil
		}

		return outcome, nil
	}

	return &Outcome{Status: StatusFail, FailureReason: "max retries exceeded"}, nil
}

// Edge selection algorithm (5-step priority).
func selectEdge(node *dot.Node, outcome *Outcome, ctx *Context, graph *dot.Graph) *dot.Edge {
	edges := graph.OutgoingEdges(node.ID)
	if len(edges) == 0 {
		return nil
	}

	// Step 1: Condition-matching edges
	var condMatched []*dot.Edge
	for _, e := range edges {
		cond := e.Attr("condition", "")
		if cond != "" && EvaluateCondition(cond, outcome, ctx) {
			condMatched = append(condMatched, e)
		}
	}
	if len(condMatched) > 0 {
		return bestByWeightThenLexical(condMatched)
	}

	// Step 2: Preferred label
	if outcome != nil && outcome.PreferredLabel != "" {
		for _, e := range edges {
			if normalizeLabel(e.Attr("label", "")) == normalizeLabel(outcome.PreferredLabel) {
				return e
			}
		}
	}

	// Step 3: Suggested next IDs
	if outcome != nil && len(outcome.SuggestedNextIDs) > 0 {
		for _, sugID := range outcome.SuggestedNextIDs {
			for _, e := range edges {
				if e.To == sugID {
					return e
				}
			}
		}
	}

	// Step 4 & 5: Weight with lexical tiebreak (unconditional only)
	var unconditional []*dot.Edge
	for _, e := range edges {
		if e.Attr("condition", "") == "" {
			unconditional = append(unconditional, e)
		}
	}
	if len(unconditional) > 0 {
		return bestByWeightThenLexical(unconditional)
	}

	return bestByWeightThenLexical(edges)
}

func bestByWeightThenLexical(edges []*dot.Edge) *dot.Edge {
	if len(edges) == 0 {
		return nil
	}
	sort.Slice(edges, func(i, j int) bool {
		wi := parseWeight(edges[i].Attr("weight", "0"))
		wj := parseWeight(edges[j].Attr("weight", "0"))
		if wi != wj {
			return wi > wj
		}
		return edges[i].To < edges[j].To
	})
	return edges[0]
}

func parseWeight(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func normalizeLabel(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	// Strip accelerator prefixes
	for _, pattern := range []string{`[`, `(`} {
		if strings.HasPrefix(label, pattern) {
			if idx := strings.IndexAny(label, "])-"); idx >= 0 {
				label = strings.TrimSpace(label[idx+1:])
			}
		}
	}
	return label
}

func findStartNode(graph *dot.Graph) *dot.Node {
	for _, n := range graph.Nodes {
		if n.Attr("shape", "") == "Mdiamond" {
			return n
		}
	}
	for _, id := range []string{"start", "Start"} {
		if n, ok := graph.Nodes[id]; ok {
			return n
		}
	}
	return nil
}

func isTerminal(node *dot.Node) bool {
	shape := node.Attr("shape", "")
	if shape == "Msquare" {
		return true
	}
	nodeType := node.Attr("type", "")
	return nodeType == "exit"
}

func checkGoalGates(graph *dot.Graph, outcomes map[string]*Outcome) (bool, *dot.Node) {
	for nodeID, outcome := range outcomes {
		node, ok := graph.Nodes[nodeID]
		if !ok {
			continue
		}
		if node.Attr("goal_gate", "") == "true" {
			if outcome.Status != StatusSuccess && outcome.Status != StatusPartialSuccess {
				return false, node
			}
		}
	}
	return true, nil
}

func getRetryTarget(node *dot.Node, graph *dot.Graph) string {
	if t := node.Attr("retry_target", ""); t != "" {
		return t
	}
	if t := node.Attr("fallback_retry_target", ""); t != "" {
		return t
	}
	if t := graph.GraphAttr("retry_target", ""); t != "" {
		return t
	}
	return graph.GraphAttr("fallback_retry_target", "")
}

func mirrorGraphAttributes(graph *dot.Graph, ctx *Context) {
	for k, v := range graph.Attrs {
		ctx.Set("graph."+k, v)
	}
}

func writeManifest(logsRoot string, graph *dot.Graph) {
	manifest := map[string]any{
		"name":       graph.Name,
		"goal":       graph.GraphAttr("goal", ""),
		"label":      graph.GraphAttr("label", ""),
		"started_at": time.Now().UTC().Format(time.RFC3339),
		"node_count": len(graph.Nodes),
		"edge_count": len(graph.Edges),
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	_ = os.WriteFile(filepath.Join(logsRoot, "manifest.json"), data, 0644)
}

func writeStatusFile(stageDir string, outcome *Outcome) {
	data, _ := json.MarshalIndent(outcome, "", "  ")
	_ = os.WriteFile(filepath.Join(stageDir, "status.json"), data, 0644)
}

// Retry policy
type retryPolicy struct {
	maxAttempts    int
	initialDelayMs int
	backoffFactor  float64
	maxDelayMs     int
	jitter         bool
}

func buildRetryPolicy(node *dot.Node, graph *dot.Graph) retryPolicy {
	maxRetries := 0
	if v := node.Attr("max_retries", ""); v != "" {
		maxRetries, _ = strconv.Atoi(v)
	}
	if maxRetries == 0 {
		if v := graph.GraphAttr("default_max_retry", ""); v != "" {
			maxRetries, _ = strconv.Atoi(v)
		}
	}

	return retryPolicy{
		maxAttempts:    maxRetries + 1,
		initialDelayMs: 200,
		backoffFactor:  2.0,
		maxDelayMs:     60000,
		jitter:         true,
	}
}

func (p retryPolicy) delayForAttempt(attempt int) time.Duration {
	delay := float64(p.initialDelayMs) * math.Pow(p.backoffFactor, float64(attempt-1))
	if delay > float64(p.maxDelayMs) {
		delay = float64(p.maxDelayMs)
	}
	if p.jitter {
		delay = delay * (0.5 + rand.Float64())
	}
	return time.Duration(delay) * time.Millisecond
}
