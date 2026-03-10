package dot

// Graph represents a parsed DOT digraph.
type Graph struct {
	Name       string
	Attrs      map[string]string
	Nodes      map[string]*Node
	Edges      []*Edge
	NodeOrder  []string // preserves declaration order
	Subgraphs []*Subgraph
}

// Node represents a node in the graph.
type Node struct {
	ID    string
	Attrs map[string]string
}

// Edge represents a directed edge.
type Edge struct {
	From  string
	To    string
	Attrs map[string]string
}

// Subgraph represents a subgraph block.
type Subgraph struct {
	Name         string
	Label        string
	NodeDefaults map[string]string
	EdgeDefaults map[string]string
	NodeIDs      []string
}

func NewGraph(name string) *Graph {
	return &Graph{
		Name:  name,
		Attrs: make(map[string]string),
		Nodes: make(map[string]*Node),
	}
}

func (g *Graph) AddNode(id string, attrs map[string]string) *Node {
	if existing, ok := g.Nodes[id]; ok {
		for k, v := range attrs {
			existing.Attrs[k] = v
		}
		return existing
	}
	n := &Node{ID: id, Attrs: attrs}
	g.Nodes[id] = n
	g.NodeOrder = append(g.NodeOrder, id)
	return n
}

func (g *Graph) AddEdge(from, to string, attrs map[string]string) *Edge {
	e := &Edge{From: from, To: to, Attrs: attrs}
	g.Edges = append(g.Edges, e)
	return e
}

func (g *Graph) OutgoingEdges(nodeID string) []*Edge {
	var out []*Edge
	for _, e := range g.Edges {
		if e.From == nodeID {
			out = append(out, e)
		}
	}
	return out
}

func (g *Graph) IncomingEdges(nodeID string) []*Edge {
	var in []*Edge
	for _, e := range g.Edges {
		if e.To == nodeID {
			in = append(in, e)
		}
	}
	return in
}

// NodeAttr returns a node attribute with a fallback default.
func (n *Node) Attr(key, defaultVal string) string {
	if v, ok := n.Attrs[key]; ok {
		return v
	}
	return defaultVal
}

// GraphAttr returns a graph attribute with a fallback default.
func (g *Graph) GraphAttr(key, defaultVal string) string {
	if v, ok := g.Attrs[key]; ok {
		return v
	}
	return defaultVal
}

// EdgeAttr returns an edge attribute with a fallback default.
func (e *Edge) Attr(key, defaultVal string) string {
	if v, ok := e.Attrs[key]; ok {
		return v
	}
	return defaultVal
}
