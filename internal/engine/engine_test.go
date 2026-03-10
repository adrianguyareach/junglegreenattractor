package engine

import (
	"testing"

	"github.com/adrianguyareach/gilbeys/internal/dot"
)

func TestSelectEdgeConditionMatch(t *testing.T) {
	g := dot.NewGraph("test")
	g.AddNode("A", map[string]string{})
	g.AddNode("B", map[string]string{})
	g.AddNode("C", map[string]string{})
	g.AddEdge("A", "B", map[string]string{"condition": "outcome=success", "label": "Yes"})
	g.AddEdge("A", "C", map[string]string{"condition": "outcome=fail", "label": "No"})

	node := g.Nodes["A"]
	outcome := &Outcome{Status: StatusSuccess}
	ctx := NewContext()

	edge := selectEdge(node, outcome, ctx, g)
	if edge == nil {
		t.Fatal("expected an edge to be selected")
	}
	if edge.To != "B" {
		t.Errorf("expected edge to B (success path), got %s", edge.To)
	}
}

func TestSelectEdgeWeight(t *testing.T) {
	g := dot.NewGraph("test")
	g.AddNode("A", map[string]string{})
	g.AddNode("B", map[string]string{})
	g.AddNode("C", map[string]string{})
	g.AddEdge("A", "B", map[string]string{"weight": "1"})
	g.AddEdge("A", "C", map[string]string{"weight": "5"})

	node := g.Nodes["A"]
	outcome := &Outcome{Status: StatusSuccess}
	ctx := NewContext()

	edge := selectEdge(node, outcome, ctx, g)
	if edge == nil {
		t.Fatal("expected an edge")
	}
	if edge.To != "C" {
		t.Errorf("expected edge to C (higher weight), got %s", edge.To)
	}
}

func TestSelectEdgeLexicalTiebreak(t *testing.T) {
	g := dot.NewGraph("test")
	g.AddNode("A", map[string]string{})
	g.AddNode("B", map[string]string{})
	g.AddNode("C", map[string]string{})
	g.AddEdge("A", "C", map[string]string{})
	g.AddEdge("A", "B", map[string]string{})

	node := g.Nodes["A"]
	outcome := &Outcome{Status: StatusSuccess}
	ctx := NewContext()

	edge := selectEdge(node, outcome, ctx, g)
	if edge == nil {
		t.Fatal("expected an edge")
	}
	if edge.To != "B" {
		t.Errorf("expected edge to B (lexical first), got %s", edge.To)
	}
}

func TestSelectEdgeSuggestedNextIDs(t *testing.T) {
	g := dot.NewGraph("test")
	g.AddNode("A", map[string]string{})
	g.AddNode("B", map[string]string{})
	g.AddNode("C", map[string]string{})
	g.AddEdge("A", "B", map[string]string{})
	g.AddEdge("A", "C", map[string]string{})

	node := g.Nodes["A"]
	outcome := &Outcome{Status: StatusSuccess, SuggestedNextIDs: []string{"C"}}
	ctx := NewContext()

	edge := selectEdge(node, outcome, ctx, g)
	if edge == nil {
		t.Fatal("expected an edge")
	}
	if edge.To != "C" {
		t.Errorf("expected edge to C (suggested), got %s", edge.To)
	}
}

func TestSelectEdgeNoEdges(t *testing.T) {
	g := dot.NewGraph("test")
	g.AddNode("A", map[string]string{})

	node := g.Nodes["A"]
	outcome := &Outcome{Status: StatusSuccess}
	ctx := NewContext()

	edge := selectEdge(node, outcome, ctx, g)
	if edge != nil {
		t.Error("expected nil edge for node with no outgoing edges")
	}
}
