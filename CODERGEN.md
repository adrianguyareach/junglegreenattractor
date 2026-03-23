# CODERGEN.md

This document is Part 2 of the repository walkthrough.

`CODE_WALKTHROUGH.md` explained how this repository currently implements the Attractor pipeline runner. This document explains how to build the **Coding Agent Loop** and the **Unified LLM integration layer** on top of that foundation, and how to integrate both into this repository cleanly.

The primary upstream references are:

- [Attractor Specification](https://github.com/strongdm/attractor/blob/main/attractor-spec.md)
- [Coding Agent Loop Specification](https://raw.githubusercontent.com/strongdm/attractor/main/coding-agent-loop-spec.md)

This guide is implementation-first. It is not just "what the spec says." It is:

- what you need to build,
- where it should live in this repo,
- how it should interact with the existing engine,
- and how to roll it out incrementally without breaking the current CLI runner.

---

## 1. Purpose of This Document

`CODE_WALKTHROUGH.md` answered:

- What is Attractor?
- How does this repository work today?
- How do the parser, transforms, validator, engine, handlers, and logs fit together?

This document answers the next set of questions:

- How do you build the **Coding Agent Loop** described in the coding-agent-loop spec?
- What does a **Unified LLM client** need to expose so the loop can be provider-agnostic?
- How do you integrate those layers into this codebase without fighting its current design?
- How do you use the resulting system from code and from the CLI?

This is intentionally **implementation-first**:

- we will define interfaces,
- propose package boundaries,
- describe control flow,
- show concrete Go skeletons,
- and map those pieces back to the current repository structure.

The most important thing to keep in mind:

> This repository already has a strong **pipeline execution engine**.  
> It does **not** yet have a full **coding agent loop** or a **unified multi-provider LLM client**.

So this document is a blueprint for the next layer.

### 🔗 Relation to `CODE_WALKTHROUGH.md`

This document extends:

- `CODE_WALKTHROUGH.md` section `2. Execution Flow (Top → Down)`
- `CODE_WALKTHROUGH.md` section `3.3 Spec Section 3: Pipeline Execution Engine`
- `CODE_WALKTHROUGH.md` section `4.5 internal/handler`
- `CODE_WALKTHROUGH.md` section `6.4 How I would extend it`

Those sections explained the current runtime. This document takes the next step and shows how to add the missing agentic layer above the current `CodergenBackend` seam.

---

## 2. Coding Agent Loop — Mental Model (Top-Down)

The Coding Agent Loop is not "just another handler." It is a full orchestration subsystem.

At a high level, it looks like this:

```text
User input
   |
   v
+------------------+
|  Planner         |
|  "What next?"    |
+------------------+
   |
   v
+------------------+
|  LLM Turn        |
|  text + tools    |
+------------------+
   |
   +--------------------------+
   |                          |
   v                          v
text only                 tool calls
done                      execute tools
                              |
                              v
                       observe outputs
                              |
                              v
                       update state/history
                              |
                              v
                           repeat
```

A more complete model:

```text
Input
  -> Build request
  -> Ask model
  -> Receive assistant response
  -> If response has tool calls:
       -> execute tools
       -> record tool results
       -> inject steering if any
       -> loop again
     else:
       -> complete turn
```

The Coding Agent Loop has five core moving parts.

### 2.1 Planner

The planner is not necessarily a separate "planning model." In this spec, planning is mostly embedded in the turn loop itself:

- the current conversation state,
- tool results,
- steering messages,
- and the model's own reasoning

together determine what the agent does next.

So "planner" really means:

- build the next request correctly,
- include the right tools,
- include the right history,
- inject the right system prompt and environment context.

### 2.2 Executor

The executor runs tool calls requested by the model.

Examples:

- read a file
- write a file
- edit a file
- run a shell command
- grep
- glob
- spawn a subagent

The executor must:

- validate tool arguments,
- run the tool in an execution environment,
- capture output,
- truncate output for the model if needed,
- emit full output as events for the host.

### 2.3 State

The loop needs explicit state:

- conversation history
- tool results
- turn counts
- subagent registry
- steering queue
- follow-up queue
- loop-detection memory
- config

This is **different** from the pipeline engine's `Context`.

The pipeline `Context` is stage-to-stage execution state inside a graph run.  
The agent-loop state is conversation-and-orchestration state across model/tool rounds.

### 2.4 Orchestrator

The orchestrator is the thing that owns the loop:

- create a request
- call the LLM
- inspect the response
- dispatch tools
- update history
- stop when done

This is the `Session` in the coding-agent-loop spec.

### 2.5 Tools

Tools are the bridge from the model into the real world.

The model does not:

- open files directly
- edit files directly
- run commands directly

Instead, it emits **tool calls**, and the host executes them.

That distinction is central to the whole design.

### ASCII Architecture

```text
+------------------------------------------------------+
| Host Application                                     |
| CLI / IDE / Web UI / Pipeline Runner                 |
+------------------------------------------------------+
                     |
                     v
+------------------------------------------------------+
| Coding Agent Session                                 |
| - history                                            |
| - event emitter                                      |
| - steering queue                                     |
| - follow-up queue                                    |
| - subagents                                          |
| - config                                             |
+------------------------------------------------------+
          |                     |                 |
          v                     v                 v
   Provider Profile       Tool Registry     Execution Environment
   - model                - definitions     - read/write files
   - system prompt        - validators      - shell commands
   - provider options     - executors       - grep / glob
          |                     |                 |
          +---------------------+-----------------+
                                |
                                v
                        Unified LLM Client
                     - complete(request)
                     - stream(request)
```

### Why this matters for this repo

Today, this repository has:

- parser
- transform layer
- validation
- engine
- handlers
- CLI

What it does **not** have is the agentic subsystem above the codergen handler.

Right now, the repo expects this:

```go
type CodergenBackend interface {
	Run(node *dot.Node, prompt string, ctx *engine.Context) (*engine.Outcome, error)
}
```

That is a narrow stage-level backend.

The coding-agent-loop spec requires something much richer:

- session lifecycle
- provider profiles
- request/response normalization
- tool registry
- execution environment abstraction
- truncation policy
- event model
- subagents

### 🔗 Relation to `CODE_WALKTHROUGH.md`

This section extends:

- `CODE_WALKTHROUGH.md` section `1. High-Level Overview`
- `CODE_WALKTHROUGH.md` section `2. Execution Flow (Top → Down)`
- `CODE_WALKTHROUGH.md` section `4.6 internal/engine`

Those sections described the current **pipeline engine loop**:

```text
parse -> transform -> validate -> execute graph
```

What was missing there was the **agent loop inside codergen-like work**:

```text
request -> model -> tool calls -> tool execution -> updated request -> repeat
```

This document adds that missing inner loop.

---

## 3. Spec → Implementation Translation

This section maps the major concepts from the Coding Agent Loop spec into concrete repository work.

It is not enough to understand the pseudocode. You need to know:

- what type to build,
- which package to put it in,
- and how it plugs into the current runner.

For each concept below:

- "What It Means" translates the spec into plain English.
- "What You Need to Build" names the actual code artifact.
- "Where It Fits in the Repo" connects it to current files and to `CODE_WALKTHROUGH.md`.

---

### Spec Item
> `2.1 Session`

### What It Means (Plain English)

The session is the top-level object for one coding-agent conversation.

It owns:

- history
- event delivery
- current config
- active provider profile
- execution environment
- subagents
- steering and follow-up messages

It is the equivalent of the current `engine.Runner`, but for a model/tool conversation rather than a graph traversal.

### What You Need to Build

Concrete type:

- `type Session struct { ... }`

Concrete package:

- `internal/codergen/session.go`

Companion types:

- `SessionConfig`
- `SessionState`
- `Turn`

### Where It Fits in the Repo

Current equivalent ideas:

- `internal/engine/engine.go` has `Runner`
- `internal/engine/context.go` has run state
- `internal/event/event.go` has event emission

New fit:

- `internal/codergen` becomes the orchestration layer above the current backend seam
- `internal/handler/codergen.go` stops thinking in terms of "backend returns one response" and starts delegating to a `Session`

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `3.3 Spec Section 3: Pipeline Execution Engine`
- `CODE_WALKTHROUGH.md` section `4.6 internal/engine`

That document explained the pipeline `Runner`. This section introduces the **agent-session analogue** that would live one layer deeper, inside a codergen stage.

---

### Spec Item
> `2.2 Session Configuration`

### What It Means (Plain English)

The agent loop needs runtime limits and knobs:

- max turns
- max tool rounds
- timeout defaults
- truncation limits
- loop detection
- reasoning effort
- subagent depth

These are not graph-level concerns. They are agent-loop concerns.

### What You Need to Build

Concrete type:

- `type SessionConfig struct { ... }`

Suggested file:

- `internal/codergen/config.go`

### Where It Fits in the Repo

Current nearby type:

```go
type Config struct {
	LogsRoot string
	Vars     map[string]string
}
```

from `internal/engine/engine.go`

That config is for graph execution, not LLM/tool orchestration.

The new config should live in `internal/codergen`, and `CodergenBackend` should accept or own one.

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `3.3 Spec Section 3: Pipeline Execution Engine`

That section explained engine-level configuration. This section adds **agent-loop configuration**, which is a different layer entirely.

---

### Spec Item
> `2.3 Session Lifecycle`

### What It Means (Plain English)

The loop must have explicit runtime states:

- idle
- processing
- awaiting input
- closed

Without explicit lifecycle state, you cannot safely:

- steer a running agent,
- await user clarification,
- or manage subagents predictably.

### What You Need to Build

Concrete types:

- `type SessionState string`
- state transition methods:
  - `Submit()`
  - `Close()`
  - internal `setState()`

Suggested file:

- `internal/codergen/state.go`

### Where It Fits in the Repo

Current nearest concept:

- the engine loop is synchronous and stage-scoped

The new session lifecycle will sit above any single LLM call and may survive across many tool rounds.

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `5. End-to-End Example`

That example showed a single graph run. The missing concept there was that the codergen phase itself can have its own independent lifecycle.

---

### Spec Item
> `2.4 Turn Types`

### What It Means (Plain English)

The conversation history cannot just be a slice of strings.

You need structured turns for:

- user input
- assistant output
- tool results
- steering messages
- system messages

This is necessary because the loop must reconstruct a proper request for each model round.

### What You Need to Build

Concrete types:

- `UserTurn`
- `AssistantTurn`
- `ToolResultsTurn`
- `SystemTurn`
- `SteeringTurn`

Suggested file:

- `internal/codergen/history.go`

### Where It Fits in the Repo

Current repo state:

- pipeline state is stored as `Context`
- stage results are `Outcome`

New history layer:

- conversation history should be separate from `engine.Context`
- the backend should use conversation history to build LLM requests

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `3.5 Spec Section 5: State and Context`

That section explained `Context` and `Outcome`. This section adds a second state system: **conversation history**, which does not exist in the current repo.

---

### Spec Item
> `2.5 The Core Agentic Loop`

### What It Means (Plain English)

This is the core algorithm:

1. append user input to history
2. build request
3. call LLM
4. record assistant response
5. if no tool calls, stop
6. otherwise execute tools
7. append tool results
8. repeat

This is the agent equivalent of the pipeline engine's `runLoop()`.

### What You Need to Build

Concrete methods:

- `processInput()`
- `drainSteering()`
- `executeToolCalls()`
- `executeSingleTool()`

Suggested files:

- `internal/codergen/session.go`
- `internal/codergen/tools.go`

### Where It Fits in the Repo

Current fit:

- `internal/handler/codergen.go` currently does one prompt -> one response

New fit:

- `CodergenHandler.Execute()` should delegate into something like:

```go
session := codergen.NewSession(...)
result, err := session.RunPrompt(...)
```

instead of doing a single backend call.

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `3.3 Spec Section 3: Pipeline Execution Engine`
- `CODE_WALKTHROUGH.md` section `3.4 Spec Section 4: Node Handlers`

That document explained the outer graph loop. This section defines the inner tool-using LLM loop that belongs inside the codergen handler.

---

### Spec Item
> `2.6 Steering`

### What It Means (Plain English)

Steering lets a host inject a message into the loop between tool rounds.

Example:

- the model is exploring the wrong approach
- the user says, "Actually use the existing repository layer"
- the session queues that as a steering message
- the next LLM round sees it as new user guidance

### What You Need to Build

Concrete methods:

- `Steer(message string)`
- `FollowUp(message string)`
- internal queue handling

Suggested file:

- `internal/codergen/steering.go`

### Where It Fits in the Repo

Current repo gap:

- no concept like this exists today

Most natural integration:

- expose it from the new `Session`
- optionally surface it later via CLI flags or a future interactive mode

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `6.4 How I would extend it`

That section talked about future extensions in general terms. This section turns one of those extensions into concrete queue-based session behavior.

---

### Spec Item
> `2.7 Reasoning Effort`

### What It Means (Plain English)

The session should be able to tell the model how hard to think:

- low
- medium
- high

This must be passed through to the provider-specific request options.

### What You Need to Build

Concrete fields:

- `ReasoningEffort string`

Concrete plumbing:

- request builder sets it on outgoing LLM requests

Suggested files:

- `internal/llm/types.go`
- `internal/codergen/request_builder.go`

### Where It Fits in the Repo

There is no current equivalent. This is part of the future Unified LLM layer.

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `4.1 internal/cli`

That section covered current CLI command routing. This section adds one of the agent-loop concerns that the current CLI does not yet expose.

---

### Spec Item
> `2.8 Stop Conditions`

### What It Means (Plain English)

The agent loop must know when to stop:

- natural completion
- max tool rounds
- max turns
- abort signal
- unrecoverable error

### What You Need to Build

Concrete logic:

- stop-condition checks inside `processInput()`

Suggested file:

- `internal/codergen/session.go`

### Where It Fits in the Repo

Current equivalent:

- `engine.runLoop()` stops when the graph reaches a terminal node

New layer:

- the agent loop stops when the model stops requesting tools

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `3.3 Spec Section 3: Pipeline Execution Engine`

That section covered graph termination. This section adds **conversation-loop termination**.

---

### Spec Item
> `2.9 Event System`

### What It Means (Plain English)

The coding agent loop needs a richer event model than the current pipeline engine.

Today the repo emits:

- pipeline started/completed
- stage started/completed/failed
- retrying
- checkpoint saved

The coding-agent-loop spec adds:

- session start/end
- assistant text streaming
- tool call start/end
- tool output deltas
- steering injected
- loop detection
- warnings

### What You Need to Build

Concrete types:

- a new event type set for codergen session events

You have two choices:

1. extend `internal/event`
2. create `internal/codergen/events.go`

Recommended:

- keep pipeline events and agent-loop events separate at the type level
- bridge them later if you want a unified observer

### Where It Fits in the Repo

Current file:

- `internal/event/event.go`

Likely future split:

- `internal/event/pipeline.go`
- `internal/event/codergen.go`

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `4.8 internal/event`

That section explained the current pipeline pub-sub model. This section shows how the event model must expand for the agent loop.

---

### Spec Item
> `2.10 Loop Detection`

### What It Means (Plain English)

If the model keeps making the same tool calls over and over, the session should detect that pattern and inject a warning.

This prevents useless infinite loops like:

- `grep`
- `read_file`
- `grep`
- `read_file`
- repeating forever

### What You Need to Build

Concrete functions:

- `detectLoop()`
- `toolCallSignature()`

Suggested file:

- `internal/codergen/loop_detection.go`

### Where It Fits in the Repo

No current equivalent exists. This is brand-new behavior.

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `6.3 Weaknesses and limitations`

That section highlighted missing deeper agent behavior. Loop detection is one of the concrete guardrails the current repo does not yet have.

---

### Spec Item
> `3. Provider-Aligned Toolsets`

### What It Means (Plain English)

Different model families are best served by different tool interfaces and system prompts.

This is the opposite of "one universal tool schema for every provider."

### What You Need to Build

Concrete types:

- `ProviderProfile`
- `ToolRegistry`
- provider-specific builders:
  - `NewOpenAIProfile(...)`
  - `NewAnthropicProfile(...)`
  - `NewGeminiProfile(...)`

Suggested package:

- `internal/codergen/profile`

### Where It Fits in the Repo

Current repo has:

- `handler.Registry`, which maps graph node types to handlers

New layer needs:

- `ToolRegistry`, which maps tool names to executors

Do not confuse the two:

- `handler.Registry` is for graph execution
- `ToolRegistry` is for coding-agent tool calls

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `3.4 Spec Section 4: Node Handlers`
- `CODE_WALKTHROUGH.md` section `4.5 internal/handler`

That document explained handler registry resolution. This section introduces the **agent tool registry**, which is similar structurally but serves a different layer.

---

### Spec Item
> `4. Tool Execution Environment`

### What It Means (Plain English)

The agent loop must not hardcode "run everything on the local machine" into every tool.

Instead, all tools should go through an `ExecutionEnvironment` interface.

That allows:

- local execution
- Docker execution
- Kubernetes execution
- SSH execution
- WASM / in-memory execution

### What You Need to Build

Concrete types:

- `ExecutionEnvironment`
- `LocalExecutionEnvironment`
- optional wrappers later

Suggested package:

- `internal/codergen/env`

### Where It Fits in the Repo

Current repo gap:

- `ToolHandler` directly uses `exec.Command(...)`
- codergen handler writes directly to the local filesystem

Future refactor:

- coding-agent tools should depend on `ExecutionEnvironment`
- the graph engine can continue to use the normal local process/filesystem unless you later generalize it too

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `3.4 Spec Section 4: Node Handlers`

That section explained the current `ToolHandler`. This section shows how to generalize command and file execution for the coding-agent loop.

---

### Spec Item
> `5. Tool Output and Context Management`

### What It Means (Plain English)

The model must not receive arbitrarily large tool output.

So the loop needs:

- output truncation
- line limits
- character limits
- warning markers
- full output preservation in events

### What You Need to Build

Concrete functions:

- `truncateToolOutput()`
- `truncateOutput()`
- `truncateLines()`

Suggested file:

- `internal/codergen/truncation.go`

### Where It Fits in the Repo

Current nearby concept:

- `truncateResponse()` in `internal/handler/codergen.go` only truncates the stored simulated stage response

That is not the same thing.

The new truncation layer is for:

- **tool output returned to the LLM**

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `3.5 Spec Section 5: State and Context`
- `CODE_WALKTHROUGH.md` section `4.5 internal/handler`

That document explained current stage artifacts. This section adds a separate policy for **LLM-facing tool result compaction**.

---

### Spec Item
> `6. System Prompts and Environment Context`

### What It Means (Plain English)

Every agent request needs a layered system prompt that includes:

- provider-specific base instructions
- environment context
- tool descriptions
- project instructions
- user overrides

### What You Need to Build

Concrete functions:

- `buildSystemPrompt(profile, env, docs, overrides)`
- `discoverProjectDocs(root, cwd, provider)`

Suggested files:

- `internal/codergen/prompt.go`
- `internal/codergen/projectdocs.go`

### Where It Fits in the Repo

Current repo gap:

- current `CodergenHandler` builds a prompt from node attributes only

Future flow:

- graph node prompt becomes the **task input** to the agent loop
- the session then wraps it in provider/system/tool/project context

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `3.4 Spec Section 4: Node Handlers`

That section described how prompts are built today from `prompt` and `$goal`. This section extends prompt construction into a full agent-system request builder.

---

### Spec Item
> `7. Subagents`

### What It Means (Plain English)

Subagents are child sessions spawned by a parent session to do scoped work.

They allow:

- parallel exploration
- isolated refactoring attempts
- delegated test generation

### What You Need to Build

Concrete types:

- `SubAgentHandle`
- `SubAgentResult`
- spawn / wait / close operations

Suggested files:

- `internal/codergen/subagent.go`
- `internal/codergen/tools_subagent.go`

### Where It Fits in the Repo

Current repo gap:

- no subagent concept exists today

Future fit:

- subagents are agent-loop features, not graph-engine features
- they should live fully under `internal/codergen`

### 🔗 Relation to `CODE_WALKTHROUGH.md`

See:

- `CODE_WALKTHROUGH.md` section `6.4 How I would extend it`

That section mentioned future extensibility. This section defines one of the most important extensibility mechanisms concretely.

---

## 4. Build the Coding Agent Loop From Scratch (Top-Down)

This is the most important section.

We are going to design the agent loop as if we were adding it to this repository today.

The correct order is **top-down**, not bottom-up:

1. define interfaces
2. define request/response layer
3. define tool system
4. define execution environment
5. define session state and history
6. define loop orchestration
7. integrate with existing codergen handler

### 🔗 Relation to `CODE_WALKTHROUGH.md`

This section builds directly on:

- `CODE_WALKTHROUGH.md` section `7. Re-Implementing Attractor From Scratch: The Minimum Blueprint`

That section gave a minimum blueprint for the Attractor runner itself. This section is the "Part 2" blueprint for the **coding-agent layer that sits inside a codergen stage**.

---

### Step 1: Define Core Interfaces

Start with the abstractions, not the providers.

#### 1.1 Core loop result model

```go
package llm

type FinishReason string

const (
	FinishStop      FinishReason = "stop"
	FinishToolCalls FinishReason = "tool_calls"
	FinishLength    FinishReason = "length"
)

type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

type ToolResult struct {
	ToolCallID string
	Content    string
	IsError    bool
}

type Response struct {
	ID           string
	Text         string
	Reasoning    string
	ToolCalls    []ToolCall
	Usage        Usage
	FinishReason FinishReason
}
```

Why first:

- the rest of the system must agree on one neutral shape for model responses and tool results

#### 1.2 Unified client interface

```go
package llm

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

type Message struct {
	Role    MessageRole
	Content string
}

type Request struct {
	Model           string
	Messages        []Message
	Tools           []ToolDefinition
	ReasoningEffort string
	Provider        string
	ProviderOptions map[string]any
}

type Client interface {
	Complete(req Request) (Response, error)
}
```

This is the minimum "Unified LLM" layer the coding-agent loop requires.

#### 1.3 Tool definitions

```go
package codergen

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type RegisteredTool struct {
	Definition ToolDefinition
	Executor   ToolExecutor
}

type ToolExecutor interface {
	Execute(args map[string]any, env ExecutionEnvironment) (string, error)
}
```

#### 1.4 Execution environment

```go
package codergen

type ExecResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	TimedOut   bool
	DurationMs int64
}

type ExecutionEnvironment interface {
	ReadFile(path string, offset, limit int) (string, error)
	WriteFile(path, content string) error
	FileExists(path string) bool
	ListDirectory(path string, depth int) ([]string, error)
	ExecCommand(command string, timeoutMs int, workingDir string, envVars map[string]string) (ExecResult, error)
	Grep(pattern, path string, options GrepOptions) (string, error)
	Glob(pattern, path string) ([]string, error)
	WorkingDirectory() string
	Platform() string
	OSVersion() string
}
```

#### 1.5 Session-facing domain types

```go
package codergen

type SessionState string

const (
	SessionIdle          SessionState = "idle"
	SessionProcessing    SessionState = "processing"
	SessionAwaitingInput SessionState = "awaiting_input"
	SessionClosed        SessionState = "closed"
)

type SessionConfig struct {
	MaxTurns                 int
	MaxToolRoundsPerInput    int
	DefaultCommandTimeoutMs  int
	MaxCommandTimeoutMs      int
	ReasoningEffort          string
	ToolOutputLimits         map[string]int
	ToolLineLimits           map[string]int
	EnableLoopDetection      bool
	LoopDetectionWindow      int
	MaxSubagentDepth         int
}
```

Design rule:

- keep these types boring
- keep them provider-agnostic
- keep them serializable if possible

---

### Step 2: Define Conversation History

The loop depends on history being explicit and reconstructable.

```go
package codergen

type TurnKind string

const (
	TurnUser      TurnKind = "user"
	TurnAssistant TurnKind = "assistant"
	TurnTool      TurnKind = "tool_results"
	TurnSystem    TurnKind = "system"
	TurnSteering  TurnKind = "steering"
)

type Turn interface {
	Kind() TurnKind
}

type UserTurn struct {
	Content string
}

func (UserTurn) Kind() TurnKind { return TurnUser }

type AssistantTurn struct {
	Content    string
	Reasoning  string
	ToolCalls  []llm.ToolCall
	Usage      llm.Usage
	ResponseID string
}

func (AssistantTurn) Kind() TurnKind { return TurnAssistant }

type ToolResultsTurn struct {
	Results []llm.ToolResult
}

func (ToolResultsTurn) Kind() TurnKind { return TurnTool }

type SteeringTurn struct {
	Content string
}

func (SteeringTurn) Kind() TurnKind { return TurnSteering }
```

Why this matters:

- the request builder must turn internal history into LLM messages
- tool results must preserve tool-call IDs
- steering messages must become user-role messages later

---

### Step 3: Build the Unified LLM Layer

This repo does not yet have it, so build the smallest thing that can work.

Recommended package:

- `internal/llm`

Recommended files:

- `internal/llm/types.go`
- `internal/llm/client.go`
- `internal/llm/openai.go`
- `internal/llm/anthropic.go`
- `internal/llm/gemini.go`

#### 3.1 Start with one interface

```go
package llm

type Client interface {
	Complete(req Request) (Response, error)
}
```

#### 3.2 Then implement provider adapters

Example shape:

```go
type OpenAIClient struct {
	apiKey string
	http   *http.Client
	baseURL string
}

func (c *OpenAIClient) Complete(req Request) (Response, error) {
	// Convert normalized Request -> OpenAI Responses API payload
	// POST request
	// Convert provider payload -> normalized Response
}
```

Same pattern for Anthropic and Gemini.

#### 3.3 Why normalize first

Do not let provider-specific payloads leak into the session loop.

Bad:

```go
if provider == "openai" { ... } else if provider == "anthropic" { ... }
```

inside the session loop.

Good:

- the session depends only on `llm.Client`
- provider-specific translation stays inside provider adapter packages

---

### Step 4: Build Provider Profiles

Provider profiles decide:

- model
- tool definitions
- system prompt
- provider options
- capability flags

Recommended package:

- `internal/codergen/profile`

Core interface:

```go
package profile

type ProviderProfile interface {
	ID() string
	Model() string
	ToolRegistry() *codergen.ToolRegistry
	BuildSystemPrompt(env codergen.ExecutionEnvironment, projectDocs string) string
	Tools() []llm.ToolDefinition
	ProviderOptions() map[string]any
	SupportsReasoning() bool
	SupportsStreaming() bool
	SupportsParallelToolCalls() bool
	ContextWindowSize() int
}
```

Suggested implementations:

- `OpenAIProfile`
- `AnthropicProfile`
- `GeminiProfile`

Important design rule:

- `ProviderProfile` is where provider alignment lives
- `llm.Client` is where transport/API translation lives

Those are separate responsibilities.

---

### Step 5: Build the Tool Registry

This is the coding-agent analogue of the existing handler registry.

```go
package codergen

type ToolRegistry struct {
	tools map[string]RegisteredTool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]RegisteredTool)}
}

func (r *ToolRegistry) Register(tool RegisteredTool) {
	r.tools[tool.Definition.Name] = tool
}

func (r *ToolRegistry) Get(name string) (RegisteredTool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *ToolRegistry) Definitions() []llm.ToolDefinition {
	out := make([]llm.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.Definition)
	}
	return out
}
```

Now define built-in tool executors.

Recommended files:

- `internal/codergen/tools_readfile.go`
- `internal/codergen/tools_writefile.go`
- `internal/codergen/tools_editfile.go`
- `internal/codergen/tools_shell.go`
- `internal/codergen/tools_grep.go`
- `internal/codergen/tools_glob.go`
- later:
  - `tools_spawnagent.go`
  - `tools_wait.go`

Important warning:

- do not reuse `internal/handler/registry.go` for this
- that registry is for graph-node handlers, not model tools

---

### Step 6: Build the Execution Environment

Recommended package:

- `internal/codergen/env`

Start with:

- `local.go`

Concrete minimal implementation:

```go
type LocalExecutionEnvironment struct {
	workingDir string
}

func NewLocalExecutionEnvironment(workingDir string) *LocalExecutionEnvironment {
	return &LocalExecutionEnvironment{workingDir: workingDir}
}

func (e *LocalExecutionEnvironment) ReadFile(path string, offset, limit int) (string, error) {
	// read file from local filesystem
}

func (e *LocalExecutionEnvironment) WriteFile(path, content string) error {
	// write file to local filesystem
}

func (e *LocalExecutionEnvironment) ExecCommand(command string, timeoutMs int, workingDir string, envVars map[string]string) (ExecResult, error) {
	// use exec.CommandContext or process-group timeout logic
}
```

For this repo specifically:

- you can initially borrow ideas from:
  - current `ToolHandler`
  - current CLI operational commands
- but do not couple the environment to graph nodes or stage directories

It should be generic.

---

### Step 7: Build the Event Model

Recommended file:

- `internal/codergen/events.go`

Start with:

```go
type SessionEventKind string

const (
	EventSessionStart      SessionEventKind = "session.start"
	EventSessionEnd        SessionEventKind = "session.end"
	EventUserInput         SessionEventKind = "user.input"
	EventAssistantTextEnd  SessionEventKind = "assistant.text.end"
	EventToolCallStart     SessionEventKind = "tool.call.start"
	EventToolCallEnd       SessionEventKind = "tool.call.end"
	EventSteeringInjected  SessionEventKind = "steering.injected"
	EventTurnLimit         SessionEventKind = "turn.limit"
	EventLoopDetected      SessionEventKind = "loop.detected"
	EventWarning           SessionEventKind = "warning"
	EventError             SessionEventKind = "error"
)

type SessionEvent struct {
	Kind      SessionEventKind
	Timestamp time.Time
	SessionID string
	Data      map[string]any
}
```

Recommended design:

- do not overload the current pipeline `event.Kind`
- create a codergen-specific event type set
- if needed later, create adapters to convert both into a shared UI stream

---

### Step 8: Implement the Session Loop

This is the core.

Recommended file:

- `internal/codergen/session.go`

Suggested implementation shape:

```go
type Session struct {
	id            string
	profile       profile.ProviderProfile
	client        llm.Client
	env           ExecutionEnvironment
	registry      *ToolRegistry
	config        SessionConfig
	state         SessionState
	history       []Turn
	steeringQueue []string
	followups     []string
	events        *Emitter
	subagents     map[string]*SubAgentHandle
}
```

Core loop:

```go
func (s *Session) ProcessInput(userInput string) (string, error) {
	s.state = SessionProcessing
	s.history = append(s.history, UserTurn{Content: userInput})
	s.emit(EventUserInput, map[string]any{"content": userInput})

	s.drainSteering()

	roundCount := 0

	for {
		if s.hitStopCondition(roundCount) {
			break
		}

		req := s.buildRequest()
		resp, err := s.client.Complete(req)
		if err != nil {
			return "", err
		}

		s.history = append(s.history, AssistantTurn{
			Content:    resp.Text,
			Reasoning:  resp.Reasoning,
			ToolCalls:  resp.ToolCalls,
			Usage:      resp.Usage,
			ResponseID: resp.ID,
		})

		if len(resp.ToolCalls) == 0 {
			s.state = SessionIdle
			return resp.Text, nil
		}

		results := s.executeToolCalls(resp.ToolCalls)
		s.history = append(s.history, ToolResultsTurn{Results: results})

		s.drainSteering()

		roundCount++
		s.detectLoopAndWarn()
	}

	s.state = SessionIdle
	return s.finalAssistantText(), nil
}
```

That is the heart of the coding agent loop.

---

### Step 9: Build Request Construction

Recommended file:

- `internal/codergen/request_builder.go`

Core idea:

```go
func (s *Session) buildRequest() llm.Request {
	systemPrompt := s.profile.BuildSystemPrompt(s.env, s.discoverProjectDocs())
	messages := convertHistoryToMessages(s.history)

	return llm.Request{
		Model:           s.profile.Model(),
		Messages:        append([]llm.Message{{Role: llm.RoleSystem, Content: systemPrompt}}, messages...),
		Tools:           s.profile.Tools(),
		ReasoningEffort: s.config.ReasoningEffort,
		Provider:        s.profile.ID(),
		ProviderOptions: s.profile.ProviderOptions(),
	}
}
```

History conversion rules:

- `UserTurn` -> user message
- `AssistantTurn` -> assistant message
- `ToolResultsTurn` -> tool-role messages or provider-equivalent
- `SteeringTurn` -> user message
- `SystemTurn` -> optional extra system message

Keep that conversion in one place. Do not scatter it across the loop.

---

### Step 10: Implement Tool Execution

Recommended file:

- `internal/codergen/tools.go`

Shape:

```go
func (s *Session) executeToolCalls(calls []llm.ToolCall) []llm.ToolResult {
	results := make([]llm.ToolResult, 0, len(calls))
	for _, call := range calls {
		results = append(results, s.executeSingleTool(call))
	}
	return results
}

func (s *Session) executeSingleTool(call llm.ToolCall) llm.ToolResult {
	s.emit(EventToolCallStart, map[string]any{
		"tool_name": call.Name,
		"call_id":   call.ID,
	})

	registered, ok := s.registry.Get(call.Name)
	if !ok {
		return llm.ToolResult{
			ToolCallID: call.ID,
			Content:    "Unknown tool: " + call.Name,
			IsError:    true,
		}
	}

	rawOutput, err := registered.Executor.Execute(call.Arguments, s.env)
	if err != nil {
		return llm.ToolResult{
			ToolCallID: call.ID,
			Content:    "Tool error (" + call.Name + "): " + err.Error(),
			IsError:    true,
		}
	}

	truncated := truncateToolOutput(rawOutput, call.Name, s.config)

	s.emit(EventToolCallEnd, map[string]any{
		"tool_name":   call.Name,
		"call_id":     call.ID,
		"full_output": rawOutput,
	})

	return llm.ToolResult{
		ToolCallID: call.ID,
		Content:    truncated,
		IsError:    false,
	}
}
```

This is where tool execution, truncation, and eventing meet.

---

### Step 11: Implement Truncation

Recommended file:

- `internal/codergen/truncation.go`

Core rules:

1. truncate by character count first
2. then optionally truncate by lines
3. include visible markers
4. emit full output separately

Minimal implementation:

```go
func truncateToolOutput(output, toolName string, config SessionConfig) string {
	maxChars := defaultToolCharLimit(toolName, config)
	out := truncateChars(output, maxChars)

	maxLines, ok := defaultToolLineLimit(toolName, config)
	if ok {
		out = truncateLines(out, maxLines)
	}
	return out
}
```

This should be reusable and fully independent from provider code.

---

### Step 12: Add Subagents

Do this only after the core session works.

Recommended files:

- `internal/codergen/subagent.go`
- `internal/codergen/tools_subagent.go`

Core idea:

```go
type SubAgentHandle struct {
	ID      string
	Session *Session
	Status  string
	Result  string
}
```

Tool interface:

- `spawn_agent`
- `send_input`
- `wait`
- `close_agent`

Important rule:

- subagents are just child sessions
- same execution environment
- independent history
- bounded depth

---

### Step 13: Integrate with the Existing Codergen Handler

This is where the new subsystem actually touches the current repo.

Today:

```go
type CodergenBackend interface {
	Run(node *dot.Node, prompt string, ctx *engine.Context) (*engine.Outcome, error)
}
```

Recommended migration:

#### Option A: Keep `CodergenBackend`, make it a facade

```go
type CodergenBackend interface {
	Run(node *dot.Node, prompt string, ctx *engine.Context) (*engine.Outcome, error)
}

type AgentLoopBackend struct {
	SessionFactory *SessionFactory
}

func (b *AgentLoopBackend) Run(node *dot.Node, prompt string, ctx *engine.Context) (*engine.Outcome, error) {
	session := b.SessionFactory.NewSession(...)
	text, err := session.ProcessInput(prompt)
	if err != nil {
		return &engine.Outcome{
			Status:        engine.StatusFail,
			FailureReason: err.Error(),
		}, nil
	}
	return &engine.Outcome{
		Status: engine.StatusSuccess,
		Notes:  "Agent loop completed: " + node.ID,
		ContextUpdates: map[string]string{
			"last_stage":    node.ID,
			"last_response": text,
		},
	}, nil
}
```

This is the safest migration because the graph engine does not change.

#### Option B: Expand the handler/backend boundary

Less safe in the short term, but cleaner long term:

- codergen handler gets a `SessionFactory`
- backend abstraction moves from "single call" to "session orchestration"

Recommended for this repo:

- **Option A first**
- **Option B later**

That lets the engine and CLI remain stable while the agent loop matures.

---

## 5. Integrating the Coding Agent Loop Into This Repository

This section turns the abstract build steps into a concrete repository migration plan.

### 🔗 Relation to `CODE_WALKTHROUGH.md`

This section builds on:

- `CODE_WALKTHROUGH.md` section `4.1 internal/cli`
- `CODE_WALKTHROUGH.md` section `4.5 internal/handler`
- `CODE_WALKTHROUGH.md` section `4.6 internal/engine`
- `CODE_WALKTHROUGH.md` section `6.4 How I would extend it`

That document explained current package boundaries. This section proposes how to extend those boundaries without breaking them.

---

### 5.1 Current insertion point

The current insertion point is:

```go
type CodergenBackend interface {
	Run(node *dot.Node, prompt string, ctx *engine.Context) (*engine.Outcome, error)
}
```

This is good news.

It means the graph engine already has a clean seam where the coding-agent loop can be inserted.

### 5.2 Proposed package layout

Add these packages:

```text
internal/
  llm/
    types.go
    client.go
    openai.go
    anthropic.go
    gemini.go
  codergen/
    config.go
    state.go
    history.go
    session.go
    request_builder.go
    prompt.go
    projectdocs.go
    tools.go
    truncation.go
    loop_detection.go
    subagent.go
    events.go
    env/
      local.go
    profile/
      profile.go
      openai.go
      anthropic.go
      gemini.go
```

Keep existing packages:

- `internal/engine`
- `internal/handler`
- `internal/validate`
- `internal/dot`
- `internal/transform`
- `internal/interviewer`

Do not bury the session loop inside `internal/handler`. That would mix orchestration and graph-node execution too tightly.

### 5.3 Adapt `internal/handler/codergen.go`

Right now, the handler:

- resolves a prompt
- writes `prompt.md`
- calls a backend
- writes `response.md`
- returns success/failure

That should remain its job.

What changes is the backend implementation.

So the handler stays stage-focused, and the new backend becomes conversation-focused.

That is the right separation.

### 5.4 Extend CLI configuration

Eventually `runCmd()` should be able to configure:

- provider (`openai`, `anthropic`, `gemini`)
- model name
- API base URL overrides
- reasoning effort
- timeout defaults
- project-doc loading behavior
- whether codergen uses simulation or real model execution

Possible future flags:

```text
-provider <name>
-model <id>
-reasoning <low|medium|high>
-command-timeout <duration>
-context-window-warning <percent>
```

Recommended rollout:

1. keep `-simulate`
2. add `-provider` and `-model`
3. later add more loop-level controls

### 5.5 Event integration

The current CLI subscribes to pipeline events:

- `pipeline.started`
- `stage.started`
- `stage.completed`
- `stage.failed`

When the agent loop exists, the CLI should optionally subscribe to codergen session events too.

Example:

```text
  ► [write_main] Wire main.go
    ↳ agent session started
    ↳ tool: read_file internal/platform/config/config.go
    ↳ tool: shell go test ./...
    ↳ agent completed
    ✓ completed
```

This will make debugging dramatically easier.

### 5.6 How stage artifacts should evolve

Current codergen stages write:

- `prompt.md`
- `response.md`
- `status.json`

With the agent loop, a richer artifact layout is better:

```text
006_write_main/
  prompt.md
  response.md
  status.json
  agent/
    session.json
    messages.json
    tool_calls.json
    events.json
    tools/
      001_read_file.txt
      002_shell.txt
```

This is not required for correctness, but it is the best way to make the new layer inspectable.

### 5.7 Migration strategy

Use this order:

1. implement `internal/llm` normalized client
2. implement `internal/codergen` session loop
3. build one provider profile first
4. create `AgentLoopBackend`
5. wire it into `CodergenHandler`
6. add CLI flags
7. add richer artifacts and events
8. add subagents last

Do not start with subagents or multi-provider complexity.

Start with:

- one provider
- one execution environment
- one session loop
- a small tool set

---

## 6. How To Use the Coding Agent Loop Once Implemented

This section explains usage, not just architecture.

### 🔗 Relation to `CODE_WALKTHROUGH.md`

This section extends:

- `CODE_WALKTHROUGH.md` section `5. End-to-End Example`
- `CODE_WALKTHROUGH.md` section `8. Final Takeaways`

That document showed how to run the existing graph engine. This section shows how a host application would use the future coding-agent subsystem directly and indirectly.

---

### 6.1 Direct library usage

If you want to use the coding-agent loop programmatically:

```go
client := llm.NewOpenAIClient(apiKey, baseURL)
env := env.NewLocalExecutionEnvironment("/path/to/project")

registry := codergen.NewToolRegistry()
codergen.RegisterCoreTools(registry)

profile := profile.NewOpenAIProfile(
	"gpt-5.2-codex",
	registry,
)

session := codergen.NewSession(
	profile,
	client,
	env,
	codergen.DefaultSessionConfig(),
)

result, err := session.ProcessInput("Read main.go, then add a healthcheck endpoint and run tests.")
if err != nil {
	log.Fatal(err)
}

fmt.Println(result)
```

This is the "library-first" usage model from the spec.

### 6.2 Using it indirectly through the graph engine

Once wrapped in `AgentLoopBackend`, normal graph execution can use it automatically.

Example concept:

```go
backend := &handler.AgentLoopBackend{
	SessionFactory: codergen.NewSessionFactory(...),
}

reg := handler.BuildDefaultRegistry(backend, interviewer.NewConsoleInterviewer())
runner := engine.NewRunner(graph, config, &handler.RegistryAdapter{Reg: reg}, event.NewEmitter())
```

Now any `box` / `codergen` node uses the full tool-using coding-agent loop instead of a single-shot simulated response.

### 6.3 CLI usage in this repo

Once wired into `run`, expected usage could look like:

```bash
jga run examples/gorestspec/init_rest_app.dot \
  -provider openai \
  -model gpt-5.2-codex \
  -reasoning high \
  -var module_name="github.com/acme/api" \
  -var first_module="user"
```

For dry-run or current behavior:

```bash
jga run examples/gorestspec/init_rest_app.dot -simulate
```

### 6.4 Interactive steering

Later, if the CLI becomes interactive, it could support steering commands like:

```text
:steer Use the existing repository package instead of creating a new one
:followup After finishing, write tests for the handler
```

That would be a natural evolution once the session loop exists.

### 6.5 Operational inspection

This repo already has:

- `jga inspect`
- `jga list`
- `jga graph`

Those commands will become even more valuable once codergen stages contain richer agent-loop artifacts.

---

## 7. Concrete Build Plan for This Repository

If you were implementing this in the current repo, here is the exact order I would use.

### 🔗 Relation to `CODE_WALKTHROUGH.md`

This section directly extends:

- `CODE_WALKTHROUGH.md` section `6.4 How I would extend it`
- `CODE_WALKTHROUGH.md` section `7. Re-Implementing Attractor From Scratch: The Minimum Blueprint`

That document ended with extension ideas. This section turns those ideas into a concrete, repo-specific build sequence.

---

### Phase 1: Establish the Unified LLM layer

Files:

- `internal/llm/types.go`
- `internal/llm/client.go`
- `internal/llm/openai.go`

Goal:

- one provider only
- one normalized `Client` interface
- one normalized `Request` / `Response`

Do not do three providers at once.

### Phase 2: Establish `internal/codergen/session.go`

Files:

- `internal/codergen/config.go`
- `internal/codergen/state.go`
- `internal/codergen/history.go`
- `internal/codergen/session.go`

Goal:

- minimal loop with no tools yet
- text-only request/response cycle

### Phase 3: Add tool registry and local execution environment

Files:

- `internal/codergen/tools.go`
- `internal/codergen/env/local.go`
- `internal/codergen/tools_readfile.go`
- `internal/codergen/tools_shell.go`

Goal:

- first useful agentic loop
- model can read code and run commands

### Phase 4: Add truncation, events, and project docs

Files:

- `internal/codergen/events.go`
- `internal/codergen/truncation.go`
- `internal/codergen/prompt.go`
- `internal/codergen/projectdocs.go`

Goal:

- safe tool output handling
- inspectable runtime
- better prompting

### Phase 5: Wrap it as a backend

Files:

- `internal/handler/agent_loop_backend.go`

Goal:

- keep the current engine API stable
- use the new loop from existing graph execution

### Phase 6: Add CLI flags

Files:

- `internal/cli/cli.go`
- `internal/cli/usage.md`
- `README.md`

Goal:

- let users actually select provider/model/reasoning

### Phase 7: Add subagents

Files:

- `internal/codergen/subagent.go`
- `internal/codergen/tools_subagent.go`

Goal:

- parallel delegated work

Only do this once the single-session path is stable.

---

## 8. Testing Strategy for the New Layer

You should not build this without tests.

### 8.1 Unit tests

Add tests for:

- request construction
- history conversion
- truncation
- loop detection
- tool registry dispatch
- environment variable filtering

### 8.2 Provider adapter tests

Use fake HTTP servers to verify:

- request serialization
- response normalization
- tool-call extraction

### 8.3 Session tests

Add a fake LLM client:

```go
type FakeClient struct {
	Responses []llm.Response
	Pos       int
}

func (c *FakeClient) Complete(req llm.Request) (llm.Response, error) {
	resp := c.Responses[c.Pos]
	c.Pos++
	return resp, nil
}
```

Now you can test:

- text-only completion
- one tool round
- multiple tool rounds
- steering injection
- loop detection
- stop conditions

### 8.4 Integration tests

Once `AgentLoopBackend` exists, add a pipeline integration test that runs:

- a tiny graph with one codergen node
- a fake client that asks for `read_file`
- then returns final text

That proves the graph engine and agent loop are actually connected.

---

## 9. Design Decisions and Tradeoffs

### 9.1 Why not put the coding-agent loop inside `internal/engine`?

Because the engine already has one job:

- traverse a graph

If you put agent-session logic there too, you mix:

- graph orchestration
- LLM transport
- tool orchestration
- provider profiles

That would make the engine package much harder to reason about.

### 9.2 Why keep `CodergenBackend` initially?

Because it gives you a stable migration seam.

Current system:

```text
CodergenHandler -> CodergenBackend
```

Future system:

```text
CodergenHandler -> AgentLoopBackend -> Session -> LLM Client + Tool Registry + Env
```

That lets you evolve the backend without rewriting the engine.

### 9.3 Why separate provider profile from LLM client?

Because they solve different problems.

`Client` answers:

- how do I call this provider API?

`ProviderProfile` answers:

- what tools, prompt style, options, and constraints should I use for this model family?

If you combine them, the design becomes harder to extend and harder to test.

### 9.4 Why keep pipeline events and session events distinct?

Because they describe different layers of behavior.

Pipeline events:

- stage started
- stage completed
- pipeline completed

Session events:

- tool called
- steering injected
- assistant text produced
- loop detected

They may eventually share a transport, but they should not share a type definition prematurely.

---

## 10. Final Implementation Summary

If you remember only one thing from this document, remember this architecture:

```text
Attractor graph engine = outer workflow loop
Coding agent session   = inner LLM/tool loop
Unified LLM client     = provider-agnostic model transport
Provider profile       = provider-specific behavior contract
Execution environment  = where tools actually run
```

In this repository, the cleanest integration point is still:

```go
type CodergenBackend interface {
	Run(node *dot.Node, prompt string, ctx *engine.Context) (*engine.Outcome, error)
}
```

So the best path forward is:

1. build `internal/llm`
2. build `internal/codergen`
3. implement `AgentLoopBackend`
4. wire it into `CodergenHandler`
5. expose configuration through the CLI

That gives you a true coding-agent loop **without destabilizing the current Attractor runner**.

---

## 11. Suggested Reading Order After This Document

Once you finish `CODERGEN.md`, read the repo in this order:

1. `CODE_WALKTHROUGH.md`
2. `internal/handler/registry.go`
3. `internal/handler/codergen.go`
4. `internal/engine/engine.go`
5. `internal/event/event.go`

Then, when you start implementing the new layer, create and read in this order:

1. `internal/llm/types.go`
2. `internal/llm/client.go`
3. `internal/codergen/config.go`
4. `internal/codergen/history.go`
5. `internal/codergen/session.go`
6. `internal/codergen/tools.go`
7. `internal/codergen/env/local.go`
8. `internal/codergen/profile/profile.go`
9. `internal/handler/agent_loop_backend.go`

That sequence minimizes confusion because each file depends on the concepts before it.

---

## 12. Practical Next Step

If you want the shortest path to a working implementation in this repo, do this first:

### Milestone 1

- implement `internal/llm/types.go`
- implement `internal/llm/openai.go`
- implement `internal/codergen/session.go`
- implement only three tools:
  - `read_file`
  - `shell`
  - `glob`
- wrap them in `AgentLoopBackend`
- call that backend from `CodergenHandler`

That is enough to move from:

```text
single prompt -> simulated response
```

to:

```text
prompt -> model -> tool calls -> tool execution -> final answer
```

And that is the moment this repository stops being only an Attractor-style graph runner and starts becoming an Attractor-based coding agent system.

