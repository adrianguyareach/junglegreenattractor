package handler

import "github.com/adrianguyareach/junglegreenattractor/internal/interviewer"

// BuildDefaultRegistry creates a handler registry with all built-in handlers
// wired to the given backend and interviewer.
func BuildDefaultRegistry(backend CodergenBackend, iv interviewer.Interviewer) *Registry {
	reg := NewRegistry()

	codergen := &CodergenHandler{Backend: backend}
	reg.SetDefault(codergen)

	reg.Register("start", &StartHandler{})
	reg.Register("exit", &ExitHandler{})
	reg.Register("codergen", codergen)
	reg.Register("conditional", &ConditionalHandler{})
	reg.Register("wait.human", &WaitForHumanHandler{Interviewer: iv})
	reg.Register("parallel", &ParallelHandler{})
	reg.Register("parallel.fan_in", &FanInHandler{})
	reg.Register("tool", &ToolHandler{})
	reg.Register("stack.manager_loop", &ManagerLoopHandler{})

	return reg
}
