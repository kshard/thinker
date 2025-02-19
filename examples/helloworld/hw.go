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

	"github.com/kshard/chatter"
	"github.com/kshard/chatter/bedrock"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/reasoner"
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
	// create instance of LLM client
	llm, err := bedrock.New(
		bedrock.WithLLM(bedrock.LLAMA3_1_70B_INSTRUCT),
		bedrock.WithRegion("us-west-2"),
	)
	if err != nil {
		panic(err)
	}

	// We create an agent that takes string (sentence) and returns string (anagram).
	agt := agent.NewAutomata(llm,
		// Configures memory for the agent. Typically, memory retains all of
		// the agent's observations. Here, we use a void memory, meaning no
		// observations are retained.
		memory.NewVoid(),

		// Configures the reasoner, which determines the agent's next actions and prompts.
		// Here, we use a void reasoner, meaning no reasoning is performed—the agent
		// simply returns the result.
		reasoner.NewVoid[string, string](),

		// Configures the encoder to transform input of type A into a `chatter.Prompt`.
		// Here, we use an encoder that converts string expressions into prompt.
		codec.FromEncoder(anagram),

		// Configure the decoder to transform output of LLM into type B.
		// Here, we use the identity decoder that returns LLMs output as-is.
		codec.DecoderID,
	)

	// Evaluate expression and receive the result
	val, err := agt.Prompt(context.Background(), "a gentleman seating on horse")
	fmt.Printf("==> %v\n%+v\n", err, val)
}
