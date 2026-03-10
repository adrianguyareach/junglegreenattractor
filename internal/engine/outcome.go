package engine

import "encoding/json"

type StageStatus string

const (
	StatusSuccess        StageStatus = "success"
	StatusPartialSuccess StageStatus = "partial_success"
	StatusRetry          StageStatus = "retry"
	StatusFail           StageStatus = "fail"
	StatusSkipped        StageStatus = "skipped"
)

// Outcome is the result of executing a node handler.
type Outcome struct {
	Status           StageStatus       `json:"outcome"`
	PreferredLabel   string            `json:"preferred_next_label,omitempty"`
	SuggestedNextIDs []string          `json:"suggested_next_ids,omitempty"`
	ContextUpdates   map[string]string `json:"context_updates,omitempty"`
	Notes            string            `json:"notes,omitempty"`
	FailureReason    string            `json:"failure_reason,omitempty"`
}

func (o *Outcome) MarshalJSON() ([]byte, error) {
	type Alias Outcome
	return json.Marshal((*Alias)(o))
}
