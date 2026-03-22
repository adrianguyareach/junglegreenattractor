package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/handler"
)

func graphCmd(progName string, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no pipeline file specified")
		fmt.Fprintf(os.Stderr, "Usage: %s graph <pipeline.dot>\n", progName)
		os.Exit(1)
	}

	source, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", args[0], err)
		os.Exit(1)
	}

	graph, err := dot.Parse(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	printGraphOverview(graph)
	printNodeTable(graph)
	printEdgeList(graph)
}

func printGraphOverview(graph *dot.Graph) {
	fmt.Printf("  Graph: %s\n", graph.Name)
	if goal := graph.GraphAttr("goal", ""); goal != "" {
		fmt.Printf("  Goal:  %s\n", goal)
	}
	fmt.Printf("  Nodes: %d    Edges: %d\n", len(graph.Nodes), len(graph.Edges))
	fmt.Println()
}

func printNodeTable(graph *dot.Graph) {
	fmt.Println("  Nodes")
	fmt.Println("  " + strings.Repeat("─", 60))
	for _, id := range graph.NodeOrder {
		node := graph.Nodes[id]
		nodeType := resolveNodeType(node)
		label := node.Attr("label", "")
		if label == "" {
			label = id
		}
		fmt.Printf("  %-20s  %-15s  %s\n", id, "["+nodeType+"]", label)
	}
	fmt.Println()
}

func resolveNodeType(node *dot.Node) string {
	if t := node.Attr("type", ""); t != "" {
		return t
	}
	shape := node.Attr("shape", "box")
	if ht, ok := handler.ShapeToType[shape]; ok {
		return ht
	}
	return "codergen"
}

func printEdgeList(graph *dot.Graph) {
	if len(graph.Edges) == 0 {
		return
	}
	fmt.Println("  Edges")
	fmt.Println("  " + strings.Repeat("─", 60))
	for _, e := range graph.Edges {
		label := e.Attr("label", "")
		cond := e.Attr("condition", "")
		extra := ""
		if label != "" {
			extra = fmt.Sprintf("  label=%q", label)
		}
		if cond != "" {
			extra += fmt.Sprintf("  condition=%q", cond)
		}
		fmt.Printf("  %s → %s%s\n", e.From, e.To, extra)
	}
	fmt.Println()
}
