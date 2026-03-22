The same commands apply when the binary is invoked as `jga` (substitute `jga` for `junglegreenattractor` in the block below when reading this file).

```
{{PROGRAM}} — Jungle Green Attractor: a DOT-based pipeline runner for AI workflows

Usage:
  {{PROGRAM}} run <pipeline.dot> [flags]    Run a pipeline
  {{PROGRAM}} validate <pipeline.dot>       Validate a pipeline without running
  {{PROGRAM}} inspect <run-path>            Inspect a completed pipeline run
  {{PROGRAM}} list [log-dir]                List pipeline runs
  {{PROGRAM}} graph <pipeline.dot>          Display graph structure
  {{PROGRAM}} version                       Print version
  {{PROGRAM}} help                          Show this help

Run Flags:
  -log <dir>            Log directory (default: .jgattractorlogs)
  -name <name>          Run name for the log folder (default: derived from .dot filename)
  -var key=value        Set pipeline variable (repeatable)
  -auto-approve         Auto-approve all human gates
  -simulate             Run in simulation mode (no LLM backend)

Build:
  go build -o junglegreenattractor ./cmd/junglegreenattractor/
  go build -o jga ./cmd/jga/

Examples:
  # Run a pipeline (logs default to .jgattractorlogs/<dot-basename>/)
  {{PROGRAM}} run pipeline.dot

  # Custom log directory
  {{PROGRAM}} run pipeline.dot -log logs/.jgattractorlogs

  # With variables and auto-approve (for CI/CD)
  {{PROGRAM}} run init_rest_app.dot -auto-approve \
    -var module_name="github.com/acme/api" \
    -var first_module="user"

  # Multiple variables
  {{PROGRAM}} run add_module.dot \
    -var module_name=product \
    -var entity_fields="ID string, Name string, Price int64"

  # Custom run name
  {{PROGRAM}} run full_feature.dot -name "feature-search-v2"

  # Validate only (no execution)
  {{PROGRAM}} validate pipeline.dot

  # Inspect a previous run
  {{PROGRAM}} inspect .jgattractorlogs/init_rest_app

  # List all pipeline runs
  {{PROGRAM}} list

  # Display graph nodes and edges
  {{PROGRAM}} graph examples/gorestspec/init_rest_app.dot
```
