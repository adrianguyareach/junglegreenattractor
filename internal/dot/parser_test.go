package dot

import (
	"testing"
)

func TestParseSimpleLinear(t *testing.T) {
	src := `digraph Simple {
		graph [goal="Run tests"]
		start [shape=Mdiamond]
		exit  [shape=Msquare]
		step1 [label="Step 1", prompt="Do step 1"]
		start -> step1 -> exit
	}`

	g, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if g.Name != "Simple" {
		t.Errorf("expected graph name 'Simple', got %q", g.Name)
	}
	if g.GraphAttr("goal", "") != "Run tests" {
		t.Errorf("expected goal 'Run tests', got %q", g.GraphAttr("goal", ""))
	}
	if len(g.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(g.Edges))
	}
	if g.Nodes["start"].Attr("shape", "") != "Mdiamond" {
		t.Error("expected start node shape Mdiamond")
	}
	if g.Nodes["step1"].Attr("prompt", "") != "Do step 1" {
		t.Errorf("expected prompt 'Do step 1', got %q", g.Nodes["step1"].Attr("prompt", ""))
	}
}

func TestParseChainedEdges(t *testing.T) {
	src := `digraph Test {
		A -> B -> C -> D [label="chain"]
	}`

	g, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(g.Edges) != 3 {
		t.Fatalf("expected 3 edges from chain, got %d", len(g.Edges))
	}
	if g.Edges[0].From != "A" || g.Edges[0].To != "B" {
		t.Error("first edge should be A->B")
	}
	if g.Edges[1].From != "B" || g.Edges[1].To != "C" {
		t.Error("second edge should be B->C")
	}
	if g.Edges[2].From != "C" || g.Edges[2].To != "D" {
		t.Error("third edge should be C->D")
	}
	for _, e := range g.Edges {
		if e.Attr("label", "") != "chain" {
			t.Error("all chained edges should carry the label attribute")
		}
	}
}

func TestParseNodeDefaults(t *testing.T) {
	src := `digraph Test {
		node [shape=box, timeout="60s"]
		A [label="Node A"]
		B [label="Node B", shape=diamond]
	}`

	g, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if g.Nodes["A"].Attr("shape", "") != "box" {
		t.Error("A should inherit shape=box from defaults")
	}
	if g.Nodes["A"].Attr("timeout", "") != "60s" {
		t.Error("A should inherit timeout=60s from defaults")
	}
	if g.Nodes["B"].Attr("shape", "") != "diamond" {
		t.Error("B should override shape to diamond")
	}
}

func TestParseEdgeAttributes(t *testing.T) {
	src := `digraph Test {
		start [shape=Mdiamond]
		exit  [shape=Msquare]
		A [shape=box]
		start -> A [label="begin"]
		A -> exit [label="done", condition="outcome=success", weight=5]
	}`

	g, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(g.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(g.Edges))
	}

	e := g.Edges[1]
	if e.Attr("condition", "") != "outcome=success" {
		t.Errorf("expected condition 'outcome=success', got %q", e.Attr("condition", ""))
	}
	if e.Attr("weight", "") != "5" {
		t.Errorf("expected weight '5', got %q", e.Attr("weight", ""))
	}
}

func TestParseComments(t *testing.T) {
	src := `
	// This is a line comment
	digraph Test {
		/* Block comment */
		start [shape=Mdiamond] // inline comment
		exit  [shape=Msquare]
		start -> exit
	}`

	g, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(g.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(g.Nodes))
	}
}

func TestParseSubgraph(t *testing.T) {
	src := `digraph Test {
		start [shape=Mdiamond]
		exit  [shape=Msquare]

		subgraph cluster_loop {
			label = "Loop A"
			node [thread_id="loop-a"]
			Plan [label="Plan next step"]
			Impl [label="Implement"]
		}

		start -> Plan -> Impl -> exit
	}`

	g, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(g.Subgraphs) != 1 {
		t.Fatalf("expected 1 subgraph, got %d", len(g.Subgraphs))
	}

	if g.Nodes["Plan"].Attr("thread_id", "") != "loop-a" {
		t.Error("Plan should inherit thread_id from subgraph")
	}
}

func TestParseGraphLevelKeyValue(t *testing.T) {
	src := `digraph Test {
		rankdir=LR
		start [shape=Mdiamond]
		exit  [shape=Msquare]
		start -> exit
	}`

	g, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if g.GraphAttr("rankdir", "") != "LR" {
		t.Errorf("expected rankdir=LR, got %q", g.GraphAttr("rankdir", ""))
	}
}

func TestParseMultilinePrompt(t *testing.T) {
	src := `digraph Test {
		start [shape=Mdiamond]
		exit  [shape=Msquare]
		step [label="Step", prompt="Line 1\nLine 2\nLine 3"]
		start -> step -> exit
	}`

	g, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	prompt := g.Nodes["step"].Attr("prompt", "")
	if prompt != "Line 1\nLine 2\nLine 3" {
		t.Errorf("expected multiline prompt, got %q", prompt)
	}
}

func TestParseGoalGate(t *testing.T) {
	src := `digraph Test {
		start [shape=Mdiamond]
		exit  [shape=Msquare]
		validate [label="Validate", goal_gate=true, max_retries=3]
		start -> validate -> exit
	}`

	g, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if g.Nodes["validate"].Attr("goal_gate", "") != "true" {
		t.Error("expected goal_gate=true")
	}
	if g.Nodes["validate"].Attr("max_retries", "") != "3" {
		t.Error("expected max_retries=3")
	}
}
