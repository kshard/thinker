//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package agent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
)

//------------------------------------------------------------------------------
// Mock Registry
//------------------------------------------------------------------------------

type MockRegistry struct{}

func (r *MockRegistry) Context() chatter.Registry {
	return chatter.Registry{}
}

func (r *MockRegistry) Invoke(reply *chatter.Reply) (thinker.Phase, chatter.Message, error) {
	return thinker.AGENT_RETURN, nil, nil
}

//------------------------------------------------------------------------------
// Test Manifold
//------------------------------------------------------------------------------

func TestManifold(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		llm := &Mock{}
		registry := &MockRegistry{}
		manifold := agent.NewManifold(
			llm,
			codec.EncoderID,
			codec.DecoderID,
			registry,
		)

		reply, err := manifold.Prompt(context.Background(), "Hello, World!")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).ShouldNot(
			it.Nil(reply),
		)

		// The reply should contain the echoed prompt
		replyText := reply.String()
		it.Then(t).Should(
			it.String(replyText).Contain("Hello, World!"),
		)
	})

	t.Run("StringCodec", func(t *testing.T) {
		llm := &Mock{}
		registry := &MockRegistry{}
		manifold := agent.NewManifold(
			llm,
			codec.String,
			codec.String,
			registry,
		)

		result, err := manifold.Prompt(context.Background(), "Test input")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.String(result).Contain("Test input"),
		)
	})

	t.Run("CustomEncoder", func(t *testing.T) {
		encoder := codec.FromEncoder(
			func(input string) (chatter.Message, error) {
				var prompt chatter.Prompt
				prompt.WithTask("Custom: %s", input)
				return &prompt, nil
			},
		)

		llm := &Mock{}
		registry := &MockRegistry{}
		manifold := agent.NewManifold(
			llm,
			encoder,
			codec.DecoderID,
			registry,
		)

		reply, err := manifold.Prompt(context.Background(), "data")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.String(reply.String()).Contain("Custom: data"),
		)
	})

	t.Run("CustomDecoder", func(t *testing.T) {
		decoder := codec.FromDecoder(
			func(reply *chatter.Reply) (float64, string, error) {
				return 1.0, "Processed: " + reply.String(), nil
			},
		)

		llm := &Mock{}
		registry := &MockRegistry{}
		manifold := agent.NewManifold(
			llm,
			codec.EncoderID,
			decoder,
			registry,
		)

		result, err := manifold.Prompt(context.Background(), "input")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.String(result).Contain("Processed:"),
			it.String(result).Contain("input"),
		)
	})

	t.Run("EncoderError", func(t *testing.T) {
		encoder := codec.FromEncoder(
			func(input string) (chatter.Message, error) {
				return nil, errors.New("encoder error")
			},
		)

		llm := &Mock{}
		registry := &MockRegistry{}
		manifold := agent.NewManifold(
			llm,
			encoder,
			codec.DecoderID,
			registry,
		)

		_, err := manifold.Prompt(context.Background(), "test")

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.String(err.Error()).Contain("encoder error"),
		)
	})

	t.Run("MultiplePrompts", func(t *testing.T) {
		llm := &Mock{}
		registry := &MockRegistry{}
		manifold := agent.NewManifold(
			llm,
			codec.String,
			codec.String,
			registry,
		)

		// Test multiple prompts with the same manifold
		inputs := []string{"First", "Second", "Third"}
		for _, input := range inputs {
			result, err := manifold.Prompt(context.Background(), input)

			it.Then(t).Should(
				it.Nil(err),
			)
			it.Then(t).Should(
				it.String(result).Contain(input),
			)
		}
	})

	t.Run("ComplexWorkflow", func(t *testing.T) {
		encoder := codec.FromEncoder(
			func(input string) (chatter.Message, error) {
				var prompt chatter.Prompt
				prompt.WithTask("Process: %s", input)
				prompt.WithRules("Be precise", "Follow guidelines")
				return &prompt, nil
			},
		)

		decoder := codec.FromDecoder(
			func(reply *chatter.Reply) (float64, map[string]string, error) {
				return 1.0, map[string]string{
					"input":  "test",
					"output": reply.String(),
				}, nil
			},
		)

		llm := &Mock{}
		registry := &MockRegistry{}
		manifold := agent.NewManifold(
			llm,
			encoder,
			decoder,
			registry,
		)

		result, err := manifold.Prompt(context.Background(), "data")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).ShouldNot(
			it.Nil(result),
		)
		it.Then(t).Should(
			it.Equal(result["input"], "test"),
		)
		it.Then(t).Should(
			it.String(result["output"]).Contain("Process: data"),
		)
	})
}
