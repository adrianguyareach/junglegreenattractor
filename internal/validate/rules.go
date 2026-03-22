package validate

import (
	"fmt"
	"strings"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/handler"
)

func checkStartNode(graph *dot.Graph) []Diagnostic {
	count := 0
	for _, n := range graph.Nodes {
		if n.Attr("shape", "") == "Mdiamond" {
			count++
		}
	}
	if count == 0 {
		if _, ok := graph.Nodes["start"]; !ok {
			if _, ok := graph.Nodes["Start"]; !ok {
				return []Diagnostic{{
					Rule:     "start_node",
					Severity: SeverityError,
					Message:  "Pipeline must have exactly one start node (shape=Mdiamond)",
					Fix:      "Add a node with shape=Mdiamond",
				}}
			}
		}
	}
	if count > 1 {
		return []Diagnostic{{
			Rule:     "start_node",
			Severity: SeverityError,
			Message:  fmt.Sprintf("Pipeline has %d start nodes; exactly one is required", count),
		}}
	}
	return nil
}

func checkTerminalNode(graph *dot.Graph) []Diagnostic {
	count := 0
	for _, n := range graph.Nodes {
		if n.Attr("shape", "") == "Msquare" {
			count++
		}
	}
	if count == 0 {
		found := false
		for _, id := range []string{"exit", "Exit", "end", "End"} {
			if _, ok := graph.Nodes[id]; ok {
				found = true
				break
			}
		}
		if !found {
			return []Diagnostic{{
				Rule:     "terminal_node",
				Severity: SeverityError,
				Message:  "Pipeline must have at least one terminal node (shape=Msquare)",
				Fix:      "Add a node with shape=Msquare",
			}}
		}
	}
	return nil
}

func checkEdgeTargets(graph *dot.Graph) []Diagnostic {
	var diags []Diagnostic
	for _, e := range graph.Edges {
		if _, ok := graph.Nodes[e.From]; !ok {
			diags = append(diags, Diagnostic{
				Rule:     "edge_target_exists",
				Severity: SeverityError,
				Message:  fmt.Sprintf("Edge source node %q does not exist", e.From),
				Edge:     [2]string{e.From, e.To},
			})
		}
		if _, ok := graph.Nodes[e.To]; !ok {
			diags = append(diags, Diagnostic{
				Rule:     "edge_target_exists",
				Severity: SeverityError,
				Message:  fmt.Sprintf("Edge target node %q does not exist", e.To),
				Edge:     [2]string{e.From, e.To},
			})
		}
	}
	return diags
}

func checkStartNoIncoming(graph *dot.Graph) []Diagnostic {
	startNode := findStartNode(graph)
	if startNode == nil {
		return nil
	}
	for _, e := range graph.Edges {
		if e.To == startNode.ID {
			return []Diagnostic{{
				Rule:     "start_no_incoming",
				Severity: SeverityError,
				NodeID:   startNode.ID,
				Message:  "Start node must have no incoming edges",
				Edge:     [2]string{e.From, e.To},
			}}
		}
	}
	return nil
}

func checkExitNoOutgoing(graph *dot.Graph) []Diagnostic {
	exitNode := findExitNode(graph)
	if exitNode == nil {
		return nil
	}
	for _, e := range graph.Edges {
		if e.From == exitNode.ID {
			return []Diagnostic{{
				Rule:     "exit_no_outgoing",
				Severity: SeverityError,
				NodeID:   exitNode.ID,
				Message:  "Exit node must have no outgoing edges",
				Edge:     [2]string{e.From, e.To},
			}}
		}
	}
	return nil
}

func checkReachability(graph *dot.Graph) []Diagnostic {
	startNode := findStartNode(graph)
	if startNode == nil {
		return nil
	}

	visited := make(map[string]bool)
	queue := []string{startNode.ID}
	visited[startNode.ID] = true

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, e := range graph.Edges {
			if e.From == cur && !visited[e.To] {
				visited[e.To] = true
				queue = append(queue, e.To)
			}
		}
	}

	var diags []Diagnostic
	for id := range graph.Nodes {
		if !visited[id] {
			diags = append(diags, Diagnostic{
				Rule:     "reachability",
				Severity: SeverityError,
				NodeID:   id,
				Message:  fmt.Sprintf("Node %q is not reachable from start node", id),
				Fix:      "Add an edge from an existing node to this node, or remove it",
			})
		}
	}
	return diags
}

func checkPromptOnLLMNodes(graph *dot.Graph) []Diagnostic {
	var diags []Diagnostic
	for _, n := range graph.Nodes {
		if resolveHandlerType(n) != "codergen" {
			continue
		}
		if n.Attr("prompt", "") == "" && n.Attr("label", "") == "" {
			diags = append(diags, Diagnostic{
				Rule:     "prompt_on_llm_nodes",
				Severity: SeverityWarning,
				NodeID:   n.ID,
				Message:  "Codergen node has no prompt or label",
				Fix:      "Add a prompt or label attribute",
			})
		}
	}
	return diags
}

func resolveHandlerType(n *dot.Node) string {
	if t := n.Attr("type", ""); t != "" {
		return t
	}
	shape := n.Attr("shape", "box")
	if ht, ok := handler.ShapeToType[shape]; ok {
		return ht
	}
	return "codergen"
}

func checkRetryTargets(graph *dot.Graph) []Diagnostic {
	var diags []Diagnostic
	for _, n := range graph.Nodes {
		for _, attr := range []string{"retry_target", "fallback_retry_target"} {
			target := n.Attr(attr, "")
			if target != "" {
				if _, ok := graph.Nodes[target]; !ok {
					diags = append(diags, Diagnostic{
						Rule:     "retry_target_exists",
						Severity: SeverityWarning,
						NodeID:   n.ID,
						Message:  fmt.Sprintf("%s references non-existent node %q", attr, target),
					})
				}
			}
		}
	}
	return diags
}

func checkGoalGateHasRetry(graph *dot.Graph) []Diagnostic {
	var diags []Diagnostic
	for _, n := range graph.Nodes {
		if n.Attr("goal_gate", "") != "true" {
			continue
		}
		hasRetry := n.Attr("retry_target", "") != "" ||
			n.Attr("fallback_retry_target", "") != "" ||
			graph.GraphAttr("retry_target", "") != ""
		if !hasRetry {
			diags = append(diags, Diagnostic{
				Rule:     "goal_gate_has_retry",
				Severity: SeverityWarning,
				NodeID:   n.ID,
				Message:  "Goal gate node has no retry_target or fallback_retry_target",
				Fix:      "Add a retry_target attribute to this node or the graph",
			})
		}
	}
	return diags
}

func checkConditionSyntax(graph *dot.Graph) []Diagnostic {
	var diags []Diagnostic
	for _, e := range graph.Edges {
		cond := e.Attr("condition", "")
		if cond == "" {
			continue
		}
		for _, clause := range strings.Split(cond, "&&") {
			clause = strings.TrimSpace(clause)
			if clause == "" || !strings.Contains(clause, "=") {
				continue
			}
			if !strings.Contains(clause, "!=") && !strings.Contains(clause, "=") {
				diags = append(diags, Diagnostic{
					Rule:     "condition_syntax",
					Severity: SeverityError,
					Edge:     [2]string{e.From, e.To},
					Message:  fmt.Sprintf("Invalid condition clause: %q", clause),
				})
			}
		}
	}
	return diags
}

func checkFidelityValid(graph *dot.Graph) []Diagnostic {
	validFidelity := map[string]bool{
		"full": true, "truncate": true, "compact": true,
		"summary:low": true, "summary:medium": true, "summary:high": true,
		"": true,
	}

	var diags []Diagnostic
	for _, n := range graph.Nodes {
		f := n.Attr("fidelity", "")
		if !validFidelity[f] {
			diags = append(diags, Diagnostic{
				Rule:     "fidelity_valid",
				Severity: SeverityWarning,
				NodeID:   n.ID,
				Message:  fmt.Sprintf("Invalid fidelity mode: %q", f),
			})
		}
	}
	return diags
}
