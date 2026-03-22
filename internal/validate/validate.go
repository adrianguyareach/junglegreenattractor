// Package validate implements pipeline lint rules that check structural and
// semantic correctness of DOT graphs before execution. Diagnostics are
// returned at ERROR, WARNING, or INFO severity.
package validate

import (
	"fmt"
	"strings"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
)

// Severity classifies how serious a diagnostic is.
type Severity int

const (
	SeverityError   Severity = iota
	SeverityWarning
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "ERROR"
	case SeverityWarning:
		return "WARNING"
	case SeverityInfo:
		return "INFO"
	}
	return "UNKNOWN"
}

// Diagnostic is a single validation finding.
type Diagnostic struct {
	Rule     string
	Severity Severity
	Message  string
	NodeID   string
	Edge     [2]string
	Fix      string
}

func (d Diagnostic) String() string {
	prefix := d.Severity.String()
	if d.NodeID != "" {
		return fmt.Sprintf("[%s] %s (node: %s): %s", prefix, d.Rule, d.NodeID, d.Message)
	}
	if d.Edge[0] != "" {
		return fmt.Sprintf("[%s] %s (edge: %s->%s): %s", prefix, d.Rule, d.Edge[0], d.Edge[1], d.Message)
	}
	return fmt.Sprintf("[%s] %s: %s", prefix, d.Rule, d.Message)
}

// Validate runs all built-in lint rules on the graph and returns diagnostics.
func Validate(graph *dot.Graph) []Diagnostic {
	rules := []func(*dot.Graph) []Diagnostic{
		checkStartNode,
		checkTerminalNode,
		checkEdgeTargets,
		checkStartNoIncoming,
		checkExitNoOutgoing,
		checkReachability,
		checkPromptOnLLMNodes,
		checkRetryTargets,
		checkGoalGateHasRetry,
		checkConditionSyntax,
		checkFidelityValid,
	}

	var diags []Diagnostic
	for _, rule := range rules {
		diags = append(diags, rule(graph)...)
	}
	return diags
}

// ValidateOrRaise returns an error if there are any error-severity diagnostics.
func ValidateOrRaise(graph *dot.Graph) ([]Diagnostic, error) {
	diags := Validate(graph)
	var errors []string
	for _, d := range diags {
		if d.Severity == SeverityError {
			errors = append(errors, d.String())
		}
	}
	if len(errors) > 0 {
		return diags, fmt.Errorf("validation failed:\n  %s", strings.Join(errors, "\n  "))
	}
	return diags, nil
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

func findExitNode(graph *dot.Graph) *dot.Node {
	for _, n := range graph.Nodes {
		if n.Attr("shape", "") == "Msquare" {
			return n
		}
	}
	for _, id := range []string{"exit", "Exit", "end", "End"} {
		if n, ok := graph.Nodes[id]; ok {
			return n
		}
	}
	return nil
}
