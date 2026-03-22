package engine

import (
	"sort"
	"strconv"
	"strings"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
)

// selectEdge implements the 5-step edge priority algorithm:
//  1. Condition-matching edges (highest weight wins, then lexical tiebreak)
//  2. Preferred label match from handler outcome
//  3. Suggested next node IDs from handler outcome
//  4. Highest-weight unconditional edge
//  5. Lexical tiebreak among remaining edges
func selectEdge(node *dot.Node, outcome *Outcome, ctx *Context, graph *dot.Graph) *dot.Edge {
	edges := graph.OutgoingEdges(node.ID)
	if len(edges) == 0 {
		return nil
	}

	if e := matchByCondition(edges, outcome, ctx); e != nil {
		return e
	}
	if e := matchByPreferredLabel(edges, outcome); e != nil {
		return e
	}
	if e := matchBySuggestedIDs(edges, outcome); e != nil {
		return e
	}
	return matchByWeightOrLexical(edges)
}

func matchByCondition(edges []*dot.Edge, outcome *Outcome, ctx *Context) *dot.Edge {
	var matched []*dot.Edge
	for _, e := range edges {
		cond := e.Attr("condition", "")
		if cond != "" && EvaluateCondition(cond, outcome, ctx) {
			matched = append(matched, e)
		}
	}
	return bestByWeightThenLexical(matched)
}

func matchByPreferredLabel(edges []*dot.Edge, outcome *Outcome) *dot.Edge {
	if outcome == nil || outcome.PreferredLabel == "" {
		return nil
	}
	target := normalizeLabel(outcome.PreferredLabel)
	for _, e := range edges {
		if normalizeLabel(e.Attr("label", "")) == target {
			return e
		}
	}
	return nil
}

func matchBySuggestedIDs(edges []*dot.Edge, outcome *Outcome) *dot.Edge {
	if outcome == nil || len(outcome.SuggestedNextIDs) == 0 {
		return nil
	}
	for _, sugID := range outcome.SuggestedNextIDs {
		for _, e := range edges {
			if e.To == sugID {
				return e
			}
		}
	}
	return nil
}

func matchByWeightOrLexical(edges []*dot.Edge) *dot.Edge {
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
	for _, pattern := range []string{"[", "("} {
		if strings.HasPrefix(label, pattern) {
			if idx := strings.IndexAny(label, "])-"); idx >= 0 {
				label = strings.TrimSpace(label[idx+1:])
			}
		}
	}
	return label
}
