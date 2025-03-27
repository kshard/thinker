//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package agent

import (
	"fmt"

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
		reasoner.NewEpoch(4, reasoner.From(w.deduct)),
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

func (w *Worker[A]) deduct(state thinker.State[thinker.CmdOut]) (thinker.Phase, chatter.Prompt, error) {
	// the registry has failed to execute command, we have to supply the feedback to LLM
	if state.Feedback != nil && state.Confidence < 1.0 {
		var prompt chatter.Prompt
		prompt.WithTask("Refine the previous prompt using the feedback below.")
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
