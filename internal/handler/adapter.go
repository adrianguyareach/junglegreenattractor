package handler

import (
	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/engine"
)

// RegistryAdapter adapts handler.Registry to engine.HandlerResolver.
type RegistryAdapter struct {
	Reg *Registry
}

func (a *RegistryAdapter) Resolve(node *dot.Node) engine.NodeHandler {
	h := a.Reg.Resolve(node)
	if h == nil {
		return nil
	}
	return &handlerBridge{h: h}
}

type handlerBridge struct {
	h Handler
}

func (b *handlerBridge) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return b.h.Execute(node, ctx, graph, logsRoot)
}
