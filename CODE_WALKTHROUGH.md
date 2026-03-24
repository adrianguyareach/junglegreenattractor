# Jungle Green Attractor Code Walkthrough

This document is a comprehensive teaching guide for understanding this repository as a practical implementation of the [Attractor specification](https://github.com/strongdm/attractor/blob/main/attractor-spec.md).

It is written for a developer who wants to go from:

- "I have heard of Attractor, but I do not really understand it"
- to "I can trace execution through this repository"
- to "I can modify or re-implement the system confidently"

## Table of Contents

[1. How To Read This Guide](#1-how-to-read-this-guide)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[1.1 Spec Item](#11-spec-item)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[1.2 Explanation](#12-explanation)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[1.3 Implementation in This Repo](#13-implementation-in-this-repo)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[1.4 Code Breakdown](#14-code-breakdown)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[1.5 Status](#15-status)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[1.6 Keeping this walkthrough in sync with the code](#16-keeping-this-walkthrough-in-sync-with-the-code)<br/>
[2. High-Level Overview](#2-high-level-overview)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[2.1 What is Attractor?](#21-what-is-attractor)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[2.2 What problem does Attractor solve?](#22-what-problem-does-attractor-solve)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[2.3 Key Attractor concepts](#23-key-attractor-concepts)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[2.3.1 Pipeline](#231-pipeline)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[2.3.2 Node](#232-node)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[2.3.3 Edge](#233-edge)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[2.3.4 Context](#234-context)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[2.3.5 Outcome](#235-outcome)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[2.3.6 Handler](#236-handler)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[2.3.7 Checkpoint and logs](#237-checkpoint-and-logs)<br/>
[3. Execution Flow (Top Down)](#3-execution-flow-top--down)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[3.1 Entry points in `cmd`](#31-entry-points-in-cmd)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[3.2 Top-down control-flow diagram](#32-top-down-control-flow-diagram)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[3.3 CLI routing](#33-cli-routing)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[3.4 What happens in `runCmd`](#34-what-happens-in-runcmd)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[3.5 How the engine takes over](#35-how-the-engine-takes-over)<br/>
[4. Specification -> Implementation Mapping (Critical)](#4-specification--implementation-mapping-critical)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.1 Spec Section 1: Overview and Goals](#41-spec-section-1-overview-and-goals)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.2 Spec Section 2: DOT DSL Schema](#42-spec-section-2-dot-dsl-schema)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.3 Spec Section 3: Pipeline Execution Engine](#43-spec-section-3-pipeline-execution-engine)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.4 Spec Section 4: Node Handlers](#44-spec-section-4-node-handlers)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.5 Spec Section 5: State and Context](#45-spec-section-5-state-and-context)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.6 Spec Section 6: Human-in-the-Loop (Interviewer Pattern)](#46-spec-section-6-human-in-the-loop-interviewer-pattern)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.7 Spec Section 7: Validation and Linting](#47-spec-section-7-validation-and-linting)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.8 Spec Section 8: Model Stylesheet](#48-spec-section-8-model-stylesheet)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.9 Spec Section 9: Transforms and Extensibility](#49-spec-section-9-transforms-and-extensibility)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.10 Spec Section 10: Condition Expression Language](#410-spec-section-10-condition-expression-language)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.11 Spec Section 11: Definition of Done](#411-spec-section-11-definition-of-done)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[4.12 Spec Appendices](#412-spec-appendices)<br/>
[5. Key Components Deep Dive](#5-key-components-deep-dive)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[5.1 `internal/cli`](#51-internalcli)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[5.2 `internal/dot`](#52-internaldot)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[5.3 `internal/transform`](#53-internaltransform)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[5.4 `internal/validate`](#54-internalvalidate)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[5.5 `internal/handler`](#55-internalhandler)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[5.6 `internal/engine`](#56-internalengine)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[5.7 `internal/interviewer`](#57-internalinterviewer)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[5.8 `internal/event`](#58-internalevent)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[5.9 `internal/stylesheet`](#59-internalstylesheet)<br/>
[6. End-to-End Example](#6-end-to-end-example)<br/>
[7. Design Insights](#7-design-insights)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[7.1 Why this architecture was chosen](#71-why-this-architecture-was-chosen)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[7.2 Strengths](#72-strengths)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[7.3 Weaknesses and limitations](#73-weaknesses-and-limitations)<br/>
&nbsp;&nbsp;&nbsp;&nbsp;[7.4 How I would extend it](#74-how-i-would-extend-it)<br/>
[8. Re-Implementing Attractor From Scratch: The Minimum Blueprint](#8-re-implementing-attractor-from-scratch-the-minimum-blueprint)<br/>
[9. Final Takeaways](#9-final-takeaways)<br/>
[10. Suggested Next Reading Order Inside This Repo](#10-suggested-next-reading-order-inside-this-repo)<br/>
[11. Source References](#11-source-references)

---

## 1. How To Read This Guide

This walkthrough has three goals:

1. Explain **what the Attractor spec is trying to achieve**.
2. Show **how this repository realizes those ideas in Go**.
3. Be honest about **what is fully implemented, partially implemented, or not yet implemented**.

Throughout this document, each spec section is handled using this pattern:

### 1.1 Spec Item

> The Attractor spec concept, section title, or rule being implemented.

### 1.2 Explanation

- What it means in plain English
- Why it exists
- What problem it solves

### 1.3 Implementation in This Repo

- Exact file and function
- A code snippet

### 1.4 Code Breakdown

- A line-by-line or idea-by-idea explanation of the important lines

### 1.5 Status

- `Implemented`
- `Partially implemented`
- `Not implemented in this repository`

That status is important. This repository is a strong CLI-first Attractor-style runner, but it does **not** implement every optional or future-facing part of the upstream specification.

### 1.6 Keeping this walkthrough in sync with the code

Whenever the repository’s structure or behavior changes (new packages, renamed files, new CLI commands, engine or handler changes), **update this document in the same change** so paths, responsibilities, and reading order stay accurate. Treat `CODE_WALKTHROUGH.md` as part of the change, not an afterthought.

---

## 2. High-Level Overview

### 2.1 What is Attractor?

Attractor is a specification for describing and executing **AI workflows as graphs**.

Instead of hard-coding a workflow in regular imperative code, you define a pipeline in Graphviz DOT:

- nodes represent work
- edges represent control flow
- node attributes configure behavior
- edge attributes control routing

At runtime, an engine:

1. parses the DOT graph,
2. validates it,
3. executes handlers node-by-node,
4. records state,
5. and chooses the next edge using deterministic routing rules.

In short:

```text
DOT file + attributes + handlers = executable AI workflow
```

### 2.2 What problem does Attractor solve?

Attractor solves a coordination problem:

- AI workflows often mix different kinds of steps:
  - generate code
  - ask a human for approval
  - run a tool command
  - retry a step
  - branch conditionally
  - fan out work
- if you implement that directly in regular code, the flow becomes hard to inspect and hard to change
- a graph-based workflow makes the process:
  - visible
  - declarative
  - easier to validate
  - easier to extend

This repository applies that idea to a **CLI pipeline runner in Go**.

### 2.3 Key Attractor concepts

#### 2.3.1 Pipeline

A pipeline is one directed graph declared in a `.dot` file.

#### 2.3.2 Node

A node represents one stage of work:

- start
- exit
- codergen / LLM step
- human review
- conditional branch
- tool execution
- parallel fan-out
- fan-in

#### 2.3.3 Edge

An edge connects one node to another and may include:

- a `label`
- a `condition`
- a `weight`
- a `loop_restart` flag

#### 2.3.4 Context

Context is the mutable key-value state shared across the whole pipeline run.

#### 2.3.5 Outcome

Each node returns an `Outcome` saying:

- whether it succeeded or failed
- what notes to record
- what context updates to apply
- which label or next node is preferred

#### 2.3.6 Handler

A handler is the concrete Go implementation that executes a node.

#### 2.3.7 Checkpoint and logs

The runner persists:

- `manifest.json`
- `checkpoint.json`
- per-stage `status.json`
- prompt / response artifacts for codergen stages

That makes runs inspectable and, conceptually, resumable.

---

## 3. Execution Flow (Top → Down)

This section traces execution from the command entrypoint all the way down into the core engine.

### 3.1 Entry points in `cmd`

Both binaries are extremely small:

- `cmd/jga/main.go`
- `cmd/junglegreenattractor/main.go`

They both delegate immediately to the shared CLI package.

**Implementation**

File: `cmd/jga/main.go`  
Function: `main`

```go
package main

import "github.com/adrianguyareach/junglegreenattractor/internal/cli"

func main() {
	cli.Main()
}
```

**Why this exists**

- it gives the project two equivalent executables:
  - `jga`
  - `junglegreenattractor`
- all real logic stays in `internal/cli`
- the command packages stay trivial

### 3.2 Top-down control-flow diagram

```text
cmd/jga/main.go
        |
        v
internal/cli.Main()
        |
        +--> parse subcommand (run / validate / inspect / list / graph)
        |
        +--> for run:
                |
                v
            dot.Parse()
                |
                v
            transform.ApplyAll()
                |
                v
            validate.ValidateOrRaise()
                |
                v
            handler.BuildDefaultRegistry()
                |
                v
            event.NewEmitter()
                |
                v
            engine.NewRunner(...).Run()
                |
                +--> execute handler for current node
                +--> persist status/checkpoint
                +--> select next edge
                +--> repeat until exit
```

### 3.3 CLI routing

The CLI package is the true application interface layer.

**Implementation**

File: `internal/cli/cli.go`  
Function: `Main`

```go
func Main() {
	progName := filepath.Base(os.Args[0])
	if len(os.Args) < 2 {
		printUsage(progName)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd(progName, os.Args[2:])
	case "validate":
		validateCmd(progName, os.Args[2:])
	case "inspect":
		inspectCmd(progName, os.Args[2:])
	case "list":
		listCmd(progName, os.Args[2:])
	case "graph":
		graphCmd(progName, os.Args[2:])
	case "version":
		fmt.Println(progName, version)
	case "help", "--help", "-h":
		printUsage(progName)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage(progName)
		os.Exit(1)
	}
}
```

**Code Breakdown**

- `filepath.Base(os.Args[0])` lets the CLI print either `jga` or `junglegreenattractor`
- the `switch` defines the public CLI surface
- `run` is the full parse → transform → validate → execute path
- `validate` stops before execution
- `inspect`, `list`, and `graph` are operational/observability commands for users

### 3.4 What happens in `runCmd`

The `run` command performs the application bootstrap for a pipeline execution.

**Implementation**

File: `internal/cli/run.go`  
Function: `runCmd`

```go
func runCmd(progName string, args []string) {
	// parse args ...
	// read DOT source ...
	// parse
	graph, err := dot.Parse(string(source))

	transforms := []transform.Transform{
		&transform.CustomVariableExpansion{Vars: varMap},
		&transform.VariableExpansion{},
		&transform.StylesheetApplication{},
	}
	transform.ApplyAll(graph, transforms)

	diags, err := validate.ValidateOrRaise(graph)

	var iv interviewer.Interviewer
	if *autoApprove {
		iv = interviewer.NewAutoApproveInterviewer()
	} else {
		iv = interviewer.NewConsoleInterviewer()
	}

	reg := handler.BuildDefaultRegistry(backend, iv)
	resolver := &handler.RegistryAdapter{Reg: reg}

	emitter := event.NewEmitter()
	// CLI subscribes to execution events here

	config := engine.Config{
		LogsRoot: logsRoot,
		Vars:     varMap,
	}

	runner := engine.NewRunner(graph, config, resolver, emitter)
	outcome, err := runner.Run()
	_ = diags
	_ = outcome
}
```

**Why this exists**

This function is the composition root for a run:

- it owns command-line parsing
- it loads the graph
- it prepares the transforms
- it validates upfront
- it wires handlers and interviewer
- it creates the engine
- it starts execution

That separation is good architecture: the CLI composes; the engine executes.

### 3.5 How the engine takes over

Once `engine.NewRunner(...).Run()` is called, the CLI stops being the center of control.

**Implementation**

File: `internal/engine/engine.go`  
Functions: `Run`, `runLoop`

```go
func (r *Runner) Run() (*Outcome, error) {
	ctx := NewContext()
	mirrorGraphAttributes(r.Graph, ctx)

	if err := os.MkdirAll(r.Config.LogsRoot, dirPermissions); err != nil {
		return nil, fmt.Errorf("create logs root: %w", err)
	}
	if err := writeManifest(r.Config.LogsRoot, r.Graph); err != nil {
		ctx.AppendLog(fmt.Sprintf("WARNING: failed to write manifest: %v", err))
	}

	r.Emitter.Emit(event.Event{
		Kind:    event.PipelineStarted,
		Message: r.Graph.GraphAttr("label", r.Graph.Name),
		Data:    map[string]any{"goal": r.Graph.GraphAttr("goal", "")},
	})

	startNode := findStartNode(r.Graph)
	if startNode == nil {
		return nil, fmt.Errorf("no start node found in graph")
	}

	return r.runLoop(ctx, startNode)
}
```

```go
func (r *Runner) runLoop(ctx *Context, startNode *dot.Node) (*Outcome, error) {
	var completedNodes []string
	nodeOutcomes := make(map[string]*Outcome)
	nodeRetries := make(map[string]int)
	currentNode := startNode
	var lastOutcome *Outcome
	stepIndex := 0

	for {
		if isTerminal(currentNode) {
			if failed := r.handleGoalGates(currentNode, nodeOutcomes); failed != nil {
				return failed, nil
			}
			break
		}

		stepIndex++
		stageDir := filepath.Join(r.Config.LogsRoot, fmt.Sprintf("%03d_%s", stepIndex, currentNode.ID))

		outcome, err := r.executeStage(currentNode, ctx, stageDir, nodeRetries)
		if err != nil {
			return nil, err
		}

		completedNodes = append(completedNodes, currentNode.ID)
		nodeOutcomes[currentNode.ID] = outcome
		lastOutcome = outcome

		r.recordOutcome(ctx, currentNode, outcome, stageDir, completedNodes, nodeRetries)

		nextNode, done, err := r.advance(currentNode, outcome, ctx)
		if err != nil {
			return outcome, err
		}
		if done {
			break
		}
		currentNode = nextNode
	}

	return lastOutcome, nil
}
```

**Code Breakdown**

- `NewContext()` creates shared mutable run state
- `mirrorGraphAttributes()` copies graph-level attributes into context so conditions and handlers can read them
- `writeManifest()` creates the run metadata file
- `findStartNode()` locates the first real execution node
- `runLoop()` is the pipeline machine:
  - stop if terminal
  - create a stage directory
  - execute the current handler
  - persist status and checkpoint
  - choose the next edge
  - repeat

This is the heart of the repository.

---

## 4. Specification → Implementation Mapping (Critical)

This is the main section of the walkthrough.

The upstream Attractor spec is broad. This repository implements a large, useful subset of it, especially:

- DOT parsing
- transforms
- validation
- execution
- node handlers
- context/outcome/checkpoints
- interviewer pattern
- event emission

### Thin areas (easy-to-spot gap map)

> [!WARNING]
> **THIN AREA = implemented partially or not yet implemented.**  
> Use this list as your implementation backlog while reading the mapping below.

| Area | Current state | How to implement next |
|---|---|---|
| True parallel execution | `Partially implemented` | In `ParallelHandler`, spawn branch workers with `sync.WaitGroup` + bounded worker pool; give each branch an isolated child context; aggregate branch outcomes in `FanInHandler`; define deterministic failure/merge policy (fail-fast vs collect-all). |
| Pipeline composition | `Not implemented` | Promote `ManagerLoopHandler` from stub to real orchestrator: load child `.dot`, run child `engine.Runner`, map child outcome back to parent outcome/context, and guard recursion depth with explicit limits. |
| HTTP server mode | `Not implemented` | Add `cmd/jga-server` and `internal/server`: `POST /runs`, `GET /runs/:id`, `GET /runs/:id/events`; execute runs in background goroutines, persist run index, and stream events via SSE/WebSocket. |
| Artifact-store abstraction | `Partially implemented` | Introduce `ArtifactStore` interface (`Write`, `Read`, `List`, `Exists`); keep filesystem as default implementation; inject store into engine/handlers so S3/GCS backends can be added without runtime changes. |
| Tool-call hooks | `Not implemented` | Add before/after hook interfaces around `ToolHandler` execution: emit pre-exec policy event, allow deny/mutate command, capture post-exec telemetry (duration, exit code, output size), and record to event stream. |
| Advanced context-fidelity behaviors | `Partially implemented` | Define fidelity policies (`full`, `compact`, `summary:*`) in a dedicated policy module; apply at checkpoint/artifact write time; add truncation/summarization strategy and validation tests for each level. |

The walkthrough below calls these gaps out explicitly in the relevant spec sections.

For implementation-level guidance on how to close these gaps, see `CODERGEN.md`, especially `9. Implementing the Thin Areas from CODE_WALKTHROUGH.md`.

---

## 4.1 Spec Section 1: Overview and Goals

### Spec Item

> `1. Overview and Goals`  
> `1.1 Problem Statement`

### Explanation

Attractor exists so that AI workflows can be declared as graphs instead of hand-written orchestration code.

The big idea is:

- the workflow should be inspectable
- the control flow should be deterministic
- execution behavior should come from node/edge metadata, not from deeply nested code

### Implementation in This Repo

File: `README.md`  
Repository description:

```markdown
A DOT-based pipeline runner that uses directed graphs (defined in Graphviz DOT syntax) to orchestrate multi-stage AI workflows.

Each node in the graph is an AI task (LLM call, human review, conditional branch, parallel fan-out, etc.) and edges define the flow between them.
```

### Code Breakdown

- this repo is explicitly positioned as a DOT-based workflow engine
- it maps Attractor’s abstract language into a concrete Go CLI
- the implementation is not just "inspired by" Attractor; it is structured around the same core phases:
  - parse
  - transform
  - validate
  - execute

### Status

`Implemented`

---

### Spec Item

> `1.2 Why DOT Syntax`

### Explanation

The spec chooses DOT because it already has:

- a graph vocabulary
- a compact syntax
- a visual mental model
- an ecosystem for rendering and authoring

That means you can express complex control flow without inventing a custom workflow language from scratch.

### Implementation in This Repo

File: `internal/dot/parser.go`  
Function: `Parse`

```go
func Parse(source string) (*Graph, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, fmt.Errorf("lex error: %w", err)
	}
	p := &parser{tokens: tokens}
	return p.parseGraph()
}
```

### Code Breakdown

- `lex(source)` turns the DOT text into tokens
- `parser{tokens: tokens}` builds a parser over those tokens
- `parseGraph()` produces the typed in-memory AST used by the rest of the system

This is where DOT stops being text and starts being executable structure.

### Status

`Implemented`

---

### Spec Item

> `1.3 Design Principles`

### Explanation

The spec emphasizes:

- deterministic routing
- declarative workflow definition
- extensibility through handlers/transforms
- backend independence

### Implementation in This Repo

File: `internal/handler/registry.go`  
Function: `(*Registry).Resolve`

```go
func (r *Registry) Resolve(node *dot.Node) Handler {
	if t := node.Attr("type", ""); t != "" {
		if h, ok := r.handlers[t]; ok {
			return h
		}
	}

	shape := node.Attr("shape", "box")
	if handlerType, ok := ShapeToType[shape]; ok {
		if h, ok := r.handlers[handlerType]; ok {
			return h
		}
	}

	return r.defaultHandler
}
```

### Code Breakdown

- nodes do not hardcode behavior in the engine
- they resolve through metadata:
  - explicit `type`
  - otherwise `shape`
  - otherwise default handler
- this makes the system extensible without rewriting the core loop

### Status

`Implemented`

---

### Spec Item

> `1.4 Layering and LLM Backends`

### Explanation

The spec wants workflow orchestration to be independent of the specific LLM provider.

That means the engine should not know about OpenAI, Anthropic, Gemini, or any specific SDK. It should depend on an interface.

### Implementation in This Repo

File: `internal/handler/registry.go`  
Interface: `CodergenBackend`

```go
type CodergenBackend interface {
	Run(node *dot.Node, prompt string, ctx *engine.Context) (*engine.Outcome, error)
}
```

### Code Breakdown

- the backend receives:
  - the node
  - the final prompt
  - the shared context
- it returns an `Outcome`, not raw provider-specific output
- that keeps the engine and handlers provider-agnostic

### Status

`Implemented`

---

## 4.2 Spec Section 2: DOT DSL Schema

### Spec Item

> `2.1 Supported Subset`

### Explanation

The spec does not require a full Graphviz implementation. It requires a **constrained, executable subset**.

That is a smart choice:

- easier parsing
- easier validation
- more predictable semantics
- fewer ambiguous constructs

### Implementation in This Repo

File: `internal/dot/parser.go`  
Functions: `parseGraph`, `parseStatements`, `parseNodeOrEdge`

```go
func (p *parser) parseGraph() (*Graph, error) {
	if _, err := p.expect(tokDigraph); err != nil {
		return nil, fmt.Errorf("expected 'digraph': %w", err)
	}
	// ...
	g := NewGraph(name)
	p.nodeDefaults = make(map[string]string)
	p.edgeDefaults = make(map[string]string)

	if err := p.parseStatements(g, nil); err != nil {
		return nil, err
	}
	// ...
	return g, nil
}
```

### Code Breakdown

- the parser only accepts `digraph`
- it does not attempt to support the entire Graphviz language
- it initializes graph/node/edge defaults explicitly
- it parses only the statement categories the runtime actually understands

### Status

`Implemented`

---

### Spec Item

> `2.2 BNF-Style Grammar`

### Explanation

The spec defines a grammar so that:

- different implementations parse the same source consistently
- users know what constructs are legal
- validation has a stable AST to work from

### Implementation in This Repo

File: `internal/dot/parser.go`  
Function: `parseAttrBlock`

```go
func (p *parser) parseAttrBlock() (map[string]string, error) {
	if _, err := p.expect(tokLBrack); err != nil {
		return nil, err
	}

	attrs := make(map[string]string)

	for p.cur().kind != tokRBrack && p.cur().kind != tokEOF {
		for p.cur().kind == tokComma || p.cur().kind == tokSemicolon {
			p.next()
		}
		if p.cur().kind == tokRBrack {
			break
		}

		key := p.cur().val
		p.next()
		if p.cur().kind != tokEquals {
			return nil, fmt.Errorf("line %d: expected '=' after attribute key %q", p.cur().line, key)
		}
		p.next()

		val := p.cur().val
		p.next()
		attrs[key] = val
	}

	if _, err := p.expect(tokRBrack); err != nil {
		return nil, err
	}
	return attrs, nil
}
```

### Code Breakdown

- expects `[` then repeated `key=value` entries
- tolerates commas and semicolons between attributes
- fails fast on malformed syntax
- produces a simple `map[string]string` that the rest of the system can consume

### Status

`Implemented`

---

### Spec Item

> `2.3 Key Constraints`

### Explanation

The spec constrains the language so execution is predictable:

- one graph per file
- directed edges only
- executable semantics attached to node and edge attributes

### Implementation in This Repo

File: `internal/dot/parser.go`  
Function: `parseGraph`

```go
if _, err := p.expect(tokDigraph); err != nil {
	return nil, fmt.Errorf("expected 'digraph': %w", err)
}
```

### Code Breakdown

- a file must begin with `digraph`
- the parser is intentionally opinionated
- this keeps the DSL aligned with Attractor’s execution model

### Status

`Implemented`

---

### Spec Item

> `2.4 Value Types`

### Explanation

The spec uses simple attribute values so that the language is easy to parse and easy to serialize.

This repository stores attribute values as strings and interprets them later.

### Implementation in This Repo

File: `internal/dot/ast.go`  
Types: `Graph`, `Node`, `Edge`

```go
type Graph struct {
	Name      string
	Attrs     map[string]string
	Nodes     map[string]*Node
	Edges     []*Edge
	NodeOrder []string
}

type Node struct {
	ID    string
	Attrs map[string]string
}

type Edge struct {
	From  string
	To    string
	Attrs map[string]string
}
```

### Code Breakdown

- all metadata is stored as stringly-typed attributes
- interpretation happens later:
  - `max_retries` is parsed into an integer
  - `timeout` is parsed into a duration
  - `goal_gate` is checked as `"true"`
- that design keeps parsing simple and pushes semantics into later layers

### Status

`Implemented`

---

### Spec Item

> `2.5 Graph-Level Attributes`

### Explanation

Graph-level attributes configure the run globally:

- goal
- label
- model stylesheet
- default retry policy
- retry target

### Implementation in This Repo

File: `internal/dot/ast.go`  
Method: `(*Graph).GraphAttr`

```go
func (g *Graph) GraphAttr(key, defaultVal string) string {
	if v, ok := g.Attrs[key]; ok {
		return v
	}
	return defaultVal
}
```

File: `internal/engine/graph.go`  
Function: `mirrorGraphAttributes`

```go
func mirrorGraphAttributes(graph *dot.Graph, ctx *Context) {
	for k, v := range graph.Attrs {
		ctx.Set("graph."+k, v)
	}
}
```

### Code Breakdown

- `GraphAttr()` is the read API for graph metadata
- `mirrorGraphAttributes()` copies graph metadata into runtime context
- that means graph values can influence:
  - prompt expansion
  - conditions
  - goal gate handling
  - retry behavior

### Status

`Implemented`

---

### Spec Item

> `2.6 Node Attributes`

### Explanation

Node attributes configure how one stage behaves:

- label
- shape
- type
- prompt
- retry controls
- goal gate
- timeout
- class

### Implementation in This Repo

File: `internal/dot/ast.go`  
Method: `(*Node).Attr`

```go
func (n *Node) Attr(key, defaultVal string) string {
	if v, ok := n.Attrs[key]; ok {
		return v
	}
	return defaultVal
}
```

### Code Breakdown

- node attributes are accessed uniformly via `Attr()`
- the engine and handlers do not reach into `Attrs` directly unless doing bulk transforms
- that gives the code a stable read abstraction over the raw parsed metadata

### Status

`Implemented`

---

### Spec Item

> `2.7 Edge Attributes`

### Explanation

Edge attributes control routing and priority.

This is one of the most important Attractor ideas: edges are not just arrows; they encode execution policy.

### Implementation in This Repo

File: `internal/dot/ast.go`  
Method: `(*Edge).Attr`

```go
func (e *Edge) Attr(key, defaultVal string) string {
	if v, ok := e.Attrs[key]; ok {
		return v
	}
	return defaultVal
}
```

### Status

`Implemented`

---

### Spec Item

> `2.8 Shape-to-Handler-Type Mapping`

### Explanation

The spec maps DOT shapes to runtime behavior. This is a key bridge between visualization and execution.

### Implementation in This Repo

File: `internal/handler/registry.go`  
Variable: `ShapeToType`

```go
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
```

### Code Breakdown

- every supported shape has a semantic meaning
- the registry uses this mapping to resolve which handler to execute
- this keeps the graph readable: the shape is not just visual decoration

### Status

`Implemented`

---

### Spec Item

> `2.9 Chained Edges`

### Explanation

Chained edges let authors write:

```dot
A -> B -> C
```

instead of writing two separate edges manually.

### Implementation in This Repo

File: `internal/dot/parser.go`  
Function: `parseNodeOrEdge`

```go
if p.cur().kind == tokArrow {
	nodeIDs := []string{firstID}
	for p.cur().kind == tokArrow {
		p.next()
		if p.cur().kind != tokIdent {
			return fmt.Errorf("line %d: expected node ID after '->'", p.cur().line)
		}
		nodeIDs = append(nodeIDs, p.cur().val)
		p.next()
	}

	for i := 0; i < len(nodeIDs)-1; i++ {
		edgeAttrs := copyMap(p.edgeDefaults)
		for k, v := range attrs {
			edgeAttrs[k] = v
		}
		g.AddEdge(nodeIDs[i], nodeIDs[i+1], edgeAttrs)
	}
}
```

### Code Breakdown

- collect all nodes in the chain
- optionally parse one shared attribute block
- expand the chain into pairwise edges:
  - `A -> B`
  - `B -> C`

### Status

`Implemented`

---

### Spec Item

> `2.10 Subgraphs`

### Explanation

Subgraphs are useful for grouping nodes and applying shared meaning, including class derivation.

### Implementation in This Repo

File: `internal/dot/parser.go`  
Function: `parseSubgraph`

```go
sub := &Subgraph{Name: subName, NodeDefaults: make(map[string]string), EdgeDefaults: make(map[string]string)}

savedNodeDefaults := copyMap(p.nodeDefaults)
savedEdgeDefaults := copyMap(p.edgeDefaults)

if err := p.parseStatements(g, sub); err != nil {
	return err
}

if sub.Label == "" {
	sub.Label = sub.NodeDefaults["label"]
}
if sub.Label == "" {
	sub.Label = subName
}
```

### Code Breakdown

- subgraphs have their own local defaults
- parser saves outer defaults, enters subgraph scope, then restores the previous defaults
- the implementation also derives a `class` from the subgraph label and applies it to nodes

### Status

`Implemented`

---

### Spec Item

> `2.11 Node and Edge Default Blocks`

### Explanation

Default blocks let authors set shared defaults once:

- `node [shape=box]`
- `edge [weight=1]`

### Implementation in This Repo

File: `internal/dot/parser.go`  
Functions: `parseStatements`, `parseNodeOrEdge`

```go
case tokNode:
	p.next()
	if p.cur().kind == tokLBrack {
		attrs, err := p.parseAttrBlock()
		// ...
		for k, v := range attrs {
			p.nodeDefaults[k] = v
		}
	}

case tokEdge:
	p.next()
	if p.cur().kind == tokLBrack {
		attrs, err := p.parseAttrBlock()
		// ...
		for k, v := range attrs {
			p.edgeDefaults[k] = v
		}
	}
```

### Code Breakdown

- `node [...]` updates parser-level node defaults
- `edge [...]` updates parser-level edge defaults
- later nodes and edges copy those defaults unless overridden

### Status

`Implemented`

---

### Spec Item

> `2.12 Class Attribute`

### Explanation

The `class` attribute supports stylesheet-style matching.

### Implementation in This Repo

File: `internal/dot/parser.go`  
Function: `deriveClass`

```go
func deriveClass(label string) string {
	label = strings.ToLower(label)
	label = strings.ReplaceAll(label, " ", "-")
	var b strings.Builder
	for _, r := range label {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
```

### Code Breakdown

- labels become normalized CSS-like class names
- that allows selectors such as `.critical-review`
- this is used during stylesheet application

### Status

`Implemented`

---

### Spec Item

> `2.13 Minimal Examples`

### Explanation

The spec includes small examples so the mental model is concrete.

### Implementation in This Repo

File: `examples/gorestspec/init_rest_app.dot`

```dot
start -> scaffold_project -> write_platform -> write_domain -> write_migrations -> write_main -> write_tests -> validate -> check
check -> review     [label="Yes", condition="outcome=success"]
check -> write_main [label="No",  condition="outcome!=success"]
review -> exit      [label="[A] Approve"]
review -> validate  [label="[F] Fix issues"]
```

### Code Breakdown

- linear happy-path flow from `start` to `check`
- conditional branching based on the stage outcome
- human approval gate before exit
- fix loop that routes back to validation

### Status

`Implemented`

---

## 4.3 Spec Section 3: Pipeline Execution Engine

### Spec Item

> `3.1 Run Lifecycle`

### Explanation

The spec describes the lifecycle as:

1. initialize state
2. validate and prepare
3. execute stages
4. record state
5. finish cleanly

### Implementation in This Repo

File: `internal/engine/engine.go`  
Function: `Run`

```go
func (r *Runner) Run() (*Outcome, error) {
	ctx := NewContext()
	mirrorGraphAttributes(r.Graph, ctx)

	if err := os.MkdirAll(r.Config.LogsRoot, dirPermissions); err != nil {
		return nil, fmt.Errorf("create logs root: %w", err)
	}
	if err := writeManifest(r.Config.LogsRoot, r.Graph); err != nil {
		ctx.AppendLog(fmt.Sprintf("WARNING: failed to write manifest: %v", err))
	}

	r.Emitter.Emit(event.Event{
		Kind:    event.PipelineStarted,
		Message: r.Graph.GraphAttr("label", r.Graph.Name),
		Data:    map[string]any{"goal": r.Graph.GraphAttr("goal", "")},
	})

	startNode := findStartNode(r.Graph)
	if startNode == nil {
		return nil, fmt.Errorf("no start node found in graph")
	}

	return r.runLoop(ctx, startNode)
}
```

### Code Breakdown

- initialize context
- mirror graph metadata into context
- create logs directory
- persist manifest
- emit lifecycle event
- locate start node
- enter the execution loop

### Status

`Implemented`

---

### Spec Item

> `3.2 Core Execution Loop`

### Explanation

This is the most important part of Attractor. The engine repeatedly:

- executes the current node
- records the outcome
- decides the next node

### Implementation in This Repo

File: `internal/engine/engine.go`  
Function: `runLoop`

```go
for {
	if isTerminal(currentNode) {
		if failed := r.handleGoalGates(currentNode, nodeOutcomes); failed != nil {
			return failed, nil
		}
		break
	}

	stepIndex++
	stageDir := filepath.Join(r.Config.LogsRoot, fmt.Sprintf("%03d_%s", stepIndex, currentNode.ID))

	outcome, err := r.executeStage(currentNode, ctx, stageDir, nodeRetries)
	if err != nil {
		return nil, err
	}

	completedNodes = append(completedNodes, currentNode.ID)
	nodeOutcomes[currentNode.ID] = outcome
	lastOutcome = outcome

	r.recordOutcome(ctx, currentNode, outcome, stageDir, completedNodes, nodeRetries)

	nextNode, done, err := r.advance(currentNode, outcome, ctx)
	if err != nil {
		return outcome, err
	}
	if done {
		break
	}
	currentNode = nextNode
}
```

### Code Breakdown

- `isTerminal()` checks whether execution should stop
- `stageDir` creates ordered stage folders like `006_write_main`
- `executeStage()` delegates to the proper handler
- `recordOutcome()` persists status and checkpoint
- `advance()` applies the routing algorithm

### Status

`Implemented`

---

### Spec Item

> `3.3 Edge Selection Algorithm`

### Explanation

The spec requires deterministic edge selection after a node finishes.

That matters because a workflow engine must never be ambiguous about "what happens next?"

### Implementation in This Repo

File: `internal/engine/edge.go`  
Function: `selectEdge`

```go
func selectEdge(node *dot.Node, outcome *Outcome, ctx *Context, graph *dot.Graph) *dot.Edge {
	edges := graph.OutgoingEdges(node.ID)
	if len(edges) == 0 {
		return nil
	}

	if e := matchByCondition(edges, outcome, ctx); e != nil {
		return e
	}
	if e := matchByPreferredLabel(edges, outcome); e != nil {
		return e
	}
	if e := matchBySuggestedIDs(edges, outcome); e != nil {
		return e
	}
	return matchByWeightOrLexical(edges)
}
```

### Code Breakdown

- step 1: match edges whose `condition` evaluates true
- step 2: use `PreferredLabel` if the handler provided one
- step 3: use `SuggestedNextIDs` if provided
- step 4/5: fall back to weights and lexical ordering

This closely mirrors the upstream spec and is one of the cleanest parts of the implementation.

### Status

`Implemented`

---

### Spec Item

> `3.4 Goal Gate Enforcement`

### Explanation

A goal gate says:

"The pipeline may not exit unless this stage succeeded."

This prevents the graph from reaching `exit` while an important quality gate failed.

### Implementation in This Repo

File: `internal/engine/graph.go`  
Function: `checkGoalGates`

```go
func checkGoalGates(graph *dot.Graph, outcomes map[string]*Outcome) (bool, *dot.Node) {
	for nodeID, outcome := range outcomes {
		node, ok := graph.Nodes[nodeID]
		if !ok {
			continue
		}
		if node.Attr("goal_gate", "") != "true" {
			continue
		}
		if outcome.Status != StatusSuccess && outcome.Status != StatusPartialSuccess {
			return false, node
		}
	}
	return true, nil
}
```

### Code Breakdown

- inspect every completed node
- keep only nodes marked `goal_gate=true`
- allow success or partial success
- if any goal gate failed, block exit

### Status

`Implemented`

---

### Spec Item

> `3.5 Retry Logic`

### Explanation

The spec requires bounded retries for transient failure or explicit retry outcomes.

### Implementation in This Repo

File: `internal/engine/engine.go`  
Function: `executeWithRetry`

```go
for attempt := 1; attempt <= policy.maxAttempts; attempt++ {
	outcome := r.safeExecute(handler, node, ctx, stageDir)

	if outcome.Status == StatusSuccess || outcome.Status == StatusPartialSuccess {
		nodeRetries[node.ID] = 0
		return outcome, nil
	}

	if outcome.Status == StatusFail {
		return outcome, nil
	}

	if attempt < policy.maxAttempts {
		nodeRetries[node.ID]++
		r.sleepWithRetryEvent(node.ID, attempt, policy)
		continue
	}

	if node.Attr("allow_partial", "") == "true" {
		return &Outcome{Status: StatusPartialSuccess, Notes: "retries exhausted, partial accepted"}, nil
	}
	return &Outcome{Status: StatusFail, FailureReason: "max retries exceeded"}, nil
}
```

### Code Breakdown

- execute the handler
- return immediately on success
- return immediately on explicit fail
- retry only when the outcome is effectively retriable
- stop when the retry budget is exhausted
- optionally downgrade to `partial_success`

### Status

`Implemented`

---

### Spec Item

> `3.6 Retry Policy`

### Explanation

The spec requires retry configuration to be explicit and bounded.

### Implementation in This Repo

File: `internal/engine/retry.go`  
Functions: `buildRetryPolicy`, `delayForAttempt`

```go
func buildRetryPolicy(node *dot.Node, graph *dot.Graph) retryPolicy {
	maxRetries := 0
	if v := node.Attr("max_retries", ""); v != "" {
		maxRetries, _ = strconv.Atoi(v)
	}
	if maxRetries == 0 {
		if v := graph.GraphAttr("default_max_retry", ""); v != "" {
			maxRetries, _ = strconv.Atoi(v)
		}
	}

	return retryPolicy{
		maxAttempts:    maxRetries + 1,
		initialDelayMs: defaultInitialDelayMs,
		backoffFactor:  defaultBackoffFactor,
		maxDelayMs:     defaultMaxDelayMs,
		jitter:         true,
	}
}
```

### Code Breakdown

- node-specific retries override the graph default
- `maxAttempts` is `retries + 1` because the first execution is not a retry
- exponential backoff is built into `delayForAttempt`
- there is a hard max delay

### Status

`Implemented`

---

### Spec Item

> `3.7 Failure Routing`

### Explanation

The spec wants failures to still participate in graph routing, not just crash the program blindly.

### Implementation in This Repo

File: `internal/engine/engine.go`  
Function: `advance`

```go
func (r *Runner) advance(node *dot.Node, outcome *Outcome, ctx *Context) (*dot.Node, bool, error) {
	nextEdge := selectEdge(node, outcome, ctx, r.Graph)
	if nextEdge == nil {
		if outcome.Status == StatusFail {
			return nil, false, fmt.Errorf("stage %q failed with no outgoing fail edge", node.ID)
		}
		return nil, true, nil
	}
	// ...
}
```

### Code Breakdown

- the engine always tries to route through edges first
- if no edge matches:
  - successful stage => done
  - failed stage => error
- so failure can be modeled in the graph, but missing failure handling is treated as a problem

### Status

`Implemented`

---

### Spec Item

> `3.8 Concurrency Model`

### Explanation

The spec discusses parallel branches and concurrency behavior.

### Implementation in This Repo

File: `internal/handler/parallel.go`

```go
// ParallelHandler fans out execution to multiple branches. In the current
// implementation, branches are simulated sequentially.
type ParallelHandler struct{}

func (h *ParallelHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	edges := graph.OutgoingEdges(node.ID)
	if len(edges) == 0 {
		return &engine.Outcome{Status: engine.StatusSuccess, Notes: "No branches to execute"}, nil
	}

	for _, edge := range edges {
		ctx.Set("parallel.branch."+edge.To, "pending")
	}

	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  fmt.Sprintf("Parallel fan-out to %d branches (sequential simulation)", len(edges)),
	}, nil
}
```

### Code Breakdown

- the API surface for parallelism exists
- the handler records branch metadata in context
- but the implementation is explicit that this is **sequential simulation**, not true concurrency

### Status

`Partially implemented`

---

## 4.4 Spec Section 4: Node Handlers

### Spec Item

> `4.1 Handler Interface`

### Explanation

A handler is the unit of work that executes one node.

The engine should not know handler details; it should only know the contract.

### Implementation in This Repo

File: `internal/handler/registry.go`

```go
type Handler interface {
	Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error)
}
```

### Code Breakdown

- every handler gets the current node
- every handler can inspect shared context
- every handler can inspect the whole graph if necessary
- every handler knows where stage artifacts should be written
- every handler returns a normalized `Outcome`

### Status

`Implemented`

---

### Spec Item

> `4.2 Handler Registry`

### Explanation

The registry decouples:

- graph node metadata
- concrete Go handler instances

### Implementation in This Repo

File: `internal/handler/default_registry.go`

```go
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
```

### Code Breakdown

- registry wiring happens once per run
- dependencies are injected at registration time
- custom handler types can be added without changing engine code

### Status

`Implemented`

---

### Spec Item

> `4.3 Start Handler`

### Explanation

The start node exists to mark entry. It is usually a no-op.

### Implementation in This Repo

File: `internal/handler/passthrough.go`

```go
type StartHandler struct{}

func (h *StartHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{Status: engine.StatusSuccess, Notes: "Pipeline started"}, nil
}
```

### Status

`Implemented`

---

### Spec Item

> `4.4 Exit Handler`

### Explanation

The exit node marks successful termination. It is also usually a no-op.

### Implementation in This Repo

File: `internal/handler/passthrough.go`

```go
type ExitHandler struct{}

func (h *ExitHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{Status: engine.StatusSuccess, Notes: "Pipeline exit reached"}, nil
}
```

### Status

`Implemented`

---

### Spec Item

> `4.5 Codergen Handler (LLM Task)`

### Explanation

This is the main "AI work" handler:

- build prompt
- call backend
- persist artifacts
- return normalized outcome

### Implementation in This Repo

File: `internal/handler/codergen.go`  
Function: `(*CodergenHandler).Execute`

```go
func (h *CodergenHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	prompt := resolvePrompt(node, graph)
	stageDir := logsRoot

	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return nil, fmt.Errorf("create stage dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(stageDir, "prompt.md"), []byte(prompt), 0644); err != nil {
		return nil, fmt.Errorf("write prompt: %w", err)
	}

	responseText, result := h.callBackend(node, prompt, ctx, stageDir)
	if result != nil {
		return result, nil
	}

	if err := os.WriteFile(filepath.Join(stageDir, "response.md"), []byte(responseText), 0644); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}

	outcome := &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Stage completed: " + node.ID,
		ContextUpdates: map[string]string{
			"last_stage":    node.ID,
			"last_response": truncateResponse(responseText),
		},
	}

	writeStatus(stageDir, outcome)
	return outcome, nil
}
```

### Code Breakdown

- `resolvePrompt()` expands the node prompt and `$goal`
- prompt is persisted as `prompt.md`
- backend is invoked through an abstraction
- response is persisted as `response.md`
- the result is normalized into `Outcome`
- `status.json` is written for the stage

### Status

`Implemented`

---

### Spec Item

> `4.5.1 CodergenBackend Interface`

### Explanation

The backend abstraction isolates the orchestration layer from the model provider.

### Implementation in This Repo

File: `internal/handler/registry.go`

```go
type CodergenBackend interface {
	Run(node *dot.Node, prompt string, ctx *engine.Context) (*engine.Outcome, error)
}
```

### Status

`Implemented`

---

### Spec Item

> `4.6 Wait For Human Handler`

### Explanation

This handler pauses the graph and asks a human which edge to take.

### Implementation in This Repo

File: `internal/handler/human.go`  
Function: `(*WaitForHumanHandler).Execute`

```go
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
```

### Code Breakdown

- outgoing edges become user-visible options
- the interviewer abstraction is used to ask the question
- timeout and skip are explicit states
- a successful choice returns `SuggestedNextIDs`, which the edge selector can honor

### Status

`Implemented`

---

### Spec Item

> `4.7 Conditional Handler`

### Explanation

A conditional node usually does not perform work itself. It simply exists as a semantic routing point.

### Implementation in This Repo

File: `internal/handler/passthrough.go`

```go
type ConditionalHandler struct{}

func (h *ConditionalHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Conditional node evaluated: " + node.ID,
	}, nil
}
```

### Code Breakdown

- handler does almost nothing
- the real decision happens in `engine.selectEdge()`
- this is a nice example of separating control-flow semantics from execution semantics

### Status

`Implemented`

---

### Spec Item

> `4.8 Parallel Handler`

### Explanation

The spec describes fan-out across multiple branches.

### Implementation in This Repo

File: `internal/handler/parallel.go`

```go
func (h *ParallelHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	edges := graph.OutgoingEdges(node.ID)
	if len(edges) == 0 {
		return &engine.Outcome{Status: engine.StatusSuccess, Notes: "No branches to execute"}, nil
	}

	for _, edge := range edges {
		ctx.Set("parallel.branch."+edge.To, "pending")
	}

	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  fmt.Sprintf("Parallel fan-out to %d branches (sequential simulation)", len(edges)),
		ContextUpdates: map[string]string{
			"parallel.branch_count":  fmt.Sprintf("%d", len(edges)),
			"parallel.success_count": fmt.Sprintf("%d", len(edges)),
			"parallel.failure_count": "0",
		},
	}, nil
}
```

### Code Breakdown

- the handler acknowledges parallel branches
- it records metadata in context
- but it does not actually launch independent concurrent executions

### Status

`Partially implemented`

---

### Spec Item

> `4.9 Fan-In Handler`

### Explanation

Fan-in should merge branch results back into one logical continuation point.

### Implementation in This Repo

File: `internal/handler/parallel.go`

```go
type FanInHandler struct{}

func (h *FanInHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Fan-in consolidation complete",
		ContextUpdates: map[string]string{
			"parallel.fan_in.completed": "true",
		},
	}, nil
}
```

### Code Breakdown

- the semantic surface exists
- there is no real branch-result merge logic yet
- current implementation is a placeholder success marker

### Status

`Partially implemented`

---

### Spec Item

> `4.10 Tool Handler`

### Explanation

The tool handler lets a graph execute shell commands as stages.

### Implementation in This Repo

File: `internal/handler/tool.go`

```go
func (h *ToolHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	command := node.Attr("tool_command", "")
	if command == "" {
		return &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: "No tool_command specified",
		}, nil
	}

	timeout := parseToolTimeout(node)
	output, err := runWithTimeout(command, timeout)
	if err != nil {
		return &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: err.Error(),
		}, nil
	}

	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Tool completed: " + command,
		ContextUpdates: map[string]string{
			"tool.output": string(output),
		},
	}, nil
}
```

### Code Breakdown

- reads `tool_command` from the node
- parses timeout from node metadata
- executes through `sh -c`
- captures combined output
- exposes output in context so later stages can use it

### Status

`Implemented`

---

### Spec Item

> `4.11 Manager Loop Handler`

### Explanation

The spec discusses a manager/supervisor loop for child workflows.

### Implementation in This Repo

File: `internal/handler/passthrough.go`

```go
type ManagerLoopHandler struct{}

func (h *ManagerLoopHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Manager loop handler (stub): " + node.ID,
	}, nil
}
```

### Code Breakdown

- the handler type exists
- it is currently a stub
- there is no child-pipeline supervision logic yet

### Status

`Partially implemented`

---

### Spec Item

> `4.12 Custom Handlers`

### Explanation

The spec expects new handlers to be addable without changing the engine.

### Implementation in This Repo

File: `internal/handler/registry.go`

```go
func (r *Registry) Register(typeStr string, h Handler) {
	r.handlers[typeStr] = h
}
```

### Code Breakdown

- extensibility is registry-based
- adding a new handler means:
  - implement `Handler`
  - register it under a type string
  - reference it from a node’s `type`

### Status

`Implemented`

---

## 4.5 Spec Section 5: State and Context

### Spec Item

> `5.1 Context`

### Explanation

Context is shared runtime state across stages.

It is how one stage communicates useful facts to the next stage without mutating the graph definition itself.

### Implementation in This Repo

File: `internal/engine/context.go`

```go
type Context struct {
	mu     sync.RWMutex
	values map[string]string
	logs   []string
}

func (c *Context) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

func (c *Context) Get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.values[key]
}
```

### Code Breakdown

- `sync.RWMutex` protects concurrent access
- `values` stores the shared key-value data
- `logs` stores non-fatal warnings and audit messages
- helper methods expose:
  - set
  - get
  - snapshot
  - clone
  - apply updates

### Status

`Implemented`

---

### Spec Item

> `5.2 Outcome`

### Explanation

Outcome is the normalized result shape for every stage.

This is critical because it gives the engine one common language regardless of handler type.

### Implementation in This Repo

File: `internal/engine/outcome.go`

```go
type Outcome struct {
	Status           StageStatus       `json:"outcome"`
	PreferredLabel   string            `json:"preferred_next_label,omitempty"`
	SuggestedNextIDs []string          `json:"suggested_next_ids,omitempty"`
	ContextUpdates   map[string]string `json:"context_updates,omitempty"`
	Notes            string            `json:"notes,omitempty"`
	FailureReason    string            `json:"failure_reason,omitempty"`
}
```

### Code Breakdown

- `Status` drives retry and routing
- `PreferredLabel` helps edge selection
- `SuggestedNextIDs` gives explicit routing hints
- `ContextUpdates` mutates runtime context
- `Notes` captures human-readable explanation
- `FailureReason` surfaces why a stage failed

### Status

`Implemented`

---

### Spec Item

> `5.3 Checkpoint`

### Explanation

Checkpoints allow a run to persist enough state to inspect or resume later.

### Implementation in This Repo

File: `internal/engine/checkpoint.go`

```go
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
```

### Code Breakdown

- checkpoint captures both control state and data state
- this includes:
  - current node
  - completed nodes
  - retry counts
  - context snapshot
  - warning log entries

### Status

`Implemented`

---

### Spec Item

> `5.4 Context Fidelity`

### Explanation

The spec discusses different levels of context fidelity and how much data should be carried forward.

### Implementation in This Repo

File: `internal/validate/rules.go`

```go
func checkFidelityValid(graph *dot.Graph) []Diagnostic {
	validFidelity := map[string]bool{
		"full": true, "truncate": true, "compact": true,
		"summary:low": true, "summary:medium": true, "summary:high": true,
		"": true,
	}
	// ...
}
```

### Code Breakdown

- the repo validates allowed fidelity values
- but it does **not** implement a full runtime fidelity system that changes artifact retention or context projection

### Status

`Partially implemented`

---

### Spec Item

> `5.5 Artifact Store`

### Explanation

The spec describes a generalized artifact store abstraction.

### Implementation in This Repo

Closest implementation: filesystem-backed stage artifacts in the engine and codergen handler.

File: `internal/engine/logfiles.go`

```go
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
```

### Code Breakdown

- artifacts are stored directly on disk
- there is no separate `ArtifactStore` interface yet
- but the practical effect exists:
  - manifest
  - checkpoint
  - prompt
  - response
  - status

### Status

`Partially implemented`

---

### Spec Item

> `5.6 Run Directory Structure`

### Explanation

The spec expects a stable run directory layout so tools and humans can inspect runs consistently.

### Implementation in This Repo

File: `internal/engine/engine.go`  
Function: `runLoop`

```go
stepIndex++
stageDir := filepath.Join(r.Config.LogsRoot, fmt.Sprintf("%03d_%s", stepIndex, currentNode.ID))
if err := os.MkdirAll(stageDir, dirPermissions); err != nil {
	return nil, fmt.Errorf("create stage dir: %w", err)
}
```

### Code Breakdown

- each stage gets an ordered folder:
  - `001_start`
  - `006_write_main`
- this makes runs easy to inspect visually
- the `inspect` CLI command relies on this structure

### Status

`Implemented`

---

## 4.6 Spec Section 6: Human-in-the-Loop (Interviewer Pattern)

### Spec Item

> `6.1 Interviewer Interface`

### Explanation

The engine should not read from stdin directly. Human interaction should go through an abstraction.

### Implementation in This Repo

File: `internal/interviewer/types.go`

```go
type Interviewer interface {
	Ask(q Question) Answer
	Inform(message, stage string)
}
```

### Code Breakdown

- `Ask()` gathers a decision
- `Inform()` is a side-channel for one-way messages
- this abstraction enables:
  - console mode
  - auto-approve mode
  - queue-based testing

### Status

`Implemented`

---

### Spec Item

> `6.2 Question Model`

### Explanation

Questions need structure so the same handler logic can drive different UIs.

### Implementation in This Repo

File: `internal/interviewer/types.go`

```go
type Question struct {
	Text    string
	Type    QuestionType
	Options []Option
	Stage   string
}
```

### Status

`Implemented`

---

### Spec Item

> `6.3 Answer Model`

### Explanation

Answers must represent both normal user choices and special states such as timeout or skip.

### Implementation in This Repo

File: `internal/interviewer/types.go`

```go
type AnswerValue string

const (
	AnswerYes     AnswerValue = "yes"
	AnswerNo      AnswerValue = "no"
	AnswerSkipped AnswerValue = "skipped"
	AnswerTimeout AnswerValue = "timeout"
)

type Answer struct {
	Value          AnswerValue
	SelectedOption *Option
	Text           string
}
```

### Status

`Implemented`

---

### Spec Item

> `6.4 Built-In Interviewer Implementations`

### Explanation

The spec wants multiple interviewer implementations for different environments.

### Implementation in This Repo

File: `internal/interviewer/console.go`

```go
func (c *ConsoleInterviewer) Ask(q Question) Answer {
	fmt.Printf("\n[?] %s\n", q.Text)
	switch q.Type {
	case MultipleChoice:
		for _, opt := range q.Options {
			fmt.Printf("  [%s] %s\n", opt.Key, opt.Label)
		}
		fmt.Print("Select: ")
		line := c.readLine()
		// ...
	}
	// ...
}
```

File: `internal/interviewer/auto.go`

```go
func (a *AutoApproveInterviewer) Ask(q Question) Answer {
	switch q.Type {
	case YesNo, Confirmation:
		return Answer{Value: AnswerYes}
	case MultipleChoice:
		if len(q.Options) > 0 {
			o := q.Options[0]
			return Answer{Value: AnswerValue(o.Key), SelectedOption: &o}
		}
	}
	return Answer{Value: AnswerValue("auto-approved"), Text: "auto-approved"}
}
```

### Code Breakdown

- `ConsoleInterviewer` is real interactive mode
- `AutoApproveInterviewer` is CI / unattended mode
- `QueueInterviewer` exists too, mainly useful for tests

### Status

`Implemented`

---

### Spec Item

> `6.5 Timeout Handling`

### Explanation

Human waiting cannot be infinite in every environment. Timeout needs a defined behavior.

### Implementation in This Repo

File: `internal/handler/human.go`

```go
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
```

### Code Breakdown

- if a default choice exists, the workflow can continue
- otherwise the stage returns `retry`
- that cleanly integrates with engine retry logic

### Status

`Implemented`

---

## 4.7 Spec Section 7: Validation and Linting

### Spec Item

> `7.1 Diagnostic Model`

### Explanation

Validation findings must be structured, not just free-form strings.

### Implementation in This Repo

File: `internal/validate/validate.go`

```go
type Diagnostic struct {
	Rule     string
	Severity Severity
	Message  string
	NodeID   string
	Edge     [2]string
	Fix      string
}
```

### Code Breakdown

- `Rule` tells you which validator fired
- `Severity` separates blocking errors from warnings
- `NodeID` and `Edge` point to the faulty graph element
- `Fix` helps the user recover

### Status

`Implemented`

---

### Spec Item

> `7.2 Built-In Lint Rules`

### Explanation

The spec expects a baseline rule set that catches invalid pipelines before execution.

### Implementation in This Repo

File: `internal/validate/validate.go`

```go
func Validate(graph *dot.Graph) []Diagnostic {
	rules := []func(*dot.Graph) []Diagnostic{
		checkStartNode,
		checkTerminalNode,
		checkEdgeTargets,
		checkStartNoIncoming,
		checkExitNoOutgoing,
		checkReachability,
		checkPromptOnLLMNodes,
		checkRetryTargets,
		checkGoalGateHasRetry,
		checkConditionSyntax,
		checkFidelityValid,
	}
	// ...
}
```

### Code Breakdown

- validation is assembled as an ordered rule list
- each rule is a focused function
- diagnostics are aggregated into one report

### Status

`Implemented`

---

### Spec Item

> `7.3 Validation API`

### Explanation

Validation should be usable in two ways:

- get all diagnostics
- fail fast if any blocking errors exist

### Implementation in This Repo

File: `internal/validate/validate.go`

```go
func ValidateOrRaise(graph *dot.Graph) ([]Diagnostic, error) {
	diags := Validate(graph)
	var errors []string
	for _, d := range diags {
		if d.Severity == SeverityError {
			errors = append(errors, d.String())
		}
	}
	if len(errors) > 0 {
		return diags, fmt.Errorf("validation failed:\n  %s", strings.Join(errors, "\n  "))
	}
	return diags, nil
}
```

### Status

`Implemented`

---

### Spec Item

> `7.4 Custom Lint Rules`

### Explanation

The spec discusses extensible validation beyond built-in rules.

### Implementation in This Repo

Closest implementation: rule list inside `Validate`.

```go
rules := []func(*dot.Graph) []Diagnostic{
	checkStartNode,
	checkTerminalNode,
	// ...
}
```

### Code Breakdown

- the structure is friendly to extension
- but there is no exported registration API for external rule injection yet

### Status

`Partially implemented`

---

## 4.8 Spec Section 8: Model Stylesheet

### Spec Item

> `8.1 Overview`

### Explanation

The stylesheet lets a graph centralize model/provider choices rather than repeating them on every node.

### Implementation in This Repo

File: `internal/transform/transforms.go`

```go
type StylesheetApplication struct{}

func (t *StylesheetApplication) Apply(graph *dot.Graph) {
	raw := graph.GraphAttr("model_stylesheet", "")
	if raw == "" {
		return
	}
	rules := stylesheet.Parse(raw)
	stylesheet.Apply(graph, rules)
}
```

### Status

`Implemented`

---

### Spec Item

> `8.2 Stylesheet Grammar`

### Explanation

The spec defines a small CSS-like rule format.

### Implementation in This Repo

File: `internal/stylesheet/stylesheet.go`

```go
func Parse(source string) []Rule {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}

	var rules []Rule
	remaining := source

	for remaining != "" {
		remaining = strings.TrimSpace(remaining)
		braceIdx := strings.Index(remaining, "{")
		selector := strings.TrimSpace(remaining[:braceIdx])
		remaining = remaining[braceIdx+1:]
		closeBrace := strings.Index(remaining, "}")
		body := strings.TrimSpace(remaining[:closeBrace])
		remaining = remaining[closeBrace+1:]

		props := parseDeclarations(body)
		// ...
	}
	return rules
}
```

### Code Breakdown

- repeatedly parse `selector { declarations }`
- selectors and properties become a typed `Rule`
- grammar is intentionally simple and purpose-built

### Status

`Implemented`

---

### Spec Item

> `8.3 Selectors and Specificity`

### Explanation

The spec allows:

- `*`
- `.class`
- `#id`
- and selector precedence

### Implementation in This Repo

File: `internal/stylesheet/stylesheet.go`

```go
switch {
case strings.HasPrefix(selector, "#"):
	specificity = 2
case strings.HasPrefix(selector, "."):
	specificity = 1
case selector == "*":
	specificity = 0
}
```

```go
func matches(selector string, node *dot.Node) bool {
	switch {
	case selector == "*":
		return true
	case strings.HasPrefix(selector, "#"):
		return node.ID == selector[1:]
	case strings.HasPrefix(selector, "."):
		className := selector[1:]
		nodeClasses := node.Attr("class", "")
		// ...
	default:
		return node.Attr("shape", "box") == selector
	}
}
```

### Code Breakdown

- specificity is encoded numerically
- selector matching supports wildcard, node ID, class, and shape fallback

### Status

`Implemented`

---

### Spec Item

> `8.4 Recognized Properties`

### Explanation

The spec expects stylesheet properties such as model/provider-related settings.

### Implementation in This Repo

Closest implementation:

```go
for key, val := range rule.Properties {
	if _, exists := node.Attrs[key]; !exists {
		node.Attrs[key] = val
	}
}
```

### Code Breakdown

- this implementation is intentionally generic
- it does not hardcode a small whitelist of recognized properties
- instead it copies arbitrary attributes onto nodes

That is flexible, but slightly looser than a stricter spec-driven property contract.

### Status

`Partially implemented`

---

### Spec Item

> `8.5 Application Order`

### Explanation

Stylesheet order and specificity determine which rule wins.

### Implementation in This Repo

File: `internal/stylesheet/stylesheet.go`

```go
for spec := 0; spec <= 2; spec++ {
	for _, rule := range rules {
		if rule.Specificity != spec {
			continue
		}
		for _, node := range graph.Nodes {
			if matches(rule.Selector, node) {
				for key, val := range rule.Properties {
					node.Attrs[key] = val
				}
			}
		}
	}
}
```

### Code Breakdown

- rules are reapplied in specificity order
- later, more specific rules override earlier, broader ones

### Status

`Implemented`

---

### Spec Item

> `8.6 Example`

### Explanation

The spec includes example stylesheet declarations to make the model concrete.

### Implementation in This Repo

File: `README.md`

```dot
graph [
    model_stylesheet="
        * { llm_model: claude-sonnet-4-5; llm_provider: anthropic; }
        .code { llm_model: claude-opus-4-6; }
        #critical_review { llm_model: gpt-5.2; llm_provider: openai; }
    "
]
```

### Status

`Implemented`

---

## 4.9 Spec Section 9: Transforms and Extensibility

### Spec Item

> `9.1 AST Transforms`

### Explanation

Transforms let you modify the parsed graph before validation or execution.

### Implementation in This Repo

File: `internal/transform/transforms.go`

```go
type Transform interface {
	Apply(graph *dot.Graph)
}
```

### Code Breakdown

- transform API is intentionally tiny
- a transform is just a graph mutation step
- that makes pre-execution rewrites easy to reason about

### Status

`Implemented`

---

### Spec Item

> `9.2 Built-In Transforms`

### Explanation

The spec expects some built-in transforms to exist out of the box.

### Implementation in This Repo

File: `internal/transform/transforms.go`

```go
transforms := []transform.Transform{
	&transform.CustomVariableExpansion{Vars: varMap},
	&transform.VariableExpansion{},
	&transform.StylesheetApplication{},
}
transform.ApplyAll(graph, transforms)
```

### Code Breakdown

- custom variables first
- built-in `$goal` expansion second
- stylesheet application third

That ordering matters because later phases rely on a fully materialized graph.

### Status

`Implemented`

---

### Spec Item

> `9.3 Custom Transforms`

### Explanation

Users should be able to add their own graph rewriting steps.

### Implementation in This Repo

File: `internal/transform/transforms.go`

```go
func ApplyAll(graph *dot.Graph, transforms []Transform) {
	for _, t := range transforms {
		t.Apply(graph)
	}
}
```

### Code Breakdown

- custom transforms are supported structurally
- the CLI currently hardcodes the built-in list
- but the engine itself is transform-agnostic

### Status

`Partially implemented`

---

### Spec Item

> `9.4 Pipeline Composition`

### Explanation

The spec discusses composing or nesting pipelines.

### Implementation in This Repo

Closest implementation: manager-loop stub only.

```go
type ManagerLoopHandler struct{}

func (h *ManagerLoopHandler) Execute(node *dot.Node, ctx *engine.Context, graph *dot.Graph, logsRoot string) (*engine.Outcome, error) {
	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Manager loop handler (stub): " + node.ID,
	}, nil
}
```

### Code Breakdown

- there is no actual child-pipeline composition feature yet
- the type exists as a placeholder extension point

### Status

`Not implemented in this repository`

---

### Spec Item

> `9.5 HTTP Server Mode`

### Explanation

The spec discusses exposing pipeline execution through an HTTP server.

### Implementation in This Repo

Evidence that the current repo is CLI-only:

```go
package main

import "github.com/adrianguyareach/junglegreenattractor/internal/cli"

func main() {
	cli.Main()
}
```

### Code Breakdown

- both entrypoints call `cli.Main()`
- there is no HTTP server package, router, or API transport for pipeline execution

### Status

`Not implemented in this repository`

---

### Spec Item

> `9.6 Observability and Events`

### Explanation

The spec expects structured runtime events so execution can be observed externally.

### Implementation in This Repo

File: `internal/event/event.go`

```go
type Event struct {
	Kind      Kind
	Timestamp time.Time
	NodeID    string
	Message   string
	Data      map[string]any
}

type Emitter struct {
	mu       sync.RWMutex
	handlers []Handler
}

func (e *Emitter) On(h Handler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = append(e.handlers, h)
}

func (e *Emitter) Emit(evt Event) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, h := range e.handlers {
		h(evt)
	}
}
```

### Code Breakdown

- events are typed and timestamped
- handlers can subscribe to events
- CLI output is implemented as an event subscriber

### Status

`Implemented`

---

### Spec Item

> `9.7 Tool Call Hooks`

### Explanation

The spec mentions hooks around tool calls for extra observability or policy control.

### Implementation in This Repo

Closest implementation: `ToolHandler` itself.

```go
func runWithTimeout(command string, timeout time.Duration) ([]byte, error) {
	cmd := exec.Command("sh", "-c", command)
	// ...
}
```

### Code Breakdown

- tool execution exists
- but there is no before/after hook API surrounding tool invocation

### Status

`Not implemented in this repository`

---

## 4.10 Spec Section 10: Condition Expression Language

### Spec Item

> `10.1 Overview`

### Explanation

Conditions let edges become executable guards instead of passive connections.

### Implementation in This Repo

File: `internal/engine/condition.go`

```go
func EvaluateCondition(condition string, outcome *Outcome, ctx *Context) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true
	}

	clauses := strings.Split(condition, "&&")
	for _, clause := range clauses {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}
		if !evaluateClause(clause, outcome, ctx) {
			return false
		}
	}
	return true
}
```

### Status

`Implemented`

---

### Spec Item

> `10.2 Grammar`

### Explanation

The grammar is intentionally tiny:

- equality
- inequality
- conjunction
- truthy bare keys

### Implementation in This Repo

File: `internal/engine/condition.go`

```go
func evaluateClause(clause string, outcome *Outcome, ctx *Context) bool {
	if idx := strings.Index(clause, "!="); idx >= 0 {
		key := strings.TrimSpace(clause[:idx])
		val := strings.TrimSpace(clause[idx+2:])
		return resolveKey(key, outcome, ctx) != val
	}
	if idx := strings.Index(clause, "="); idx >= 0 {
		key := strings.TrimSpace(clause[:idx])
		val := strings.TrimSpace(clause[idx+1:])
		return resolveKey(key, outcome, ctx) == val
	}
	return resolveKey(strings.TrimSpace(clause), outcome, ctx) != ""
}
```

### Code Breakdown

- `!=` has priority over `=`
- bare keys are treated as truthy checks
- the evaluator is deliberately small and deterministic

### Status

`Implemented`

---

### Spec Item

> `10.3 Semantics`

### Explanation

A condition is evaluated against runtime data:

- outcome
- preferred label
- context values

### Implementation in This Repo

File: `internal/engine/condition.go`

```go
func resolveKey(key string, outcome *Outcome, ctx *Context) string {
	switch key {
	case "outcome":
		if outcome != nil {
			return string(outcome.Status)
		}
		return ""
	case "preferred_label":
		if outcome != nil {
			return outcome.PreferredLabel
		}
		return ""
	}

	if strings.HasPrefix(key, "context.") {
		ctxKey := key
		if v := ctx.Get(ctxKey); v != "" {
			return v
		}
		stripped := strings.TrimPrefix(key, "context.")
		return ctx.Get(stripped)
	}

	return ctx.Get(key)
}
```

### Code Breakdown

- special outcome fields are handled explicitly
- `context.foo` works
- plain `foo` falls back to the same context store

### Status

`Implemented`

---

### Spec Item

> `10.4 Variable Resolution`

### Explanation

The spec defines how names inside condition expressions map to runtime values.

### Implementation in This Repo

Same `resolveKey()` implementation as above.

### Status

`Implemented`

---

### Spec Item

> `10.5 Evaluation`

### Explanation

Evaluation should be deterministic and easy to audit.

### Implementation in This Repo

File: `internal/engine/condition.go`

```go
clauses := strings.Split(condition, "&&")
for _, clause := range clauses {
	clause = strings.TrimSpace(clause)
	if clause == "" {
		continue
	}
	if !evaluateClause(clause, outcome, ctx) {
		return false
	}
}
return true
```

### Code Breakdown

- `&&` is the only logical combinator implemented
- evaluation short-circuits on the first false clause
- simple grammar keeps runtime behavior transparent

### Status

`Implemented`

---

### Spec Item

> `10.6 Examples`

### Explanation

The spec includes examples like:

- `outcome=success`
- `outcome!=fail`
- `context.loop_state!=exhausted`

### Implementation in This Repo

File: `examples/gorestspec/init_rest_app.dot`

```dot
check -> review     [label="Yes", condition="outcome=success"]
check -> write_main [label="No",  condition="outcome!=success"]
```

### Status

`Implemented`

---

### Spec Item

> `10.7 Extended Operators (Future)`

### Explanation

The spec explicitly marks richer operators as future work.

### Implementation in This Repo

File: `internal/engine/condition.go`

```go
// Grammar: clause ( '&&' clause )*
// Clause:  key '=' value | key '!=' value
```

### Code Breakdown

- only the minimal operator set is implemented
- there is no `||`, regex, numeric comparison, or grouping

### Status

`Implemented as minimal subset`

---

## 4.11 Spec Section 11: Definition of Done

This section is best understood as a parity checklist.

### Spec Item

> `11.1 DOT Parsing`

### Implementation in This Repo

- parser exists
- tests exist in `internal/dot/parser_test.go`

### Status

`Implemented`

---

### Spec Item

> `11.2 Validation and Linting`

### Implementation in This Repo

- validator exists
- lint rules exist
- `validate` CLI command exists

### Status

`Implemented`

---

### Spec Item

> `11.3 Execution Engine`

### Implementation in This Repo

- engine loop exists in `internal/engine/engine.go`
- status files and checkpoints are persisted

### Status

`Implemented`

---

### Spec Item

> `11.4 Goal Gate Enforcement`

### Implementation in This Repo

- implemented via `checkGoalGates()` and `handleGoalGates()`

### Status

`Implemented`

---

### Spec Item

> `11.5 Retry Logic`

### Implementation in This Repo

- exponential retry exists
- partial success exists

### Status

`Implemented`

---

### Spec Item

> `11.6 Node Handlers`

### Implementation in This Repo

- all major built-in handlers are present
- some advanced ones are stubs or partial

### Status

`Partially implemented`

---

### Spec Item

> `11.7 State and Context`

### Implementation in This Repo

- context, outcome, checkpoint exist
- artifact store abstraction is not fully generalized

### Status

`Partially implemented`

---

### Spec Item

> `11.8 Human-in-the-Loop`

### Implementation in This Repo

- interviewer abstraction exists
- console and auto-approve modes exist

### Status

`Implemented`

---

### Spec Item

> `11.9 Condition Expressions`

### Implementation in This Repo

- minimal grammar fully implemented

### Status

`Implemented`

---

### Spec Item

> `11.10 Model Stylesheet`

### Implementation in This Repo

- parsing and application exist
- property validation is loose

### Status

`Partially implemented`

---

### Spec Item

> `11.11 Transforms and Extensibility`

### Implementation in This Repo

- transform API exists
- built-ins exist
- external registration model is limited

### Status

`Partially implemented`

---

### Spec Item

> `11.12 Cross-Feature Parity Matrix`

### Explanation

The spec compares supported features across implementations.

### Implementation in This Repo

This repository effectively falls into this parity profile:

- parsing: strong
- validation: strong
- execution: strong
- handlers: good, with some stubs
- concurrency: partial
- HTTP mode: absent
- composition: absent

### Status

`Partially implemented`

---

### Spec Item

> `11.13 Integration Smoke Test`

### Explanation

The spec wants an end-to-end smoke test proving the overall flow works.

### Implementation in This Repo

Closest implementation:

- example pipelines under `examples/gorestspec/`
- `go test ./...` covers parser and engine logic
- no single end-to-end integration test currently executes an example pipeline as a smoke test

### Status

`Partially implemented`

---

## 4.12 Spec Appendices

### Spec Item

> `Appendix A: Complete Attribute Reference`

### Explanation

The appendices centralize the contract for graph, node, and edge attributes.

### Implementation in This Repo

The practical implementation lives in:

- parser: stores raw attributes
- handlers: consume node attributes
- engine: consumes graph/edge attributes
- README: documents the supported subset

Relevant code:

```go
func (n *Node) Attr(key, defaultVal string) string { /* ... */ }
func (g *Graph) GraphAttr(key, defaultVal string) string { /* ... */ }
func (e *Edge) Attr(key, defaultVal string) string { /* ... */ }
```

### Status

`Implemented`

---

### Spec Item

> `Appendix B: Shape-to-Handler-Type Mapping`

### Implementation in This Repo

Already covered by `handler.ShapeToType`.

### Status

`Implemented`

---

### Spec Item

> `Appendix C: Status File Contract`

### Explanation

The spec wants stage status files to be machine-readable and stable.

### Implementation in This Repo

File: `internal/engine/logfiles.go`

```go
func WriteStatusFile(stageDir string, outcome *Outcome) error {
	data, err := json.MarshalIndent(outcome, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}
	return os.WriteFile(filepath.Join(stageDir, "status.json"), data, filePermissions)
}
```

### Code Breakdown

- status files are JSON
- they serialize the normalized `Outcome`
- this keeps artifact inspection simple and stable

### Status

`Implemented`

---

### Spec Item

> `Appendix D: Error Categories`

### Explanation

The spec describes error families such as parse, validation, handler, and runtime errors.

### Implementation in This Repo

Representative examples:

```go
return nil, fmt.Errorf("lex error: %w", err)
return nil, fmt.Errorf("validation failed:\n  %s", strings.Join(errors, "\n  "))
return &Outcome{Status: StatusFail, FailureReason: err.Error()}
return nil, fmt.Errorf("stage %q failed with no outgoing fail edge", node.ID)
```

### Code Breakdown

- errors are wrapped with context
- stage-level failures are encoded in `Outcome`
- structural failures are returned as Go errors

### Status

`Implemented`

---

## 5. Key Components Deep Dive

This section now switches from spec mapping to architectural understanding.

## 5.1 `internal/cli`

### Purpose

The CLI package is the interface layer. It is where users interact with the system.

### Package layout (one file per command / concern)

| File          | Role                                                                       |
| ------------- | -------------------------------------------------------------------------- |
| `cli.go`      | `Main()` — argument routing only                                           |
| `usage.go`    | Embedded `usage.md`, `printUsage` / `usageText`, `version`                 |
| `run.go`      | `runCmd`, `-var` flag type (`varFlags`), string helpers used by run output |
| `validate.go` | `validateCmd`                                                              |
| `inspect.go`  | `inspectCmd` and run-directory reporting                                   |
| `list.go`     | `listCmd`                                                                  |
| `graph.go`    | `graphCmd`                                                                 |

This keeps each subcommand isolated (SOLID-friendly) and avoids a single oversized `cli.go`.

### Responsibilities

- parse commands and flags
- read `.dot` files
- apply transforms
- validate pipelines
- wire dependencies
- print execution progress
- provide operational commands:
  - `inspect`
  - `list`
  - `graph`

### Key functions

- `Main()` (`cli.go`)
- `runCmd()` (`run.go`)
- `validateCmd()` (`validate.go`)
- `inspectCmd()` (`inspect.go`)
- `listCmd()` (`list.go`)
- `graphCmd()` (`graph.go`)

### Why it is designed this way

The CLI owns composition, not execution. That is clean architecture:

- the CLI decides which command to run
- the engine runs pipelines
- handlers perform work

## 5.2 `internal/dot`

### Purpose

This package turns DOT text into an executable graph AST.

### Responsibilities

- tokenizing
- parsing
- building graph/node/edge structures
- tracking defaults
- handling chained edges and subgraphs

### Key functions

- `Parse()`
- `parseGraph()`
- `parseStatements()`
- `parseNodeOrEdge()`
- `parseAttrBlock()`

### Interaction with other parts

- `transform` mutates the resulting graph
- `validate` checks the graph
- `engine` executes the graph

## 5.3 `internal/transform`

### Purpose

Transform the graph after parsing but before validation/execution.

### Responsibilities

- expand `$goal`
- expand custom CLI variables
- apply model stylesheet

### Why this layer matters

Without transforms, handlers and validation would need to reason about raw placeholders. Transforms simplify later phases by making the graph "ready to use".

## 5.4 `internal/validate`

### Purpose

Reject invalid graphs early.

### Responsibilities

- structural correctness
- routing sanity
- prompt expectations
- retry target checks
- fidelity validation

### Why this layer matters

It protects the engine from executing malformed workflows and gives users useful diagnostics before a run starts.

## 5.5 `internal/handler`

### Purpose

Implement actual node behavior.

### Responsibilities

- map shapes/types to runtime logic
- execute codergen, human, tool, and other node kinds
- return normalized outcomes

### Key design decision

The engine knows only `NodeHandler`. It never hardcodes business behavior for specific node kinds.

That makes handlers the main extension seam of the system.

## 5.6 `internal/engine`

### Purpose

Drive execution of the graph.

### Responsibilities

- manage context
- enforce lifecycle
- apply retries
- choose next edges
- persist status and checkpoints
- emit events

### Why this package is the system core

If `dot` is the parser and `handler` is the worker library, `engine` is the runtime kernel.

Everything important about execution semantics lives here:

- what is terminal
- what is success
- what triggers retry
- what counts as a goal gate failure
- how edge selection works

## 5.7 `internal/interviewer`

### Purpose

Abstract human interaction.

### Responsibilities

- define question/answer protocol
- provide console mode
- provide auto-approve mode
- provide test-friendly queue mode

### Why this matters

It keeps human decisions inside the workflow model instead of hardcoding them in the CLI or handlers.

## 5.8 `internal/event`

### Purpose

Provide structured lifecycle events.

### Responsibilities

- emit execution milestones
- support multiple subscribers
- decouple execution from presentation

### Why this matters

The CLI can print progress without the engine knowing anything about terminal output.

## 5.9 `internal/stylesheet`

### Purpose

Apply CSS-like rule sets to nodes.

### Responsibilities

- parse stylesheet rules
- compute selector specificity
- match selectors to nodes
- apply node attributes

### Why this matters

It lets the graph define model policy in one place instead of duplicating it across many nodes.

---

## 6. End-to-End Example

We will walk through `examples/gorestspec/init_rest_app.dot`.

### 6.1 Input

Command:

```bash
./jga run examples/gorestspec/init_rest_app.dot \
  -auto-approve \
  -var module_name="github.com/acme/api" \
  -var first_module="user"
```

### 6.2 Step 1: CLI reads and parses the graph

- `runCmd()` reads the file
- `dot.Parse()` builds the `Graph`

Important graph fragment:

```dot
start -> scaffold_project -> write_platform -> write_domain -> write_migrations -> write_main -> write_tests -> validate -> check
check -> review     [label="Yes", condition="outcome=success"]
check -> write_main [label="No",  condition="outcome!=success"]
review -> exit      [label="[A] Approve"]
review -> validate  [label="[F] Fix issues"]
```

### 6.3 Step 2: Transforms apply

- `$module_name` and `$first_module` expand through `CustomVariableExpansion`
- `$goal` expands through `VariableExpansion`
- stylesheet would apply here if present

### 6.4 Step 3: Validation runs

- start node exists
- exit node exists
- all targets exist
- graph is reachable
- `validate` node has a prompt

If validation fails, execution never begins.

### 6.5 Step 4: Engine initializes run state

- create `.jgattractorlogs/init_rest_app/`
- write `manifest.json`
- create a new `Context`
- emit `pipeline.started`

### 6.6 Step 5: Start node executes

`StartHandler` returns success immediately.

Stage directory:

```text
.jgattractorlogs/init_rest_app/001_start/
  status.json
```

### 6.7 Step 6: Codergen nodes execute in sequence

For `write_main`, the codergen handler:

1. resolves the prompt
2. writes `prompt.md`
3. simulates or invokes backend
4. writes `response.md`
5. writes `status.json`
6. returns success with context updates

Result:

```json
{
  "outcome": "success",
  "context_updates": {
    "last_response": "[Simulated] Response for stage: write_main",
    "last_stage": "write_main"
  },
  "notes": "Stage completed: write_main"
}
```

### 6.8 Step 7: Conditional routing happens at `check`

If the previous node’s outcome is success:

```dot
check -> review [condition="outcome=success"]
```

The engine evaluates the condition through `EvaluateCondition()` and takes that edge.

### 6.9 Step 8: Human approval occurs at `review`

Because the run uses `-auto-approve`, `AutoApproveInterviewer` selects the first option:

- `[A] Approve`

That sends execution to `exit`.

### 6.10 Step 9: Goal gate enforcement occurs before final exit

The `validate` node is marked `goal_gate=true`.

That means even if the graph reaches `exit`, the engine checks whether `validate` succeeded before allowing the pipeline to finish.

### 6.11 Step 10: Final output

The engine emits:

- `pipeline.completed`
- final `Outcome`
- final checkpoint on disk

And the CLI prints a summary.

### 6.12 Full conceptual trace

```text
read DOT
  -> parse AST
  -> apply transforms
  -> validate graph
  -> initialize runner
  -> execute start
  -> execute codergen stages
  -> evaluate conditional
  -> ask interviewer
  -> enforce goal gate
  -> exit
  -> persist artifacts
```

---

## 7. Design Insights

## 7.1 Why this architecture was chosen

This repository follows a layered design that matches the problem nicely:

- `dot` parses the workflow language
- `transform` prepares the graph
- `validate` protects execution
- `engine` owns runtime semantics
- `handler` owns node behavior
- `cli` owns user interaction

That is a strong decomposition because each package answers a different question:

- What did the user declare?
- Is it valid?
- What should happen next?
- How do we perform that step?
- How do we expose this to a human?

## 7.2 Strengths

### 1. Clear execution model

The `runLoop()` is easy to follow and closely mirrors the spec.

### 2. Strong extension seams

You can add:

- new handlers
- new transforms
- new CLI commands

without redesigning the whole system.

### 3. Good inspectability

The filesystem log structure plus `inspect` / `list` / `graph` makes runs easy to understand.

### 4. Clean handler abstraction

Every node kind returns the same `Outcome` shape, which dramatically simplifies orchestration.

## 7.3 Weaknesses and limitations

### 1. Parallel execution is mostly conceptual

The spec discusses real fan-out/fan-in behavior, but this repo only simulates it.

### 2. HTTP server mode is absent

This implementation is CLI-first only.

### 3. Some advanced spec areas are stubs

Especially:

- manager loop
- pipeline composition
- tool-call hooks
- richer fidelity behavior

### 4. Style system is flexible but loose

The stylesheet applies arbitrary properties without a tighter semantic contract.

## 7.4 How I would extend it

### To make it closer to the full Attractor spec

1. Add a real `ArtifactStore` interface and let filesystem be one implementation.
2. Implement true parallel branch execution with isolated child contexts.
3. Add a public validation rule registration API.
4. Add an HTTP server package for:
   - submit pipeline
   - inspect run
   - stream events
5. Implement a real manager-loop / child-pipeline execution model.
6. Add end-to-end smoke tests that run example pipelines in simulation mode.

### To make it better for production

1. Add structured logger abstraction in the engine.
2. Add resume support using `LoadCheckpoint()`.
3. Add run IDs separate from folder names.
4. Add explicit failure edge labels / routing conventions.
5. Add richer condition language operators only if needed.

---

## 8. Re-Implementing Attractor From Scratch: The Minimum Blueprint

If you wanted to rebuild this from scratch, implement in this order:

1. **AST**
   - `Graph`
   - `Node`
   - `Edge`

2. **Parser**
   - `digraph`
   - nodes
   - edges
   - attr blocks
   - chained edges

3. **Validation**
   - one start
   - one exit
   - reachable nodes
   - valid edge targets

4. **Outcome + Context**
   - `Outcome`
   - shared context map

5. **Handler interface**
   - start
   - exit
   - codergen
   - human

6. **Engine**
   - run loop
   - edge selection
   - retries
   - checkpoints

7. **Observability**
   - event emitter
   - status files
   - manifest

8. **Transforms**
   - variable expansion
   - stylesheet application

9. **CLI**
   - run
   - validate
   - inspect, list, graph

This repository is a very good reference for that build order because its architecture already follows that sequence.

---

## 9. Final Takeaways

By this point, you should understand the central Attractor idea:

```text
workflow definition is graph data
runtime behavior is handler + engine logic
routing is determined by edge semantics
state is persisted as structured artifacts
```

The most important files in this repository are:

- `internal/cli/cli.go` — CLI entry routing (`Main`)
- `internal/cli/run.go` — full `run` command: parse flags, transforms, validate, wire engine
- `internal/cli/usage.go` — help text and version string
- `internal/dot/parser.go` — turns DOT into executable structure
- `internal/transform/transforms.go` — prepares the graph
- `internal/validate/validate.go` and `rules.go` — protects execution
- `internal/engine/engine.go` — core runtime loop
- `internal/engine/edge.go` — routing semantics
- `internal/handler/*.go` — actual stage behavior
- `internal/interviewer/*.go` — human decision abstraction
- `internal/event/event.go` — observability

If you only study five things, study these in order:

1. `examples/gorestspec/init_rest_app.dot`
2. `internal/cli/cli.go` then `internal/cli/run.go` (routing vs. run composition)
3. `internal/dot/parser.go`
4. `internal/engine/engine.go`
5. `internal/engine/edge.go`

That sequence gives you the fastest path to understanding how this repo turns a DOT file into a real workflow run.

---

## 10. Suggested Next Reading Order Inside This Repo

If you want to continue learning by reading code in the IDE, use this order:

1. `examples/gorestspec/init_rest_app.dot`
2. `cmd/jga/main.go`
3. `internal/cli/cli.go` → `internal/cli/run.go` → `internal/cli/usage.go` → other `internal/cli/*.go` as needed
4. `internal/dot/ast.go`
5. `internal/dot/parser.go`
6. `internal/transform/transforms.go`
7. `internal/validate/validate.go`
8. `internal/validate/rules.go`
9. `internal/handler/registry.go`
10. `internal/handler/codergen.go`
11. `internal/handler/human.go`
12. `internal/engine/outcome.go`
13. `internal/engine/context.go`
14. `internal/engine/condition.go`
15. `internal/engine/edge.go`
16. `internal/engine/engine.go`
17. `internal/engine/checkpoint.go`
18. `internal/event/event.go`

That reading order goes from easiest mental model to deepest runtime details.

---

## 11. Source References

- [Attractor Specification](https://github.com/strongdm/attractor/blob/main/attractor-spec.md)
- [Jungle Green Attractor README](README.md)
- Example pipeline: `examples/gorestspec/init_rest_app.dot`
