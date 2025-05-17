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
	"github.com/kshard/chatter/llm/autoconfig"
	"github.com/kshard/thinker/agent"
)

// This function is core in the example. It takes input (the sentence)
// and generate prompt function that guides LLMs on how to create anagram.
func anagram(expr string) (chatter.Message, error) {
	var prompt chatter.Prompt

	prompt.WithTask("Create anagram using the phrase: %s", expr)

	// instruct LLM about anagram generation
	prompt.WithRules(
		"Strictly adhere to the following requirements when generating a response.",
		"The output must be the resulting anagram only.",
	)

	// Gives the example of input and expected output
	prompt.WithExample("Madam Curie", "Radium came")

	return &prompt, nil
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
