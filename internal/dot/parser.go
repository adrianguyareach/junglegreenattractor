package dot

import (
	"fmt"
	"strings"
)

// Parse parses a DOT digraph source string into a Graph.
func Parse(source string) (*Graph, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, fmt.Errorf("lex error: %w", err)
	}
	p := &parser{tokens: tokens}
	return p.parseGraph()
}

type parser struct {
	tokens       []token
	pos          int
	nodeDefaults map[string]string
	edgeDefaults map[string]string
}

func (p *parser) cur() token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return token{kind: tokEOF}
}

func (p *parser) next() token {
	t := p.cur()
	p.pos++
	return t
}

func (p *parser) expect(kind tokenKind) (token, error) {
	t := p.next()
	if t.kind != kind {
		return t, fmt.Errorf("line %d: expected token kind %d, got %d (%q)", t.line, kind, t.kind, t.val)
	}
	return t, nil
}

func (p *parser) parseGraph() (*Graph, error) {
	if _, err := p.expect(tokDigraph); err != nil {
		return nil, fmt.Errorf("expected 'digraph': %w", err)
	}

	nameToken := p.cur()
	var name string
	if nameToken.kind == tokIdent || nameToken.kind == tokString {
		name = nameToken.val
		p.next()
	}

	if _, err := p.expect(tokLBrace); err != nil {
		return nil, err
	}

	g := NewGraph(name)
	p.nodeDefaults = make(map[string]string)
	p.edgeDefaults = make(map[string]string)

	if err := p.parseStatements(g, nil); err != nil {
		return nil, err
	}

	if _, err := p.expect(tokRBrace); err != nil {
		return nil, err
	}

	return g, nil
}

func (p *parser) parseStatements(g *Graph, sub *Subgraph) error {
	for {
		p.skipSemicolons()

		t := p.cur()
		if t.kind == tokRBrace || t.kind == tokEOF {
			return nil
		}

		switch t.kind {
		case tokGraph:
			p.next()
			if p.cur().kind == tokLBrack {
				attrs, err := p.parseAttrBlock()
				if err != nil {
					return err
				}
				for k, v := range attrs {
					g.Attrs[k] = v
				}
			}

		case tokNode:
			p.next()
			if p.cur().kind == tokLBrack {
				attrs, err := p.parseAttrBlock()
				if err != nil {
					return err
				}
				if sub != nil {
					if sub.NodeDefaults == nil {
						sub.NodeDefaults = make(map[string]string)
					}
					for k, v := range attrs {
						sub.NodeDefaults[k] = v
					}
				}
				for k, v := range attrs {
					p.nodeDefaults[k] = v
				}
			}

		case tokEdge:
			p.next()
			if p.cur().kind == tokLBrack {
				attrs, err := p.parseAttrBlock()
				if err != nil {
					return err
				}
				if sub != nil {
					if sub.EdgeDefaults == nil {
						sub.EdgeDefaults = make(map[string]string)
					}
					for k, v := range attrs {
						sub.EdgeDefaults[k] = v
					}
				}
				for k, v := range attrs {
					p.edgeDefaults[k] = v
				}
			}

		case tokSubgraph:
			if err := p.parseSubgraph(g); err != nil {
				return err
			}

		case tokIdent:
			if err := p.parseNodeOrEdge(g, sub); err != nil {
				return err
			}

		default:
			return fmt.Errorf("line %d: unexpected token %q", t.line, t.val)
		}

		p.skipSemicolons()
	}
}

func (p *parser) parseSubgraph(g *Graph) error {
	p.next() // consume 'subgraph'

	var subName string
	if p.cur().kind == tokIdent || p.cur().kind == tokString {
		subName = p.cur().val
		p.next()
	}

	if _, err := p.expect(tokLBrace); err != nil {
		return err
	}

	sub := &Subgraph{Name: subName, NodeDefaults: make(map[string]string), EdgeDefaults: make(map[string]string)}

	savedNodeDefaults := copyMap(p.nodeDefaults)
	savedEdgeDefaults := copyMap(p.edgeDefaults)

	if err := p.parseStatements(g, sub); err != nil {
		return err
	}

	if _, err := p.expect(tokRBrace); err != nil {
		return err
	}

	if sub.Label == "" {
		sub.Label = sub.NodeDefaults["label"]
	}
	if sub.Label == "" {
		sub.Label = subName
	}

	// Derive class from label
	if sub.Label != "" {
		derived := deriveClass(sub.Label)
		for _, nid := range sub.NodeIDs {
			if n, ok := g.Nodes[nid]; ok {
				existing := n.Attrs["class"]
				if existing == "" {
					n.Attrs["class"] = derived
				} else if !strings.Contains(existing, derived) {
					n.Attrs["class"] = existing + "," + derived
				}
			}
		}
	}

	g.Subgraphs = append(g.Subgraphs, sub)

	p.nodeDefaults = savedNodeDefaults
	p.edgeDefaults = savedEdgeDefaults

	return nil
}

func (p *parser) parseNodeOrEdge(g *Graph, sub *Subgraph) error {
	firstID := p.cur().val
	p.next()

	// Check for graph-level key = value
	if p.cur().kind == tokEquals {
		p.next()
		val := p.cur().val
		p.next()
		g.Attrs[firstID] = val
		return nil
	}

	// Chained edges: A -> B -> C [attrs]
	if p.cur().kind == tokArrow {
		nodeIDs := []string{firstID}
		for p.cur().kind == tokArrow {
			p.next() // consume ->
			if p.cur().kind != tokIdent {
				return fmt.Errorf("line %d: expected node ID after '->'", p.cur().line)
			}
			nodeIDs = append(nodeIDs, p.cur().val)
			p.next()
		}

		var attrs map[string]string
		if p.cur().kind == tokLBrack {
			var err error
			attrs, err = p.parseAttrBlock()
			if err != nil {
				return err
			}
		}

		// Ensure all nodes in the chain exist
		for _, nid := range nodeIDs {
			if _, ok := g.Nodes[nid]; !ok {
				nodeAttrs := copyMap(p.nodeDefaults)
				if sub != nil {
					for k, v := range sub.NodeDefaults {
						if _, exists := nodeAttrs[k]; !exists {
							nodeAttrs[k] = v
						}
					}
				}
				g.AddNode(nid, nodeAttrs)
			}
			if sub != nil {
				sub.NodeIDs = appendUnique(sub.NodeIDs, nid)
			}
		}

		// Create edges for each pair
		for i := 0; i < len(nodeIDs)-1; i++ {
			edgeAttrs := copyMap(p.edgeDefaults)
			for k, v := range attrs {
				edgeAttrs[k] = v
			}
			g.AddEdge(nodeIDs[i], nodeIDs[i+1], edgeAttrs)
		}

		return nil
	}

	// Node statement
	nodeAttrs := copyMap(p.nodeDefaults)
	if sub != nil {
		for k, v := range sub.NodeDefaults {
			if _, exists := nodeAttrs[k]; !exists {
				nodeAttrs[k] = v
			}
		}
	}

	if p.cur().kind == tokLBrack {
		parsed, err := p.parseAttrBlock()
		if err != nil {
			return err
		}
		for k, v := range parsed {
			nodeAttrs[k] = v
		}
	}

	g.AddNode(firstID, nodeAttrs)
	if sub != nil {
		sub.NodeIDs = appendUnique(sub.NodeIDs, firstID)
	}

	return nil
}

func (p *parser) parseAttrBlock() (map[string]string, error) {
	if _, err := p.expect(tokLBrack); err != nil {
		return nil, err
	}

	attrs := make(map[string]string)

	for p.cur().kind != tokRBrack && p.cur().kind != tokEOF {
		// Skip commas and semicolons between attrs
		for p.cur().kind == tokComma || p.cur().kind == tokSemicolon {
			p.next()
		}
		if p.cur().kind == tokRBrack {
			break
		}

		key := p.cur().val
		p.next()

		if p.cur().kind != tokEquals {
			return nil, fmt.Errorf("line %d: expected '=' after attribute key %q", p.cur().line, key)
		}
		p.next()

		val := p.cur().val
		p.next()

		attrs[key] = val

		// Skip trailing comma
		if p.cur().kind == tokComma || p.cur().kind == tokSemicolon {
			p.next()
		}
	}

	if _, err := p.expect(tokRBrack); err != nil {
		return nil, err
	}

	return attrs, nil
}

func (p *parser) skipSemicolons() {
	for p.cur().kind == tokSemicolon {
		p.next()
	}
}

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

func deriveClass(label string) string {
	label = strings.ToLower(label)
	label = strings.ReplaceAll(label, " ", "-")
	var b strings.Builder
	for _, r := range label {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
