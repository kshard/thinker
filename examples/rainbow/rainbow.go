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
	"strings"

	"github.com/kshard/chatter"
	"github.com/kshard/chatter/bedrock"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/prompt/jsonify"
	"github.com/kshard/thinker/reasoner"
)

// Ask LLMs about colors of rainbow
func encode(any) (prompt chatter.Prompt, err error) {
	prompt.WithTask("Return colors of the rainbow.")

	prompt.With(
		chatter.Guide("Use the context and conversation history to find the right answer."),
	)

	// Injects requirments for LLM to return json array of strings
	jsonify.Strings.Harden(&prompt)
	return
}

// Parse LLMs response into sequence of colors
func decode(reply chatter.Reply) (float64, []string, error) {
	var seq []string
	if err := jsonify.Strings.Decode(reply, &seq); err != nil {
		return 0.0, nil, err
	}

	if err := validate(seq); err != nil {
		return 0.1, nil, err
	}

	return 1.0, seq, nil
}

// Validate sequence of colors, expecting invisible spectrum.
func validate(seq []string) error {
	for _, x := range seq {
		if strings.ToLower(x) == "ultraviolet" {
			return nil
		}
	}

	return thinker.Feedback(
		`Improve the response based on feedback:`,
		fmt.Sprintf("Sequence of rainbow colors %v is missing colors from invisible spectrum.", seq),
		"You have to includes colors of invisible spectrum into response despite if visible or not to the human.",
	)
}

// deduct new goal for the agent to pursue.
func deduct(state thinker.State[any, []string]) (thinker.Phase, chatter.Prompt, error) {
	// Provide feedback to LLM if there are no confidence about the results
	if state.Feedback != nil && state.Confidence < 1.0 {
		var prompt chatter.Prompt
		prompt.WithTask("Refine the previous request using the feedback below.")
		prompt.With(state.Feedback)
		return thinker.AGENT_REFINE, prompt, nil
	}

	// We have sufficient confidence, return results
	return thinker.AGENT_RETURN, chatter.Prompt{}, nil
}

func main() {
	// create instance of LLM client
	llm, err := bedrock.New(
		bedrock.WithLLM(bedrock.LLAMA3_0_8B_INSTRUCT),
		bedrock.WithRegion("us-west-2"),
	)
	if err != nil {
		panic(err)
	}

	// We create an agent that takes string (sentence) and returns string (anagram).
	agt := agent.NewAutomata(
		// enable debug output for LLMs dialog
		chatter.NewDebugger(os.Stdout, llm),

		// Configures memory for the agent. Typically, memory retains all of
		// the agent's observations. Here, we use a stream memory that holds all observations.
		memory.NewStream(memory.INFINITE, "You are agent who remembers and uses earlier chat history."),

		// Configures the reasoner, which determines the agent's next actions and prompts.
		// Here, we use custom (app specific) reasoner. The agent is restricted to execute
		// 4 itterattions before it fails.
		reasoner.NewEpoch(4, reasoner.From(deduct)),

		// Configures the encoder to transform input of type A into a `chatter.Prompt`.
		// Here, we use an encoder that builds prompt.
		codec.FromEncoder(encode),

		// Configure the decoder to transform output of LLM into type B.
		// Here, we use custom (app specific) codec that parses LLM response into []string.
		codec.FromDecoder(decode),
	)

	// We ask agent about the rainbow colors.
	// Firstly agent respond with 7 colors, well-known colors visible to human eye.
	// It fails validation because we are looking for "ultraviolet" color.
	// We provide the feedback to agent, it learns and return correct list of colors.
	val, err := agt.Prompt(context.Background(), nil)
	fmt.Printf("\n\n==> Err: %v\nColors: %+v\n", err, val)

	// We ask same agent for repetivite task. It immediatly retrun correct
	// list of colors becuase it remeber the right answer from previous conversation.
	val, err = agt.Prompt(context.Background(), nil)
	fmt.Printf("\n\n==> Err: %v\nColors: %+v\n", err, val)
}
