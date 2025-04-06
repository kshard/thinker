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
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/prompt/jsonify"
	"github.com/kshard/thinker/reasoner"
)

// Jsonify implementing request/response to LLMs, forcing the response to be JSON array.
type Jsonify[A any] struct {
	*Automata[A, []string]
	encoder   thinker.Encoder[A]
	validator func([]string) error
}

func NewJsonify[A any](
	llm chatter.Chatter,
	attempts int,
	encoder thinker.Encoder[A],
	validator func([]string) error,
) *Jsonify[A] {
	w := &Jsonify[A]{encoder: encoder, validator: validator}
	w.Automata = NewAutomata(llm,

		// Configures memory for the agent. Typically, memory retains all of
		// the agent's observations. Here, we use an infinite stream memory,
		// recalling all observations.
		memory.NewStream(memory.INFINITE, `
			You are automomous agent who perform required tasks, providing results in JSON.
		`),

		// Configures the encoder to transform input of type A into a `chatter.Prompt`.
		// Here, it is defined by application
		codec.FromEncoder(w.encode),

		// Configure the decoder to transform output of LLM into type B.
		// Here, we use the identity decoder that returns LLMs output as-is.
		codec.FromDecoder(w.decode),

		// Configures the reasoner, which determines the agent's next actions and prompts.
		// Here, we use a sequence of command reasoner, it assumes that input prompt is
		// the workflow based on command. LLM guided to execute entire workflow.
		reasoner.NewEpoch(attempts, reasoner.From(w.deduct)),
	)

	return w
}

func (w *Jsonify[A]) encode(in A) (prompt chatter.Prompt, err error) {
	prompt, err = w.encoder.Encode(in)
	if err == nil {
		jsonify.Strings.Harden(&prompt)
	}

	return
}

func (w *Jsonify[A]) decode(reply chatter.Reply) (float64, []string, error) {
	var seq []string
	if err := jsonify.Strings.Decode(reply, &seq); err != nil {
		return 0.0, nil, err
	}

	if err := w.validator(seq); err != nil {
		return 0.1, nil, err
	}

	return 1.0, seq, nil
}

func (w *Jsonify[A]) deduct(state thinker.State[[]string]) (thinker.Phase, chatter.Prompt, error) {
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
