//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package agent

import (
	"context"
	"errors"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// Generic agent automata
type Automata[A, B any] struct {
	llm      chatter.Chatter
	memory   thinker.Memory
	reasoner thinker.Reasoner[B]
	encoder  thinker.Encoder[A]
	decoder  thinker.Decoder[B]
}

func NewAutomata[A, B any](
	llm chatter.Chatter,
	memory thinker.Memory,
	encoder thinker.Encoder[A],
	decoder thinker.Decoder[B],
	reasoner thinker.Reasoner[B],
) *Automata[A, B] {
	return &Automata[A, B]{
		llm:      llm,
		memory:   memory,
		reasoner: reasoner,
		encoder:  encoder,
		decoder:  decoder,
	}
}

// Purge automata's memory
func (automata *Automata[A, B]) Purge() {
	automata.reasoner.Purge()
	automata.memory.Purge()
}

// Forget the agent state and prompt within a new session
func (automata *Automata[A, B]) PromptOnce(ctx context.Context, input A, opt ...chatter.Opt) (B, error) {
	automata.Purge()
	return automata.Prompt(ctx, input, opt...)
}

// Prompt agent
func (automata *Automata[A, B]) Prompt(ctx context.Context, input A, opt ...chatter.Opt) (B, error) {
	var nul B
	state := thinker.State[B]{Phase: thinker.AGENT_ASK, Epoch: 0}

	switch v := automata.llm.(type) {
	case interface{ ResetQuota() }:
		v.ResetQuota()
	}

	prompt, err := automata.encoder.Encode(input)
	if err != nil {
		return nul, err
	}
	shortMemory := automata.memory.Context(prompt)

	for {
		reply, err := automata.llm.Prompt(ctx, shortMemory)
		if err != nil {
			return nul, thinker.ErrLLM.With(err)
		}

		state.Confidence, state.Reply, err = automata.decoder.Decode(reply)
		if err != nil {
			if ok := errors.As(err, &state.Feedback); !ok {
				return nul, err
			}
		}

		state.Epoch++
		if state.Phase != thinker.AGENT_RETRY {
			automata.memory.Commit(thinker.NewObservation(prompt, reply))
		}

		phase, request, err := automata.reasoner.Deduct(state)
		if err != nil {
			return nul, err
		}

		switch phase {
		case thinker.AGENT_ASK:
			state = thinker.State[B]{Phase: thinker.AGENT_ASK, Epoch: 0}
			prompt = request
			shortMemory = automata.memory.Context(prompt)
			continue
		case thinker.AGENT_RETURN:
			return state.Reply, nil
		case thinker.AGENT_RETRY:
			state.Phase = phase
			continue
		case thinker.AGENT_REFINE:
			state.Phase = phase
			prompt = request
			shortMemory = automata.memory.Context(prompt)
		case thinker.AGENT_ABORT:
			return nul, thinker.ErrAborted.With(err)
		default:
			return nul, thinker.ErrAborted
		}
	}
}
