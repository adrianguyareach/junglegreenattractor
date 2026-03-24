package cli

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed usage.md
var usageMarkdown string

const usageProgramPlaceholder = "{{PROGRAM}}"

const version = "0.1.0"

func printUsage(progName string) {
	fmt.Print(usageText(progName))
}

func usageText(progName string) string {
	body := extractUsageFencedBlock(usageMarkdown)
	out := strings.ReplaceAll(body, usageProgramPlaceholder, progName)
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}

// extractUsageFencedBlock returns the first fenced ``` ... ``` body from usage.md, or the full file if none.
func extractUsageFencedBlock(md string) string {
	start := strings.Index(md, "```")
	if start < 0 {
		return strings.TrimSpace(md)
	}
	start += 3
	if nl := strings.IndexByte(md[start:], '\n'); nl >= 0 {
		start += nl + 1
	}
	end := strings.Index(md[start:], "```")
	if end < 0 {
		return strings.TrimSpace(md[start:])
	}
	return strings.TrimSpace(md[start : start+end])
}
