package stylesheet

import (
	"strings"

	"github.com/adrianguyareach/gilbeys/internal/dot"
)

// Rule represents a parsed stylesheet rule.
type Rule struct {
	Selector    string
	Specificity int
	Properties  map[string]string
}

// Parse parses a CSS-like model stylesheet string into rules.
func Parse(source string) []Rule {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}

	var rules []Rule
	remaining := source

	for remaining != "" {
		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			break
		}

		braceIdx := strings.Index(remaining, "{")
		if braceIdx < 0 {
			break
		}

		selector := strings.TrimSpace(remaining[:braceIdx])
		remaining = remaining[braceIdx+1:]

		closeBrace := strings.Index(remaining, "}")
		if closeBrace < 0 {
			break
		}

		body := strings.TrimSpace(remaining[:closeBrace])
		remaining = remaining[closeBrace+1:]

		props := parseDeclarations(body)
		if len(props) == 0 {
			continue
		}

		specificity := 0
		switch {
		case strings.HasPrefix(selector, "#"):
			specificity = 2
		case strings.HasPrefix(selector, "."):
			specificity = 1
		case selector == "*":
			specificity = 0
		}

		rules = append(rules, Rule{
			Selector:    selector,
			Specificity: specificity,
			Properties:  props,
		})
	}

	return rules
}

func parseDeclarations(body string) map[string]string {
	props := make(map[string]string)
	decls := strings.Split(body, ";")
	for _, decl := range decls {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		colonIdx := strings.Index(decl, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(decl[:colonIdx])
		val := strings.TrimSpace(decl[colonIdx+1:])
		if key != "" && val != "" {
			props[key] = val
		}
	}
	return props
}

// Apply applies stylesheet rules to a graph, setting properties on nodes that
// don't already have them explicitly set.
func Apply(graph *dot.Graph, rules []Rule) {
	for _, node := range graph.Nodes {
		for _, rule := range rules {
			if matches(rule.Selector, node) {
				for key, val := range rule.Properties {
					if _, exists := node.Attrs[key]; !exists {
						node.Attrs[key] = val
					}
				}
			}
		}
	}

	// Second pass: higher specificity overrides lower (unless explicit)
	// We re-apply in specificity order
	for spec := 0; spec <= 2; spec++ {
		for _, rule := range rules {
			if rule.Specificity != spec {
				continue
			}
			for _, node := range graph.Nodes {
				if matches(rule.Selector, node) {
					for key, val := range rule.Properties {
						// Only override if the existing value came from a lower specificity rule
						node.Attrs["_ss_"+key+"_spec"] = ""
						node.Attrs[key] = val
					}
				}
			}
		}
	}

	// Clean up tracking keys
	for _, node := range graph.Nodes {
		for key := range node.Attrs {
			if strings.HasPrefix(key, "_ss_") {
				delete(node.Attrs, key)
			}
		}
	}
}

func matches(selector string, node *dot.Node) bool {
	switch {
	case selector == "*":
		return true
	case strings.HasPrefix(selector, "#"):
		return node.ID == selector[1:]
	case strings.HasPrefix(selector, "."):
		className := selector[1:]
		nodeClasses := node.Attr("class", "")
		for _, c := range strings.Split(nodeClasses, ",") {
			if strings.TrimSpace(c) == className {
				return true
			}
		}
		return false
	default:
		return node.Attr("shape", "box") == selector
	}
}
