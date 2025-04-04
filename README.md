<p align="center">
  <img src="./doc/thinker-4.svg" height="220" />
  <h3 align="center">thinker</h3>
  <p align="center"><strong>LLM generative agents for Golang</strong></p>

  <p align="center">
    <!-- Version -->
    <a href="https://github.com/kshard/thinker/releases">
      <img src="https://img.shields.io/github/v/tag/kshard/thinker?label=version" />
    </a>
    <!-- Documentation -->
    <a href="https://pkg.go.dev/github.com/kshard/thinker">
      <img src="https://pkg.go.dev/badge/github.com/kshard/thinker" />
    </a>
    <!-- Build Status  -->
    <a href="https://github.com/kshard/thinker/actions/">
      <img src="https://github.com/kshard/thinker/workflows/build/badge.svg" />
    </a>
    <!-- GitHub -->
    <a href="https://github.com/kshard/thinker">
      <img src="https://img.shields.io/github/last-commit/kshard/thinker.svg" />
    </a>
    <!-- Coverage -->
    <a href="https://coveralls.io/github/kshard/thinker?branch=main">
      <img src="https://coveralls.io/repos/github/kshard/thinker/badge.svg?branch=main" />
    </a>
    <!-- Go Card -->
    <a href="https://goreportcard.com/report/github.com/kshard/thinker">
      <img src="https://goreportcard.com/badge/github.com/kshard/thinker" />
    </a>
  </p>
</p>

---

The library enables development of LLM-based generative agents for Golang. 

## Inspiration

The generative agents autonomously generate output as a reaction on input, past expereince and current environment. Agents records obervations and reason over with natural language description, taking advantage of LLMs.

* [Generative Agents: Interactive Simulacra of Human Behavior](https://arxiv.org/pdf/2304.03442)
* [LLM Reasoner and Automated Planner: A new NPC approach](https://arxiv.org/pdf/2501.10106)

In this library, an agent is defined as a side-effect function `ƒ: A ⟼ B`, which takes a Golang type `A` as input and autonomously produces an output `B`, while retaining memory of past experiences.

- [Inspiration](#inspiration)
- [Getting started](#getting-started)
- [Quick example](#quick-example)
- [Agent Architecture](#agent-architecture)
  - [Memory](#memory)
  - [Reasoner](#reasoner)
  - [Encoder \& Decoder](#encoder--decoder)
  - [Commands \& Tools](#commands--tools)
  - [Agent profiles](#agent-profiles)
- [Agent composition (chaining)](#agent-composition-chaining)
- [FAQ](#faq)
- [How To Contribute](#how-to-contribute)
  - [commit message](#commit-message)
  - [bugs](#bugs)
- [License](#license)


## Getting started

The latest version of the library is available at `main` branch of this repository. All development, including new features and bug fixes, take place on the `main` branch using forking and pull requests as described in contribution guidelines. The stable version is available via Golang modules.

Running the examples you need access either to AWS Bedrock or OpenAI.  

## Quick example

See ["Hello World"](./examples/helloworld/hw.go) application as the quick start. The example agent is `ƒ: string ⟼ string` that takes the sentence and returns the anagram. [HowTo](./doc/HOWTO.md) gives support to bootstrap it. The library ships more [examples](./examples/) to demonstrate library's capabilities.

```go
package main

import (
  "context"
  "fmt"

  // LLMs toolkit
  "github.com/kshard/chatter"
  "github.com/kshard/chatter/llm/autoconfig"

  // Agents toolkit
  "github.com/kshard/thinker/agent"
)

// This function is core in the example. It takes input (the sentence)
// and generate prompt function that guides LLMs on how to create anagram.
func anagram(expr string) (prompt chatter.Prompt, err error) {
  prompt.
    WithTask("Create anagram using the phrase: %s", expr).
    With(
      // instruct LLM about anagram generation
      chatter.Rules(
        "Strictly adhere to the following requirements when generating a response.",
        "The output must be the resulting anagram only.",
      ),
    ).
    With(
      // Gives the example of input and expected output
      chatter.Example{
        Input: "Madam Curie",
        Reply: "Radium came",
      },
    )

  return
}

func main() {
  // create instance of LLM API, see doc/HOWTO.md for details
  llm, err := autoconfig.New("thinker")
  if err != nil {
    panic(err)
  }

	// Create an agent that takes string (sentence) and returns string (anagram).
	// Stateless and memory less agent is used
	agt := agent.NewPrompter(llm, anagram)

  // Evaluate expression and receive the result
  val, err := agt.Prompt(context.Background(), "a gentleman seating on horse")
  fmt.Printf("==> %v\n%+v\n", err, val)
}
```

## Agent Architecture

The `thinker` library provides toolkit for running agents with type-safe constraints. It is built on a pluggable architecture, allowing applications to define custom workflows. The diagram below emphasis core building blocks.

```mermaid
%%{init: {'theme':'neutral'}}%%
graph TD
    subgraph Interface
    A[Type A]
    B[Type B]
    end
    subgraph Memory
    S[Stream]
    end
    subgraph Commands
    C[Command]
    end
    subgraph Agent
    A --"01|input"--> E[Encoder]
    E --"02|prompt"--> G((Agent))
    G --"03|eval"--> L[LLM]
    G --"04|reply"--> D[Decoder]
    D -."05|exec".-> C
    C -."06|result".-> G
    G <--"07|observations" --> S
    G --"08|deduct"--> R[Reasoner]
    R <-."09|reflect".-> S
    R --"10|new goal"--> G
    D --"11|answer" --> B
    end
```

Following this architecture, the agent is assembled from building blocks as lego constructor:

```go
agent.NewAutomata(
  // LLM used by the agent to solve the task
  llm,

  // Configures memory for the agent. Typically, memory retains all of
  // the agent's observations. Here, we use a stream memory that holds all observations.
  memory.NewStream(memory.INFINITE, "You are agent..."),

  // Configures the reasoner, which determines the agent's next actions and prompts.
  reasoner.From(deduct),

  // Configures the encoder to transform input of type A into a `chatter.Prompt`.
  // Here, we use an encoder that builds prompt.
  codec.FromEncoder(encode),

  // Configure the decoder to transform output of LLM into type B.
  codec.FromDecoder(decode),
)
```

The [rainbow example](./examples/rainbow/rainbow.go) demonstrates a simple agent that effectively utilizes the depicted agent architecture to solve a task.

### Memory

[`Memory`](./memory.go) is core element of agents behaviour. It is a database that maintains a comprehensive record of an agent’s experience. It recalls observations and builds the context windows to be used for prompting.

The following [memory classes](https://pkg.go.dev/github.com/kshard/thinker/memory) are supported:
* *Void* does not retain any observations.
* *Stream* retains all of the agent's observations in the time ordered sequence. It is possible to re-call last N observations. 


### Reasoner

[`Reasoner`](./reasoner.go) serves as the goal-setting component in the architecture. It evaluates the agent's current state, performing either deterministic or non-deterministic analysis of immediate results and past experiences. Based on this assessment, it determines whether the goal has been achieved and, if not, suggests the best new goal for the agent to pursue. It maintain the following statemachine orchestrating the agent:

```go
const (
	// Agent is asking for new facts from LLM
	AGENT_ASK Phase = iota
	// Agent has a final result to return
	AGENT_RETURN
	// Agent should retry with the same context
	AGENT_RETRY
	// Agent should refine the prompt based on feedback
	AGENT_REFINE
	// Agent aborts processing due to unrecoverable error
	AGENT_ABORT
)
```

The following [reasoner classes](https://pkg.go.dev/github.com/kshard/thinker/reasoner) are supported:
* *Void* always sets a new goal to return results.
* *Cmd* sets the goal for agent to execute a single command and return the result if/when successful.
* *CmdSeq* sets the goal for reasoner to execute sequence of commands.
* *From* is fundamental constuctor for application specific reasoners.
* *Epoch* is pseudo reasoner, it limits number of itterations agent takes to solve a task.

```go
func deduct(state thinker.State[B]) (thinker.Phase, chatter.Prompt, error) {
  // define reasoning strategy
  return thinker.AGENT_RETURN, chatter.Prompt{}, nil 
}

reasoner.From(deduct)
```

### Encoder & Decoder

The type-safe agent interface `ƒ: A ⟼ B` is well-suited for composition and agent chaining. However, encoding and decoding application-specific types must be abstracted. To facilitate this, the library provides two key traits: [`Encoder`](./codec.go) for constructing prompts and [`Decoder`](./codec.go) for parsing and validating LLM responses.

The following [codec classes](https://pkg.go.dev/github.com/kshard/thinker/reasoner) are supported:
* *EncoderID* and *DecoderID* are identity codec taking and producing strings as-is.
* *FromEncoder* and *FromDecoder* are fundamental constuctor for application specific codecs.

```go
// Encode type A to prompt
func encoder[A any](A) (prompt chatter.Prompt, err error) { /* ... */ }

// Decode LLMs response to `B` 
func decoder[B any](reply chatter.Reply) (float64, B, error)  { /* ... */ }

codec.FromEncoder(encoder)
codec.FromDecoder(decoder)
```

### Commands & Tools

The `thinker` library enables the integration of external [tools and commands](./command.go) into the agent workflow. By design, a command is a function `Decoder[CmdOut]` that takes input from the LLM, executes it, validates and returns the output and any possible feedback - similary as you implement basic decoder.

When constructing a prompt, it is essential to include a section that "advertises" the available commands and the rules for using them. There is [a registry](./command/registry.go) that automates prompting and parsing of the response.

The [script example](./examples/script/script.go) demonstrates a simple agent that utilizes `bash` to generate and modify files on the local filesystem.

The following commands are supported
* *bash* execute bash script or single command
* *golang* execute golang code block
* *python* execute python code block

### Agent profiles

The application assembles agents from three elements: memory, reasoner and codecs. To simplfy the development, there are few built-in profiles that configures it:
* `Prompter` is ask-reply from LLM;
* `Worker` uses LLMs and external tools to solve the task. 


## Agent composition (chaining)

The `thinker` library does not provide built-in mechanisms for chaining agents. Instead, it encourages the use of ideomatic Go, pure functional chaining. 

The ["Chain" example](./examples/05_chain/chain.go) demonstrates off-the-shelf techniques for agents chaining.

The ["Text Processor" example](./06_text_processor/processor.go) demonstrates chaining agnet with file system I/O.

## FAQ

<details>
<summary>Do agents support concurrent execution?</summary>

This design does not support concurency on the purpose - the pure actor architecture is used. The agent follows a sequential decision-making loop:
* Inner for {} loop causes each step depends on the previous result to maintain conversational causal effect
* While memory is thread-safe and sharable among agents in the pipeline. It is not design to support multiple isolated session.
* LLM calls are synchronous.

To enable concurrency, the application have to implement worker pools.
</details>


<details>
<summary>How can an agent maintain a global state accessible to the encoder, decoder, and reasoner?</summary>

Use a struct with receiver methods to encapsulate state and provide direct access to the encoder, decoder, and reasoner. This keeps state management simple and idiomatic in Go.

```go
type Agent struct{
  // declare global state
}

func (*Agent) Encode(string) (prompt chatter.Prompt, err error) { /* ... */ }

func (*Agent) Decode(chatter.Reply) (float64, string, error) { /* ... */ }

func (*Agent) Deduct(thinker.State[string]) (thinker.Phase, chatter.Prompt, error) { /* ... */ }
```
</details>

<details>
<summary>How to deploy agents to AWS?</summary>
You might consider a AWS Serverless solution for hosting agents.
AWS Step Functions makes chaining of agents out-of-the-box, which is recommended approach.

You might consider [typestep library](https://github.com/fogfish/typestep) that provides a simplisitc approach for defining AWS Step Functions using a type-safe notation in Go.
</details>



## How To Contribute

The library is [MIT](LICENSE) licensed and accepts contributions via GitHub pull requests:

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Added some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request

The build and testing process requires [Go](https://golang.org) version 1.21 or later.

**build** and **test** library.

```bash
git clone https://github.com/kshard/thinker
cd thinker
go test ./...
```

### commit message

The commit message helps us to write a good release note, speed-up review process. The message should address two question what changed and why. The project follows the template defined by chapter [Contributing to a Project](http://git-scm.com/book/ch5-2.html) of Git book.

### bugs

If you experience any issues with the library, please let us know via [GitHub issues](https://github.com/kshard/chatter/issue). We appreciate detailed and accurate reports that help us to identity and replicate the issue. 


## License

[![See LICENSE](https://img.shields.io/github/license/kshard/thinker.svg?style=for-the-badge)](LICENSE)

