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
		memory.NewStream(memory.INFINITE, `
			You are automomous agent who uses tools to perform required tasks.
			You are using and remember context from earlier chat history to execute the task.
		`),
		reasoner.NewEpoch(attempts, reasoner.NewCmdSeq()),
		codec.FromEncoder(w.encode),
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
