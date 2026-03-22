package transform

import (
	"strings"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/stylesheet"
)

// Transform modifies a graph in-place after parsing and before validation.
type Transform interface {
	Apply(graph *dot.Graph)
}

// VariableExpansion expands $goal in prompt attributes.
type VariableExpansion struct{}

func (t *VariableExpansion) Apply(graph *dot.Graph) {
	goal := graph.GraphAttr("goal", "")
	if goal == "" {
		return
	}
	for _, node := range graph.Nodes {
		if prompt, ok := node.Attrs["prompt"]; ok {
			node.Attrs["prompt"] = strings.ReplaceAll(prompt, "$goal", goal)
		}
	}
}

// StylesheetApplication applies the model_stylesheet graph attribute.
type StylesheetApplication struct{}

func (t *StylesheetApplication) Apply(graph *dot.Graph) {
	raw := graph.GraphAttr("model_stylesheet", "")
	if raw == "" {
		return
	}
	rules := stylesheet.Parse(raw)
	stylesheet.Apply(graph, rules)
}

// CustomVariableExpansion expands user-provided --var key=value pairs.
type CustomVariableExpansion struct {
	Vars map[string]string
}

func (t *CustomVariableExpansion) Apply(graph *dot.Graph) {
	if len(t.Vars) == 0 {
		return
	}

	expandInMap := func(attrs map[string]string) {
		for attrKey, attrVal := range attrs {
			for varKey, varVal := range t.Vars {
				placeholder := "$" + varKey
				if strings.Contains(attrVal, placeholder) {
					attrs[attrKey] = strings.ReplaceAll(attrVal, placeholder, varVal)
					attrVal = attrs[attrKey]
				}
			}
		}
	}

	expandInMap(graph.Attrs)
	for _, node := range graph.Nodes {
		expandInMap(node.Attrs)
	}
	for _, edge := range graph.Edges {
		expandInMap(edge.Attrs)
	}
}

// ApplyAll runs a list of transforms in order.
func ApplyAll(graph *dot.Graph, transforms []Transform) {
	for _, t := range transforms {
		t.Apply(graph)
	}
}
