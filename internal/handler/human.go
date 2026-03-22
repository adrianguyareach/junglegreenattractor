package handler

import (
	"regexp"
	"strings"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
	"github.com/adrianguyareach/junglegreenattractor/internal/engine"
	"github.com/adrianguyareach/junglegreenattractor/internal/interviewer"
)

// WaitForHumanHandler blocks until a human selects an option from the
// outgoing edges of the current node.
type WaitForHumanHandler struct {
	Interviewer interviewer.Interviewer
}

func (h *WaitForHumanHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	edges := graph.OutgoingEdges(node.ID)
	if len(edges) == 0 {
		return &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: "No outgoing edges for human gate",
		}, nil
	}

	choices := buildChoices(edges)
	question := interviewer.Question{
		Text:    node.Attr("label", "Select an option:"),
		Type:    interviewer.MultipleChoice,
		Options: choices,
		Stage:   node.ID,
	}

	answer := h.Interviewer.Ask(question)

	switch answer.Value {
	case interviewer.AnswerTimeout:
		return h.handleTimeout(node)
	case interviewer.AnswerSkipped:
		return &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: "human skipped interaction",
		}, nil
	}

	return resolveSelection(answer, choices, edges)
}

func buildChoices(edges []*dot.Edge) []interviewer.Option {
	choices := make([]interviewer.Option, 0, len(edges))
	for _, edge := range edges {
		label := edge.Attr("label", edge.To)
		key := parseAcceleratorKey(label)
		choices = append(choices, interviewer.Option{Key: key, Label: label})
	}
	return choices
}

func (h *WaitForHumanHandler) handleTimeout(node *dot.Node) (*engine.Outcome, error) {
	defaultChoice := node.Attr("human.default_choice", "")
	if defaultChoice != "" {
		return &engine.Outcome{
			Status:           engine.StatusSuccess,
			SuggestedNextIDs: []string{defaultChoice},
			ContextUpdates: map[string]string{
				"human.gate.selected": defaultChoice,
			},
		}, nil
	}
	return &engine.Outcome{
		Status:        engine.StatusRetry,
		FailureReason: "human gate timeout, no default",
	}, nil
}

func resolveSelection(answer interviewer.Answer, choices []interviewer.Option, edges []*dot.Edge) (*engine.Outcome, error) {
	selectedTo, selectedLabel := matchAnswerToEdge(answer, choices, edges)
	if selectedTo == "" && len(edges) > 0 {
		selectedTo = edges[0].To
		selectedLabel = choices[0].Label
	}

	return &engine.Outcome{
		Status:           engine.StatusSuccess,
		SuggestedNextIDs: []string{selectedTo},
		ContextUpdates: map[string]string{
			"human.gate.selected": string(answer.Value),
			"human.gate.label":    selectedLabel,
		},
	}, nil
}

func matchAnswerToEdge(answer interviewer.Answer, choices []interviewer.Option, edges []*dot.Edge) (string, string) {
	for i, opt := range choices {
		if answer.SelectedOption != nil && answer.SelectedOption.Key == opt.Key {
			return edges[i].To, opt.Label
		}
		if strings.EqualFold(string(answer.Value), opt.Key) {
			return edges[i].To, opt.Label
		}
	}
	return "", ""
}

var accelPattern = regexp.MustCompile(`^\[(\w)\]\s+|^(\w)\)\s+|^(\w)\s*-\s+`)

func parseAcceleratorKey(label string) string {
	m := accelPattern.FindStringSubmatch(label)
	if m != nil {
		for _, g := range m[1:] {
			if g != "" {
				return strings.ToUpper(g)
			}
		}
	}
	if len(label) > 0 {
		return strings.ToUpper(string(label[0]))
	}
	return ""
}
