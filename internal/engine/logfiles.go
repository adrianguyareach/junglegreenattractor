package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
)

const (
	dirPermissions  os.FileMode = 0755
	filePermissions os.FileMode = 0644
)

func writeManifest(logsRoot string, graph *dot.Graph) error {
	manifest := map[string]any{
		"name":       graph.Name,
		"goal":       graph.GraphAttr("goal", ""),
		"label":      graph.GraphAttr("label", ""),
		"started_at": time.Now().UTC().Format(time.RFC3339),
		"node_count": len(graph.Nodes),
		"edge_count": len(graph.Edges),
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return os.WriteFile(filepath.Join(logsRoot, "manifest.json"), data, filePermissions)
}

// WriteStatusFile persists a stage outcome to disk as status.json.
func WriteStatusFile(stageDir string, outcome *Outcome) error {
	data, err := json.MarshalIndent(outcome, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}
	return os.WriteFile(filepath.Join(stageDir, "status.json"), data, filePermissions)
}
