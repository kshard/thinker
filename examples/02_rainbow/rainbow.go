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
	"github.com/kshard/chatter/aio"
	"github.com/kshard/chatter/llm/autoconfig"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
)

// Ask LLMs about colors of rainbow
func encode(any) (prompt *chatter.Prompt, err error) {
	prompt = new(chatter.Prompt)
	prompt.WithTask("Return colors of the rainbow.")

	prompt.With(
		chatter.Guide("Use the context and conversation history to find the right answer."),
	)

	return
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

func main() {
	// create instance of LLM API, see doc/HOWTO.md for details
	llm, err := autoconfig.New("thinker")
	if err != nil {
		panic(err)
	}

	agt := agent.NewJsonify(
		// enable debug output for LLMs dialog
		aio.NewLogger(os.Stdout, llm),

		// attempts to request JSON
		4,

		// Configures the encoder to transform input of type A into a `chatter.Prompt`.
		// Here, we use an encoder that builds prompt.
		codec.FromEncoder(encode),

		// Validator function, checks correctness of response
		validate,
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
