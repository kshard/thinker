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
	"github.com/kshard/thinker/memory"
)

type Manifold[A, B any] struct {
	llm      chatter.Chatter
	memory   thinker.Memory
	encoder  thinker.Encoder[A]
	decoder  thinker.Decoder[B]
	registry thinker.Registry
}

func NewManifold[A, B any](
	llm chatter.Chatter,
	encoder thinker.Encoder[A],
	decoder thinker.Decoder[B],
	registry thinker.Registry,
) *Manifold[A, B] {
	return &Manifold[A, B]{
		llm:      llm,
		memory:   memory.NewStream(-1, ""),
		encoder:  encoder,
		decoder:  decoder,
		registry: registry,
	}
}

func (manifold *Manifold[A, B]) WithMemory(memory thinker.Memory) *Manifold[A, B] {
	manifold.memory = memory
	return manifold
}

func (manifold *Manifold[A, B]) Prompt(ctx context.Context, input A, opt ...chatter.Opt) (B, error) {
	var nul B

	prompt, err := manifold.encoder.Encode(input)
	if err != nil {
		return nul, thinker.ErrCodec.With(err)
	}

	opt = append(opt, manifold.registry.Context())

	for {
		shortMemory := manifold.memory.Context(prompt)
		reply, err := manifold.llm.Prompt(ctx, shortMemory, opt...)
		if err != nil {
			return nul, thinker.ErrLLM.With(err)
		}
		manifold.memory.Commit(thinker.NewObservation(prompt, reply))

		switch reply.Stage {
		case chatter.LLM_RETURN:
			_, ret, err := manifold.decoder.Decode(reply)
			if err != nil {
				var feedback chatter.Content
				if ok := errors.As(err, &feedback); !ok {
					return nul, err
				}

				var fprompt chatter.Prompt
				fprompt.With(feedback)
				prompt = &fprompt

				continue
			}
			return ret, nil
		case chatter.LLM_INCOMPLETE:
			_, ret, err := manifold.decoder.Decode(reply)
			if err != nil {
				var feedback chatter.Content
				if ok := errors.As(err, &feedback); !ok {
					return nul, err
				}

				var fprompt chatter.Prompt
				fprompt.With(feedback)
				prompt = &fprompt

				continue
			}
			return ret, nil
		case chatter.LLM_INVOKE:
			stage, answer, err := manifold.registry.Invoke(reply)
			if err != nil {
				var feedback chatter.Content
				if ok := errors.As(err, &feedback); !ok {
					return nul, thinker.ErrCmd.With(err)
				}

				var fprompt chatter.Prompt
				fprompt.With(feedback)
				prompt = &fprompt

				continue
			}
			switch stage {
			case thinker.AGENT_RETURN:
				_, ret, err := manifold.decoder.Decode(&chatter.Reply{Content: []chatter.Content{answer}})
				if err != nil {
					var feedback chatter.Content
					if ok := errors.As(err, &feedback); !ok {
						return nul, thinker.ErrCmd.With(err)
					}

					var fprompt chatter.Prompt
					fprompt.With(feedback)
					prompt = &fprompt

					continue
				}
				return ret, nil
			case thinker.AGENT_ABORT:
				return nul, thinker.ErrAborted
			default:
				prompt = answer
			}
		default:
			return nul, thinker.ErrAborted
		}
	}
}
