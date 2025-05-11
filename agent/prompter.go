//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package agent

import (
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/reasoner"
)

// Prompter is memoryless and stateless agent, implementing request/response to LLMs.
type Prompter[A any] struct {
	*Automata[A, string]
}

func NewPrompter[A any](llm chatter.Chatter, f func(A) (*chatter.Prompt, error)) *Prompter[A] {
	w := &Prompter[A]{}
	w.Automata = NewAutomata(
		llm,

		// Configures memory for the agent. Typically, memory retains all of
		// the agent's observations. Here, we use a void memory, meaning no
		// observations are retained.
		memory.NewVoid(`You are automomous agent who perform tasks defined in the prompt.`),

		// Configures the encoder to transform input of type A into a `chatter.Prompt`.
		// Here, we use an encoder that converts input into prompt.
		codec.FromEncoder(f),

		// Configure the decoder to transform output of LLM into type B.
		// Here, we use the identity decoder that returns LLMs output as-is.
		codec.DecoderID,

		// Configures the reasoner, which determines the agent's next actions and prompts.
		// Here, we use a void reasoner, meaning no reasoning is performed—the agent
		// simply returns the result.
		reasoner.NewVoid[string](),
	)

	return w
}
