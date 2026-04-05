# thinker — User Guide

This guide covers the library in depth for both human developers and AI coding agents. It is structured in three parts that mirror the library's own layering:

1. [Core abstractions](#1-core-abstractions) — the interfaces every component depends on
2. [Agentic toolkit](#2-agentic-toolkit) — the three ready-made agent runtimes
3. [Agent development](#3-agent-development) — how to build, compose, and deploy agents

---

## Table of contents

- [1. Core abstractions](#1-core-abstractions)
  - [1.1 The agent model](#11-the-agent-model)
  - [1.2 Encoder and Decoder](#12-encoder-and-decoder)
  - [1.3 Memory](#13-memory)
  - [1.4 Reasoner and Phase state-machine](#14-reasoner-and-phase-state-machine)
  - [1.5 Registry — MCP tools](#15-registry--mcp-tools)
  - [1.6 Errors](#16-errors)
- [2. Agentic toolkit](#2-agentic-toolkit)
  - [2.1 Prompter](#21-prompter)
  - [2.2 Manifold](#22-manifold)
  - [2.3 Automata](#23-automata)
  - [2.4 Choosing the right agent type](#24-choosing-the-right-agent-type)
- [3. Agent development](#3-agent-development)
  - [3.1 nanobot Runtime](#31-nanobot-runtime)
  - [3.2 Prompt files](#32-prompt-files)
  - [3.3 ReAct — tool-use agent from a file](#33-react--tool-use-agent-from-a-file)
  - [3.4 Seq — two-step pipeline](#34-seq--two-step-pipeline)
  - [3.5 ThinkReAct — plan then execute](#35-thinkreact--plan-then-execute)
  - [3.6 Reflect — judge-and-correct loop](#36-reflect--judge-and-correct-loop)
  - [3.7 Jsonify — JSON array extraction](#37-jsonify--json-array-extraction)
  - [3.8 Shared state pattern](#38-shared-state-pattern)
  - [3.9 Composing agents](#39-composing-agents)
  - [3.10 Deployment patterns](#310-deployment-patterns)
- [Appendix: package map](#appendix-package-map)

---

## 1. Core abstractions

Package: `github.com/kshard/thinker`

### 1.1 The agent model

Every agent in `thinker` is a function:

```
ƒ: A ⟼ B
```

It takes a typed Go input `A`, interacts with an LLM (and optionally with tools and memory), and returns a typed Go output `B`. The Go method signature is:

```go
Prompt(ctx context.Context, input A, opt ...chatter.Opt) (B, error)
```

This signature is uniform across all three agent types (`Prompter`, `Manifold`, `Automata`) and across all nanobot composites (`Seq`, `ThinkReAct`, `Reflect`). An agent of type `ƒ: A ⟼ B` can be plugged into any slot that expects the same signature, making the hierarchy fully composable.

The `chatter.Opt` variadic allows callers to pass provider-specific options (sampling temperature, max tokens, etc.) without breaking the interface.

### 1.2 Encoder and Decoder

```go
// github.com/kshard/thinker — codec.go

type Encoder[T any] interface {
    Encode(T) (chatter.Message, error)
}

type Decoder[T any] interface {
    Decode(*chatter.Reply) (float64, T, error)
}
```

**Encoder** transforms application-domain input of type `T` into a `chatter.Message` (typically a `*chatter.Prompt`). It is the single place where prompt engineering lives for an agent.

**Decoder** transforms the LLM's raw reply into a domain type `T` and returns:
- `float64` — confidence in `[0, 1]`. The `Reasoner` can use this to decide whether to retry.
- `T` — the decoded value (may be zero if decoding failed).
- `error` — either a hard error (stops the loop) or a `chatter.Feedback` (passed to the `Reasoner` to guide the next LLM call).

**Returning feedback instead of a hard error** is the key mechanism for self-correcting agents. Use `thinker.Feedback(note, lines...)` to construct feedback:

```go
func myDecoder(reply *chatter.Reply) (float64, MyType, error) {
    var result MyType
    if err := json.Unmarshal([]byte(reply.String()), &result); err != nil {
        // Send corrective feedback; the agent will retry/refine instead of aborting
        return 0, result, thinker.Feedback(
            "Improve the response based on feedback:",
            "The output is not valid JSON: "+err.Error(),
            "Reply with a single JSON object only, no prose.",
        )
    }
    return 1.0, result, nil
}
```

**Ready-made codecs** in `github.com/kshard/thinker/codec`:

| Symbol                 | Input                       | Output            | Notes                       |
| ---------------------- | --------------------------- | ----------------- | --------------------------- |
| `codec.EncoderID`      | `string`                    | `chatter.Message` | Wraps string as prompt task |
| `codec.DecoderID`      | `*chatter.Reply`            | `*chatter.Reply`  | Identity; no transformation |
| `codec.String`         | `string` / `*chatter.Reply` | `string`          | Both encoder and decoder    |
| `codec.FromEncoder(f)` | any `A`                     | `chatter.Message` | Lifts a function            |
| `codec.FromDecoder(f)` | `*chatter.Reply`            | any `B`           | Lifts a function            |

### 1.3 Memory

```go
// github.com/kshard/thinker — memory.go

type Memory interface {
    Purge()
    Commit(*Observation)
    Context(chatter.Message) []chatter.Message
}
```

Memory is the agent's experience database. Its responsibilities are:
- **`Purge`** — discard all state (called by `PromptOnce` for isolated sessions).
- **`Commit`** — record an `Observation` (the LLM prompt–reply pair with metadata).
- **`Context`** — build the ordered list of `chatter.Message` that forms the context window for the next LLM call.

Every `Context` call prepends the `stratum` (system instruction) and then appends past observations in chronological order, finishing with the current prompt.

**Built-in implementations** in `github.com/kshard/thinker/memory`:

#### `memory.NewVoid(stratum)`

Discards every observation. The context window always contains: `[stratum, currentPrompt]`.

Use for stateless agents (`Prompter`) or any agent where conversation history is not needed.

```go
memory.NewVoid("You are an autonomous agent who performs tasks defined in the prompt.")
```

#### `memory.NewStream(cap, stratum)`

Retains observations in a time-ordered list. The context window is:
```
[stratum, query₁, reply₁, query₂, reply₂, …, currentPrompt]
```

`cap` limits the number of retained observations. Pass `memory.INFINITE` (`-1`) to keep all observations. When `cap` is exceeded, the oldest observation is evicted.

```go
memory.NewStream(memory.INFINITE, "You are an autonomous agent.")
memory.NewStream(10, "You are an autonomous agent.")  // keep last 10 observations
```

`Stream` is concurrency-safe (protected by a `sync.Mutex`), so the same `Stream` instance can be shared across a pipeline while each call adds to a shared context.

### 1.4 Reasoner and Phase state-machine

```go
// github.com/kshard/thinker — reasoner.go

type Reasoner[B any] interface {
    Purge()
    Deduct(State[B]) (Phase, chatter.Message, error)
}
```

The `Reasoner` drives the `Automata` loop. After each LLM call the loop calls `Deduct` with the current `State[B]`, which carries:

```go
type State[B any] struct {
    Phase      Phase           // current phase
    Epoch      int             // number of LLM calls so far in this session
    Reply      B               // decoded reply from the last LLM call
    Confidence float64         // decoder confidence [0, 1]
    Feedback   chatter.Content // feedback from the decoder (if any)
}
```

`Deduct` returns a `Phase` that tells the `Automata` what to do next:

```
AGENT_ASK    → issue a new prompt (new sub-goal); the returned chatter.Message is the new prompt
AGENT_RETURN → accept State.Reply and return it to the caller
AGENT_RETRY  → resend the last prompt without updating memory
AGENT_REFINE → resend with a refined prompt (returned chatter.Message replaces current prompt)
AGENT_ABORT  → halt; returned error becomes the call error
```

**State-machine diagram:**

```
                     ┌──────────────────────────────────────┐
                     │           Automata loop               │
  Prompt(input) ─────▶  Encode ─▶ Context ─▶ LLM call       │
                     │              │                        │
                     │           Decode reply                │
                     │              │                        │
                     │           Commit observation           │
                     │              │                        │
                     │        Reasoner.Deduct(state)         │
                     │         /    |    \    \    \         │
                     │      ASK RETURN RETRY REFINE ABORT    │
                     │      │     │     │     │      │       │
                     │   new    return  ──────┤   error      │
                     │  prompt  B      retry prompt          │
                     └──────────────────────────────────────┘
```

**Built-in implementations** in `github.com/kshard/thinker/reasoner`:

#### `reasoner.NewVoid[B]()`

Always returns `AGENT_RETURN`. Turns `Automata` into a one-shot agent.

#### `reasoner.From(f)`

Lifts any function `func(State[B]) (Phase, chatter.Message, error)` into a `Reasoner[B]`. This is the primary extension point for custom reasoning logic.

```go
func myDeduct(state thinker.State[MyOutput]) (thinker.Phase, chatter.Message, error) {
    if state.Confidence >= 0.9 {
        return thinker.AGENT_RETURN, nil, nil
    }
    if state.Epoch >= 3 {
        return thinker.AGENT_ABORT, nil, errors.New("confidence too low after 3 attempts")
    }
    // Ask for refinement
    var p chatter.Prompt
    p.WithTask("Your previous answer had confidence %.0f%%. Please improve it.", state.Confidence*100)
    return thinker.AGENT_REFINE, &p, nil
}

reasoner.From(myDeduct)
```

#### `reasoner.NewEpoch(max, inner)`

A guard decorator that wraps any `Reasoner` and aborts when `state.Epoch >= max`. Always compose this outermost:

```go
reasoner.NewEpoch(5, reasoner.From(myDeduct))
```

### 1.5 Registry — MCP tools

```go
// github.com/kshard/thinker — registry.go

type Registry interface {
    Context() chatter.Registry
    Invoke(*chatter.Reply) (Phase, chatter.Message, error)
}
```

`Registry` is the agent's tool interface. `Context()` returns a `chatter.Registry` (the tool schema) that is injected into every LLM call so the model knows which tools are available. `Invoke` dispatches a tool-call request from the LLM reply to the appropriate MCP server.

The [`command`](../command/) package provides `*command.Registry`, the standard implementation:

```go
registry := command.NewRegistry()

// Connect via stdio (spawns a process)
registry.ConnectCmd("fs", []string{"uvx", "mcp-server-filesystem", "/tmp"})

// Connect via SSE URL
registry.ConnectUrl("web", "https://mcp.example.com/sse")

// Attach a pre-connected mcp.ClientSession
registry.Attach("local", session)
```

Tool names are automatically namespaced: a tool `read` on server `fs` becomes `fs_read`. This satisfies the `[a-zA-Z0-9_-]` constraint of AWS Bedrock and avoids conflicts between servers.

`Registry` is built into `Manifold`. For `Automata`, it must be called explicitly from the `Decoder` or `Reasoner` logic.

### 1.6 Errors

All agent errors are declared in the root package:

| Symbol                   | Meaning                                   |
| ------------------------ | ----------------------------------------- |
| `thinker.ErrCodec`       | Encoder or decoder failure                |
| `thinker.ErrLLM`         | LLM I/O failure                           |
| `thinker.ErrAborted`     | Agent was aborted (e.g. by `AGENT_ABORT`) |
| `thinker.ErrMaxEpoch`    | Epoch limit reached                       |
| `thinker.ErrCmd`         | MCP tool invocation failure               |
| `thinker.ErrCmdConflict` | Duplicate server ID in registry           |
| `thinker.ErrCmdInvalid`  | Malformed server specification            |

All errors wrap the underlying cause and can be unwrapped with `errors.As` / `errors.Is`.

---

## 2. Agentic toolkit

Package: `github.com/kshard/thinker/agent`

This package provides three agent constructors that assemble the core interfaces into working agents. All three return a type with a `Prompt(ctx, input) (B, error)` method.

### 2.1 Prompter

```go
type Prompter[A any] struct{ *Automata[A, *chatter.Reply] }

func NewPrompter[A any](
    llm chatter.Chatter,
    f func(A) (chatter.Message, error),
) *Prompter[A]
```

`Prompter[A]` is the simplest possible agent: no memory, no reasoning, one LLM call, raw reply. It is backed by `Automata` wired with:
- `memory.NewVoid` — discard all observations
- `codec.FromEncoder(f)` — use the provided function as encoder
- `codec.DecoderID` — return the raw `*chatter.Reply`
- `reasoner.NewVoid` — always return

**When to use:** translating input into an LLM prompt and returning the raw reply; when the caller handles parsing; prompt engineering experiments.

```go
agt := agent.NewPrompter(llm, func(q string) (chatter.Message, error) {
    var p chatter.Prompt
    p.WithTask("Answer the question: %s", q)
    return &p, nil
})

reply, err := agt.Prompt(ctx, "What is the boiling point of water?")
fmt.Println(reply.String())
```

### 2.2 Manifold

```go
type Manifold[A, B any] struct { /* ... */ }

func NewManifold[A, B any](
    llm      chatter.Chatter,
    encoder  thinker.Encoder[A],
    decoder  thinker.Decoder[B],
    registry thinker.Registry,
) *Manifold[A, B]
```

`Manifold[A, B]` is a tool-use loop where the LLM itself drives the reasoning. On each iteration:
1. Call the LLM with the accumulated conversation.
2. If the LLM returns a tool-call request → execute it via `registry.Invoke` → append the result to the conversation → repeat.
3. If the LLM returns a final reply → pass it to the decoder → return.

The decoder may return feedback (a `chatter.Content` error) to request a refinement, in which case the feedback is appended to the conversation and the loop continues.

**When to use:** workflows where the LLM decides which tools to call and in what order; structured output extraction with tool-assisted retrieval; any task where delegating reasoning to the LLM is acceptable.

**Important:** `Manifold` requires a model with reliable tool-use / function-calling support. Test with your chosen model before deploying.

```go
registry := command.NewRegistry()
registry.ConnectCmd("fs", []string{"mcp-server-filesystem", "/data"})

agt := agent.NewManifold(
    llm,
    codec.FromEncoder(func(q string) (chatter.Message, error) {
        var p chatter.Prompt
        p.WithTask("Find all files containing the word '%s' and return their names.", q)
        return &p, nil
    }),
    codec.String,
    registry,
)

result, err := agt.Prompt(ctx, "hello")
```

### 2.3 Automata

```go
type Automata[A, B any] struct { /* ... */ }

func NewAutomata[A, B any](
    llm      chatter.Chatter,
    memory   thinker.Memory,
    encoder  thinker.Encoder[A],
    decoder  thinker.Decoder[B],
    reasoner thinker.Reasoner[B],
) *Automata[A, B]
```

`Automata[A, B]` is the general-purpose agent runtime. The application controls all four dimensions: memory, encoding, decoding, and reasoning.

**Loop:**

```
1. Encode input A → prompt
2. Build context window from memory (prepend past observations)
3. Call LLM with context window
4. Decode reply → (confidence, B, error/feedback)
5. If no hard error: Commit observation to memory
6. Reasoner.Deduct(state) → next Phase + optional new prompt
7. Switch on Phase:
   - AGENT_ASK    → set new prompt, reset epoch, loop from step 2
   - AGENT_RETURN → return B
   - AGENT_RETRY  → loop from step 3 (skip memory update)
   - AGENT_REFINE → set new refined prompt, loop from step 2
   - AGENT_ABORT  → return error
```

**Isolated sessions:** by default, memory persists across consecutive `Prompt` calls. Call `PromptOnce` to purge memory and reasoner state first:

```go
// Stateful: observations accumulate across calls
result1, _ := agt.Prompt(ctx, "first question")
result2, _ := agt.Prompt(ctx, "related follow-up")

// Stateless: each call starts fresh
result3, _ := agt.PromptOnce(ctx, "unrelated question")
```

**Assembling an Automata:**

```go
agt := agent.NewAutomata(
    llm,

    // Memory: retain all observations, system instruction sets the agent's persona
    memory.NewStream(memory.INFINITE, `
        You are a research assistant. Think step by step.
        Always verify facts before reporting them.
    `),

    // Encoder: translate MyQuery to a chatter prompt
    codec.FromEncoder(func(q MyQuery) (chatter.Message, error) {
        var p chatter.Prompt
        p.WithTask("Research the following topic: %s", q.Topic)
        p.WithRules(
            "Cite at least one source for every claim.",
            "Reply in plain text, no markdown.",
        )
        return &p, nil
    }),

    // Decoder: parse LLM text into MyResult; send feedback if parsing fails
    codec.FromDecoder(func(reply *chatter.Reply) (float64, MyResult, error) {
        result, err := parseMyResult(reply.String())
        if err != nil {
            return 0, MyResult{}, thinker.Feedback("Fix the output format:", err.Error())
        }
        return 1.0, result, nil
    }),

    // Reasoner: retry up to 3 times, then return
    reasoner.NewEpoch(3, reasoner.From(func(s thinker.State[MyResult]) (thinker.Phase, chatter.Message, error) {
        if s.Confidence >= 0.8 {
            return thinker.AGENT_RETURN, nil, nil
        }
        return thinker.AGENT_REFINE, nil, nil
    })),
)
```

### 2.4 Choosing the right agent type

```
Need raw LLM output, no parsing?          → Prompter
Need to decode into a typed struct?       → Prompter + custom decoder  (if one-shot)
                                            Automata                   (if multi-step)
Need to call external tools?              → Manifold   (LLM drives tool use)
                                            Automata   (app drives tool use)
Need persistent memory across calls?      → Automata with memory.Stream
Need to orchestrate multi-step workflows? → nanobot (Seq, ThinkReAct, Reflect)
```

---

## 3. Agent development

Package: `github.com/kshard/thinker/agent/nanobot`

`nanobot` is the recommended toolkit for building production multi-agent workflows. It provides:
- A shared `Runtime` that wires the file system, LLM registry, tool registry, and progress output
- Five composable bot patterns that cover the most common agentic workflows
- Prompt files that separate natural-language instructions from Go code

### 3.1 nanobot Runtime

```go
type Runtime struct {
    FileSystem fs.FS
    LLMs       LLMs         // interface: Model(name string) (chatter.Chatter, bool)
    Registry   *command.Registry
    Chalk      Chalk         // interface: progress reporter
}

func NewRuntime(fs fs.FS, llms LLMs) *Runtime
func (rt *Runtime) WithRegistry(r *command.Registry) *Runtime
func (rt *Runtime) WithStdout(c Chalk) *Runtime
```

Create one `Runtime` per application and thread it through every bot constructor:

```go
// Implement LLMs
type MyLLMs struct{ base chatter.Chatter }
func (m *MyLLMs) Model(name string) (chatter.Chatter, bool) {
    if name == "base" || name == "" { return m.base, true }
    return nil, false
}

llm, _ := autoconfig.New("thinker")

rt := nanobot.NewRuntime(
    os.DirFS("prompts"),      // directory containing .md prompt files
    &MyLLMs{base: llm},
)

// Optionally attach tools and progress output
registry := command.NewRegistry()
registry.ConnectCmd("fs", []string{"mcp-server-filesystem", "/data"})
rt = rt.WithRegistry(registry)
```

The `Chalk` interface is optional. Implement it to get structured progress output:

```go
type Chalk interface {
    Sub(context.Context) context.Context    // create child context (for nesting)
    Task(context.Context, string, ...any)    // announce a task
    Done(...string)                          // mark current task done
    Fail(error)                              // mark current task failed
    Printf(string, ...any)                   // log a message
}
```

A no-op default is used if `WithStdout` is not called.

### 3.2 Prompt files

A prompt file is a Markdown document with an optional YAML front-matter block. It serves as the specification for a `ReAct` agent.

```markdown
---
name: Classify sentiment          # human-readable label (used in logs)
runs-on: base                     # LLMs key; falls back to "base"
retry: 3                          # max ReAct iterations
debug: false                      # if true, log full LLM dialog to stderr
schema:
  input:                          # JSON Schema for the input (optional, for documentation)
    type: object
    properties:
      text: { type: string }
  reply:                          # JSON Schema for the reply (drives decoder)
    type: object
    required: [sentiment, score]
    properties:
      sentiment:
        type: string
        enum: [positive, negative, neutral]
      score:
        type: number
        minimum: 0
        maximum: 1
servers:                          # MCP servers to connect for this prompt only
  - type: url
    name: kb
    url: https://kb.example.com/mcp
  - type: cmd
    name: calc
    command: [python3, tools/calc.py]
---
Analyse the sentiment of the following text and return a JSON object.

Text: {{.Text}}
```

**Template variables:** the prompt body is a Go `text/template`. The input value `A` is the dot (`.`). If `A` is a struct, use `{{.FieldName}}`; if `A` is a string, use `{{.}}`.

**Schema-driven decoding:** when `schema.reply` is provided and `B` implements `encoding.TextUnmarshaler`, `ReAct` automatically validates the JSON reply against the schema and injects the schema into the prompt instructions.

### 3.3 ReAct — tool-use agent from a file

```go
func NewReAct[A, B any](rt *Runtime, file string) (*ReAct[A, B], error)
func MustReAct[A, B any](rt *Runtime, file string) *ReAct[A, B]
```

`ReAct[A, B]` loads the prompt template from `rt.FileSystem` at path `file`, connects any servers declared in the front-matter, resolves the LLM from `rt.LLMs.Model(prompt.RunsOn)`, and wraps a `Manifold[A, B]`.

```go
// B must implement encoding.TextUnmarshaler for schema-driven decoding,
// or be *chatter.Reply / string for raw output.
type SentimentResult struct {
    Sentiment string  `json:"sentiment"`
    Score     float64 `json:"score"`
}

func (r *SentimentResult) UnmarshalText(b []byte) error {
    return json.Unmarshal(b, r)
}

bot := nanobot.MustReAct[string, *SentimentResult](rt, "classify_sentiment.md")
result, err := bot.Prompt(ctx, "I absolutely love this product!")
```

**Debugging:** set `debug: true` in the front-matter to log the full JSON LLM dialog to stderr.

### 3.4 Seq — two-step pipeline

```go
func NewSeq[S, A, B any](rt *Runtime, a Bot[S, A], b Bot[S, B]) (*Seq[S, A, B], error)
func MustSeq[S, A, B any](rt *Runtime, a Bot[S, A], b Bot[S, B]) *Seq[S, A, B]

func (s *Seq[S, A, B]) WithApply(func(S, A) S) *Seq[S, A, B]
func (s *Seq[S, A, B]) WithEffect(func(context.Context, S) (S, error)) *Seq[S, A, B]
```

`Seq` wires two bots over a shared blackboard state `S`:

```
step 1: a.Prompt(ctx, s) → A
step 2: s = apply(s, A)
step 3: s = effect(ctx, s)
step 4: b.Prompt(ctx, s) → B
return B
```

**Default `apply`:** uses `optics.ForProduct1[S, A]()` to resolve a lens automatically. This works when `S` has exactly one field of type `A`. Override with `WithApply` for any other shape.

```go
type Pipeline struct {
    Query   string           // input field
    Summary string           // output of step 1, input of step 2
}

summarizer := nanobot.MustReAct[Pipeline, string](rt, "summarize.md")
classifier := nanobot.MustReAct[Pipeline, string](rt, "classify.md")

pipe := nanobot.MustSeq(rt, summarizer, classifier).
    WithApply(func(s Pipeline, summary string) Pipeline {
        s.Summary = summary
        return s
    }).
    WithEffect(func(ctx context.Context, s Pipeline) (Pipeline, error) {
        // optional: persist intermediate summary, enforce invariants
        return s, nil
    })

result, err := pipe.Prompt(ctx, Pipeline{Query: "long document text..."})
```

### 3.5 ThinkReAct — plan then execute

```go
func NewThinkReAct[S, A, B any](rt *Runtime, think Bot[S, []A], react Bot[S, B]) (*ThinkReAct[S, A, B], error)
func MustThinkReAct[S, A, B any](rt *Runtime, think Bot[S, []A], react Bot[S, B]) *ThinkReAct[S, A, B]

func (p *ThinkReAct[S, A, B]) WithApply(func(S, A) S) *ThinkReAct[S, A, B]
func (p *ThinkReAct[S, A, B]) WithEffect(func(context.Context, S) (S, error)) *ThinkReAct[S, A, B]
```

`ThinkReAct` separates planning from execution:

```
planning:  think.Prompt(ctx, s) → []A
execution: for each a in []A:
               s' = apply(s, a)
               s' = effect(ctx, s')
               react.Prompt(ctx, s') → B
return []B
```

The `think` bot produces the full task list before any execution begins. This makes the plan inspectable and allows deterministic guardrails in `effect`.

```go
type WorkState struct {
    Request string
    Task    string   // current task name, set by apply
}

type Task struct{ Name string }

planner := nanobot.MustReAct[WorkState, []Task](rt, "plan.md")
executor := nanobot.MustReAct[WorkState, string](rt, "execute.md")

workflow := nanobot.MustThinkReAct(rt, planner, executor).
    WithApply(func(s WorkState, t Task) WorkState {
        s.Task = t.Name
        return s
    })

results, err := workflow.Prompt(ctx, WorkState{Request: "Migrate the database schema"})
// results[i] is the executor output for tasks[i]
```

Progress is reported to `rt.Chalk` (one `Task`/`Done` pair per planned item).

### 3.6 Reflect — judge-and-correct loop

```go
func NewReflect[S, A, B any](rt *Runtime, judge Bot[S, A], react Bot[S, B]) (*Reflect[S, A, B], error)
func MustReflect[S, A, B any](rt *Runtime, judge Bot[S, A], react Bot[S, B]) *Reflect[S, A, B]

func (r *Reflect[S, A, B]) WithAccept(func(S, A) (S, int)) *Reflect[S, A, B]
func (r *Reflect[S, A, B]) WithApply(func(S, B) S) *Reflect[S, A, B]
func (r *Reflect[S, A, B]) WithEffect(func(context.Context, S) (S, error)) *Reflect[S, A, B]
func (r *Reflect[S, A, B]) WithAttempts(int) *Reflect[S, A, B]
```

`Reflect` implements a review-and-correct loop:

```
loop (up to attempts times):
    verdict A = judge.Prompt(ctx, s)
    (s', decision) = accept(s, A)
    if decision > 0: return s'          // accepted
    correction B = react.Prompt(ctx, s')
    s = apply(s, B)
    s = effect(ctx, s)
return error (max attempts exceeded)
```

The `accept` function is the policy:
- Return `(s, positive int)` to accept — `s` is the final output state.
- Return `(s, negative int)` to reject — `s` should contain the judge's critique so the corrector knows why it was rejected.

```go
type Essay struct {
    Text     string
    Critique string   // set by accept when rejecting
}

reviewer := nanobot.MustReAct[Essay, string](rt, "review.md")
writer   := nanobot.MustReAct[Essay, string](rt, "rewrite.md")

loop := nanobot.MustReflect(rt, reviewer, writer).
    WithAccept(func(s Essay, verdict string) (Essay, int) {
        if strings.Contains(verdict, "APPROVED") {
            return s, 1  // accept
        }
        s.Critique = verdict
        return s, -1    // reject; embed critique for the writer
    }).
    WithApply(func(s Essay, rewrite string) Essay {
        s.Text = rewrite
        return s
    }).
    WithAttempts(3)

final, err := loop.Prompt(ctx, Essay{Text: "Draft essay..."})
```

The default `accept` always accepts (returns `(s, 1)`), so `Reflect` with no `WithAccept` behaves like a single judge call.

### 3.7 Jsonify — JSON array extraction

```go
func NewJsonify[A any](
    llm       chatter.Chatter,
    attempts  int,
    encoder   thinker.Encoder[A],
    validator func([]string) error,
) *Jsonify[A]
```

`Jsonify[A]` wraps `Automata` and forces the LLM to return a JSON array of strings. It automatically:
1. Injects 9 strict JSON formatting rules into the prompt.
2. Parses the reply with a regex that extracts the first `[…]` or `{…}` block.
3. Passes the parsed slice to `validator`; if validation fails, the error becomes feedback for the next attempt.
4. Aborts after `attempts` iterations.

Use for any task that requires extracting a flat list of strings from an LLM (e.g., enumerations, keyword extraction, classification labels).

```go
extractor := nanobot.NewJsonify(
    llm, 4,
    codec.FromEncoder(func(doc string) (chatter.Message, error) {
        var p chatter.Prompt
        p.WithTask("Extract all person names from the following text: %s", doc)
        return &p, nil
    }),
    func(names []string) error {
        if len(names) == 0 {
            return errors.New("no names found; provide at least one")
        }
        return nil
    },
)

names, err := extractor.Prompt(ctx, "Alice met Bob at the conference.")
// names = ["Alice", "Bob"]
```

### 3.8 Shared state pattern

When the encoder, decoder, and reasoner all need access to the same Go struct (e.g., configuration, accumulated results, running totals), implement them as methods on that struct:

```go
type Researcher struct {
    Topic   string
    Sources []string   // accumulated by decoder

    *agent.Automata[string, string]
}

func NewResearcher(llm chatter.Chatter, topic string) *Researcher {
    r := &Researcher{Topic: topic}
    r.Automata = agent.NewAutomata(
        llm,
        memory.NewStream(memory.INFINITE, "You are a research assistant."),
        codec.FromEncoder(r.encode),
        codec.FromDecoder(r.decode),
        reasoner.NewEpoch(5, reasoner.From(r.deduct)),
    )
    return r
}

func (r *Researcher) encode(q string) (chatter.Message, error) {
    var p chatter.Prompt
    p.WithTask("Research: %s. Known sources so far: %v", q, r.Sources)
    return &p, nil
}

func (r *Researcher) decode(reply *chatter.Reply) (float64, string, error) {
    // parse reply, accumulate sources
    r.Sources = append(r.Sources, extractSources(reply.String())...)
    return 1.0, reply.String(), nil
}

func (r *Researcher) deduct(s thinker.State[string]) (thinker.Phase, chatter.Message, error) {
    if len(r.Sources) >= 3 { return thinker.AGENT_RETURN, nil, nil }
    return thinker.AGENT_ASK, moreResearchPrompt(), nil
}
```

### 3.9 Composing agents

Every agent type (`Prompter`, `Manifold`, `Automata`, `ReAct`, `Seq`, `ThinkReAct`, `Reflect`, `Jsonify`) satisfies the `nanobot.Bot[S, A]` interface. They can be freely combined:

```go
// Nest a Seq inside a ThinkReAct
plannerBot := nanobot.MustReAct[State, []Task](rt, "plan.md")
step1      := nanobot.MustReAct[State, StepA](rt, "step1.md")
step2      := nanobot.MustReAct[State, StepB](rt, "step2.md")
pipeBot    := nanobot.MustSeq(rt, step1, step2)

// pipeBot is Bot[State, StepB] — plug it as the react bot of ThinkReAct
workflow := nanobot.MustThinkReAct(rt, plannerBot, pipeBot)
```

**Fan-out with goroutines:** `Automata`/`Manifold` instances are not safe for concurrent use within one session, but you can create one instance per goroutine:

```go
var wg sync.WaitGroup
results := make([]string, len(inputs))

for i, input := range inputs {
    wg.Add(1)
    go func(i int, input string) {
        defer wg.Done()
        // Each goroutine owns its own bot instance (and its own memory)
        bot := nanobot.MustReAct[string, string](rt, "process.md")
        results[i], _ = bot.Prompt(ctx, input)
    }(i, input)
}
wg.Wait()
```

**Shared `memory.Stream`:** if you want a pool of agents to share a single conversational history (e.g., for a multi-agent research team), create the `Stream` once and pass it to each `Automata`:

```go
shared := memory.NewStream(50, "You are part of a research team.")
agentA := agent.NewAutomata(llm, shared, encA, decA, reasonerA)
agentB := agent.NewAutomata(llm, shared, encB, decB, reasonerB)
```

`Stream` is thread-safe; agents will see each other's observations in their context window.

### 3.10 Deployment patterns

#### Stateless serverless (Lambda, Cloud Run)

Use `PromptOnce` (or `Prompter`) — no persistent state, every invocation is independent.

```go
// Handler re-creates the bot per invocation or keeps it across invocations
// with PromptOnce for isolation.
func handler(ctx context.Context, event Event) (Response, error) {
    result, err := bot.PromptOnce(ctx, event.Input)
    // ...
}
```

#### AWS Step Functions

Chain agents across Step Functions states using the [typestep library](https://github.com/fogfish/typestep). Each Lambda implements one agent, and Step Functions handles retries, parallel fan-out, and durable state. See [examples/07_aws_sfs](../examples/07_aws_sfs/) for a full working example.

#### Long-running service

Create one `Automata` per user session, keyed by session ID. Store the session's `memory.Stream` in an external store (Redis, DynamoDB) and restore it at the start of each request. Call `memory.Stream.Purge()` at session end to free memory.

---

## Appendix: package map

| Package                                    | Purpose                                                                                   |
| ------------------------------------------ | ----------------------------------------------------------------------------------------- |
| `github.com/kshard/thinker`                | Core interfaces: `Encoder`, `Decoder`, `Memory`, `Reasoner`, `Registry`, `State`, `Phase` |
| `github.com/kshard/thinker/agent`          | Three agent constructors: `Prompter`, `Manifold`, `Automata`                              |
| `github.com/kshard/thinker/agent/nanobot`  | Workflow patterns: `Runtime`, `ReAct`, `Seq`, `ThinkReAct`, `Reflect`, `Jsonify`          |
| `github.com/kshard/thinker/codec`          | Ready-made codecs: `EncoderID`, `DecoderID`, `String`, `FromEncoder`, `FromDecoder`       |
| `github.com/kshard/thinker/memory`         | Memory implementations: `Void`, `Stream`                                                  |
| `github.com/kshard/thinker/reasoner`       | Reasoner implementations: `Void`, `From`, `Epoch`                                         |
| `github.com/kshard/thinker/command`        | MCP tool registry: `Registry`, `ConnectCmd`, `ConnectUrl`, `Attach`                       |
| `github.com/kshard/thinker/prompt`         | Prompt file parser (YAML front-matter + Go template)                                      |
| `github.com/kshard/thinker/prompt/jsonify` | JSON extraction helpers used by `Jsonify`                                                 |
