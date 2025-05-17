package agent

import (
	"context"
	"errors"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

type Manifold[A, B any] struct {
	llm      chatter.Chatter
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
		encoder:  encoder,
		decoder:  decoder,
		registry: registry,
	}
}

func (manifold *Manifold[A, B]) Prompt(ctx context.Context, input A, opt ...chatter.Opt) (B, error) {
	var nul B

	switch v := manifold.llm.(type) {
	case interface{ ResetQuota() }:
		v.ResetQuota()
	}

	prompt, err := manifold.encoder.Encode(input)
	if err != nil {
		return nul, thinker.ErrCodec.With(err)
	}

	opt = append(opt, manifold.registry.Context())
	memory := []chatter.Message{prompt}

	for {
		reply, err := manifold.llm.Prompt(ctx, memory, opt...)
		if err != nil {
			return nul, thinker.ErrLLM.With(err)
		}

		switch reply.Stage {
		case chatter.LLM_RETURN:
			_, ret, err := manifold.decoder.Decode(reply)
			if err != nil {
				var feedback chatter.Content
				if ok := errors.As(err, &feedback); !ok {
					return nul, err
				}

				var prompt chatter.Prompt
				prompt.With(feedback)
				memory = append(memory, reply, &prompt)
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

				var prompt chatter.Prompt
				prompt.With(feedback)
				memory = append(memory, reply, &prompt)
				continue
			}
			return ret, nil
		case chatter.LLM_INVOKE:
			stage, answer, err := manifold.registry.Invoke(reply)
			if err != nil {
				return nul, thinker.ErrCmd.With(err)
			}
			switch stage {
			case thinker.AGENT_RETURN:
				_, ret, err := manifold.decoder.Decode(&chatter.Reply{Content: []chatter.Content{answer}})
				if err != nil {
					return nul, thinker.ErrCmd.With(err)
				}
				return ret, nil
			case thinker.AGENT_ABORT:
				return nul, thinker.ErrAborted
			default:
				memory = append(memory, reply, answer)
			}
		default:
			return nul, thinker.ErrAborted
		}
	}
}
