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

type Automata[A, B any] struct {
	llm      chatter.Chatter
	memory   thinker.Memory
	reasoner thinker.Reasoner[A, B]
	encoder  thinker.Encoder[A]
	decoder  thinker.Decoder[B]
}

func NewAutomata[A, B any](
	llm chatter.Chatter,
	memory thinker.Memory,
	reasoner thinker.Reasoner[A, B],
	encoder thinker.Encoder[A],
	decoder thinker.Decoder[B],
) *Automata[A, B] {
	return &Automata[A, B]{
		llm:      llm,
		memory:   memory,
		reasoner: reasoner,
		encoder:  encoder,
		decoder:  decoder,
	}
}

// TODO: Opts Temperature ToP
func (automata *Automata[A, B]) Prompt(ctx context.Context, input A) (B, error) {
	var nul B
	state := thinker.State[A, B]{Phase: thinker.AGENT_ASK, Epoch: 0, Input: input}

	prompt, err := automata.encoder.FMap(input)
	if err != nil {
		return nul, err
	}
	shortMemory := automata.memory.Context(prompt)

	for {
		reply, err := automata.llm.Prompt(ctx, shortMemory)
		if err != nil {
			return nul, err
		}

		state.Confidence, state.Reply, err = automata.decoder.FMap(reply)
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
			state = thinker.State[A, B]{Phase: thinker.AGENT_ASK, Epoch: 0}
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
			return nul, err
		default:
			return nul, thinker.ErrAbout
		}
	}
}
