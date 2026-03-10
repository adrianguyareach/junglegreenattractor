package handler

import (
	"github.com/adrianguyareach/gilbeys/internal/dot"
	"github.com/adrianguyareach/gilbeys/internal/engine"
)

// Handler is the interface every node handler must implement.
type Handler interface {
	Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error)
}

// CodergenBackend is the pluggable LLM execution backend.
type CodergenBackend interface {
	Run(node *dot.Node, prompt string, ctx *engine.Context) (*engine.Outcome, error)
}

// ShapeToType maps DOT shapes to handler type strings.
var ShapeToType = map[string]string{
	"Mdiamond":      "start",
	"Msquare":       "exit",
	"box":           "codergen",
	"hexagon":       "wait.human",
	"diamond":       "conditional",
	"component":     "parallel",
	"tripleoctagon": "parallel.fan_in",
	"parallelogram": "tool",
	"house":         "stack.manager_loop",
}

// Registry maps type strings to handler instances.
type Registry struct {
	handlers       map[string]Handler
	defaultHandler Handler
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]Handler),
	}
}

func (r *Registry) Register(typeStr string, h Handler) {
	r.handlers[typeStr] = h
}

func (r *Registry) SetDefault(h Handler) {
	r.defaultHandler = h
}

// Resolve resolves the handler for a node.
func (r *Registry) Resolve(node *dot.Node) Handler {
	// 1. Explicit type attribute
	if t := node.Attr("type", ""); t != "" {
		if h, ok := r.handlers[t]; ok {
			return h
		}
	}

	// 2. Shape-based resolution
	shape := node.Attr("shape", "box")
	if handlerType, ok := ShapeToType[shape]; ok {
		if h, ok := r.handlers[handlerType]; ok {
			return h
		}
	}

	// 3. Default
	return r.defaultHandler
}
