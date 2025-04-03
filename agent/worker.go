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
	"github.com/kshard/thinker/command"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/reasoner"
)

// The agent tailored for executing prompt driven workflow.
type Worker[A any] struct {
	*Automata[A, thinker.CmdOut]
	encoder  thinker.Encoder[A]
	registry *command.Registry
}

func NewWorker[A any](
	llm chatter.Chatter,
	attempts int,
	encoder thinker.Encoder[A],
	registry *command.Registry,
) *Worker[A] {
	registry.Register(command.Return())

	w := &Worker[A]{encoder: encoder, registry: registry}
	w.Automata = NewAutomata(
		llm,

		// Configures memory for the agent. Typically, memory retains all of
		// the agent's observations. Here, we use an infinite stream memory,
		// recalling all observations.
		memory.NewStream(memory.INFINITE, `
			You are automomous agent who uses tools to perform required tasks.
			You are using and remember context from earlier chat history to execute the task.
		`),

		// Configures the reasoner, which determines the agent's next actions and prompts.
		// Here, we use a sequence of command reasoner, it assumes that input prompt is
		// the workflow based on command. LLM guided to execute entire workflow.
		reasoner.NewEpoch(attempts, reasoner.NewCmdSeq()),

		// Configures the encoder to transform input of type A into a `chatter.Prompt`.
		// Here, it is defined by application
		codec.FromEncoder(w.encode),

		// Configure the decoder to transform output of LLM into type B.
		// The registry knows how to interpret the LLM's reply and executed the command.
		registry,
	)

	return w
}

func (w *Worker[A]) encode(in A) (prompt chatter.Prompt, err error) {
	prompt, err = w.encoder.Encode(in)
	if err == nil {
		w.registry.Harden(&prompt)
	}

	return
}
