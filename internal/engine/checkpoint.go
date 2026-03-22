package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Checkpoint is a serializable snapshot of pipeline execution state.
type Checkpoint struct {
	Timestamp      time.Time         `json:"timestamp"`
	CurrentNode    string            `json:"current_node"`
	CompletedNodes []string          `json:"completed_nodes"`
	NodeRetries    map[string]int    `json:"node_retries"`
	ContextValues  map[string]string `json:"context"`
	Logs           []string          `json:"logs"`
}

func NewCheckpoint(ctx *Context, currentNode string, completedNodes []string, nodeRetries map[string]int) *Checkpoint {
	return &Checkpoint{
		Timestamp:      time.Now().UTC(),
		CurrentNode:    currentNode,
		CompletedNodes: completedNodes,
		NodeRetries:    nodeRetries,
		ContextValues:  ctx.Snapshot(),
		Logs:           ctx.Logs(),
	}
}

func (cp *Checkpoint) Save(logsRoot string) error {
	path := filepath.Join(logsRoot, "checkpoint.json")
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}
	return os.WriteFile(path, data, filePermissions)
}

func LoadCheckpoint(logsRoot string) (*Checkpoint, error) {
	path := filepath.Join(logsRoot, "checkpoint.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint: %w", err)
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("unmarshal checkpoint: %w", err)
	}
	return &cp, nil
}
