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
	"github.com/kshard/chatter/llm/autoconfig"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
)

func encode(q string) (prompt chatter.Prompt, err error) {
	prompt.WithTask(`
		Use available tools to complete the workflow:
		(1) Create 5 files one by one with few lines of random text, at least one line shall contain "%s".
		(2) Use available tools to find files containing the string: "%s".
		(3) Replace the only found string with "XXXXX".
		(4) Validate completion of task by checking "%s" in the files and fix your self if it still exists.`, q, q, q)

	return
}

func main() {
	// enable `shell` command for the agent, the command is sandbox to the dir.
	registry := command.NewRegistry()
	registry.Register(command.Bash("MacOS", "/tmp/script"))

	// create instance of LLM API, see doc/HOWTO.md for details
	llm, err := autoconfig.New("thinker")
	if err != nil {
		panic(err)
	}

	// We create an agent that executes the workflow.
	agt := agent.NewWorker(
		// enable debug output for LLMs dialog
		aio.NewLogger(os.Stdout, llm),

		// Number of attempts to resolve the task
		4,

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
