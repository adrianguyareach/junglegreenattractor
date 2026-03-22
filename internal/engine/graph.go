package engine

import "github.com/adrianguyareach/junglegreenattractor/internal/dot"

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
	if node.Attr("shape", "") == "Msquare" {
		return true
	}
	return node.Attr("type", "") == "exit"
}

func checkGoalGates(graph *dot.Graph, outcomes map[string]*Outcome) (bool, *dot.Node) {
	for nodeID, outcome := range outcomes {
		node, ok := graph.Nodes[nodeID]
		if !ok {
			continue
		}
		if node.Attr("goal_gate", "") != "true" {
			continue
		}
		if outcome.Status != StatusSuccess && outcome.Status != StatusPartialSuccess {
			return false, node
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
