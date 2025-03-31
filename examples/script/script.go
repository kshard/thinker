//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kshard/chatter"
	"github.com/kshard/chatter/aio"
	"github.com/kshard/chatter/llm/bedrock"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/reasoner"
)

// declares command registry
var registry = command.NewRegistry()

// Define the workflow for LLM
func encode(q string) (prompt chatter.Prompt, err error) {
	prompt.WithTask(`
		Use available tools to complete the workflow:
		(1) Create 5 files one by one with few lines of random text, at least one line shall contain "%s".
		(2) Use available tools to find files containing the string: "%s".
		(3) Replace the only found string with "XXXXX".
		(4) Validate completion of task by checking "%s" in the files and fix your self if it still exists.`, q, q, q)

	// Inject tools
	registry.Harden(&prompt)
	return
}

// deduct new goal for the agent to pursue.
// Note, the agent uses registry as decoder therefore agent  is string -> thinker.CmdOut
func deduct(state thinker.State[thinker.CmdOut]) (thinker.Phase, chatter.Prompt, error) {
	// the registry has failed to execute command, we have to supply the feedback to LLM
	if state.Feedback != nil && state.Confidence < 1.0 {
		var prompt chatter.Prompt
		prompt.WithTask("Refine the previous workflow step using the feedback below.")
		prompt.With(state.Feedback)

		return thinker.AGENT_REFINE, prompt, nil
	}

	// the workflow has successfully completed
	// Note: pseudo-command return is executed
	if state.Reply.Cmd == command.RETURN {
		return thinker.AGENT_RETURN, chatter.Prompt{}, nil
	}

	// the workflow step is completed
	if state.Reply.Cmd == command.BASH {
		var prompt chatter.Prompt
		prompt.WithTask("Continue the workflow execution.")
		prompt.With(
			chatter.Blob("The command has returned:\n", state.Reply.Output),
		)

		return thinker.AGENT_ASK, prompt, nil
	}

	return thinker.AGENT_ABORT, chatter.Prompt{}, fmt.Errorf("unknown state")
}

func main() {
	// enable `shell` command for the agent, the command is sandbox to the dir.
	registry.Register(command.Bash("MacOS", "/tmp/script"))

	// enable pseudo tool for LLM to emphasis completion of the task
	registry.Register(command.Return())

	// create instance of LLM client
	llm, err := bedrock.New(
		bedrock.WithLLM(bedrock.LLAMA3_1_70B_INSTRUCT),
		bedrock.WithRegion("us-west-2"),
	)
	if err != nil {
		panic(err)
	}

	// We create an agent that executes the workflow.
	agt := agent.NewAutomata(
		// enable debug output for LLMs dialog
		aio.NewLogger(os.Stdout, llm),

		// Configures memory for the agent. Typically, memory retains all of
		// the agent's observations. Here, we use a stream memory that holds all observations.
		memory.NewStream(memory.INFINITE, `
			You are automomous agent who uses tools to perform required tasks.
			You are using and remember context from earlier chat history to execute the task.
		`),

		// Configures the reasoner, which determines the agent's next actions and prompts.
		// Here, we use custom (app specific) reasoner. The agent is restricted to execute
		// 4 itterattions before it fails.
		reasoner.NewEpoch(4, reasoner.From(deduct)),

		// Configures the encoder to transform input of type A into a `chatter.Prompt`.
		// Here, we use an encoder that builds prompt.
		codec.FromEncoder(encode),

		// Configure the decoder to transform output of LLM into type B.
		// Here, we use the tool registry to "decode" output into command call.
		registry,
	)

	// Execute agent
	_, err = agt.Prompt(context.Background(), "Hello World")
	fmt.Printf("\n\n==> Err: %v", err)
}
