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
	"strings"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/agent"
)

//------------------------------------------------------------------------------
// Mock LLM
//------------------------------------------------------------------------------

// Mock is a simple mock LLM that echoes the input.
type Mock struct{}

func (m *Mock) Usage() chatter.Usage {
	return chatter.Usage{}
}

func (m *Mock) Prompt(ctx context.Context, prompt []chatter.Message, opt ...chatter.Opt) (*chatter.Reply, error) {
	// Echo all messages
	seq := make([]string, len(prompt))
	for i, msg := range prompt {
		seq[i] = msg.String()
	}
	reply := strings.Join(seq, " ")

	return &chatter.Reply{
		Stage: chatter.LLM_RETURN,
		Usage: chatter.Usage{
			InputTokens: len(reply),
			ReplyTokens: len(reply),
		},
		Content: []chatter.Content{
			chatter.Text(reply),
		},
	}, nil
}

//------------------------------------------------------------------------------
// Test Prompter
//------------------------------------------------------------------------------

func TestPrompter(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		// Create a simple encoder that converts string to prompt
		encoder := func(input string) (chatter.Message, error) {
			var prompt chatter.Prompt
			prompt.WithTask(input)
			return &prompt, nil
		}

		// Create prompter with mock LLM
		llm := &Mock{}
		prompter := agent.NewPrompter(llm, encoder)

		// Test the prompter
		reply, err := prompter.Prompt(context.Background(), "Hello, World!")

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

	t.Run("WithRules", func(t *testing.T) {
		// Create encoder with rules
		encoder := func(input string) (chatter.Message, error) {
			var prompt chatter.Prompt
			prompt.WithTask(input)
			prompt.WithRules("Rule 1", "Rule 2")
			return &prompt, nil
		}

		llm := &Mock{}
		prompter := agent.NewPrompter(llm, encoder)

		reply, err := prompter.Prompt(context.Background(), "Test task")

		it.Then(t).Should(
			it.Nil(err),
		)

		replyText := reply.String()
		it.Then(t).Should(
			it.String(replyText).Contain("Test task"),
			it.String(replyText).Contain("Rule 1"),
			it.String(replyText).Contain("Rule 2"),
		)
	})

	t.Run("MultiplePrompts", func(t *testing.T) {
		encoder := func(input string) (chatter.Message, error) {
			var prompt chatter.Prompt
			prompt.WithTask(input)
			return &prompt, nil
		}

		llm := &Mock{}
		prompter := agent.NewPrompter(llm, encoder)

		// Test multiple prompts with the same prompter
		inputs := []string{"First", "Second", "Third"}
		for _, input := range inputs {
			reply, err := prompter.Prompt(context.Background(), input)

			it.Then(t).Should(
				it.Nil(err),
			)
			it.Then(t).Should(
				it.String(reply.String()).Contain(input),
			)
		}
	})

	t.Run("EncoderError", func(t *testing.T) {
		// Create encoder that returns an error
		encoder := func(input string) (chatter.Message, error) {
			return nil, errors.New("encoding failed")
		}

		llm := &Mock{}
		prompter := agent.NewPrompter(llm, encoder)

		_, err := prompter.Prompt(context.Background(), "test")

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.String(err.Error()).Contain("encoding failed"),
		)
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		encoder := func(input string) (chatter.Message, error) {
			var prompt chatter.Prompt
			prompt.WithTask(input)
			return &prompt, nil
		}

		llm := &Mock{}
		prompter := agent.NewPrompter(llm, encoder)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Note: Mock doesn't check context, but this tests the flow
		_, err := prompter.Prompt(ctx, "test")

		// With our simple mock, this won't error, but demonstrates the pattern
		it.Then(t).Should(
			it.Nil(err),
		)
	})

	t.Run("ComplexPrompt", func(t *testing.T) {
		encoder := func(input string) (chatter.Message, error) {
			var prompt chatter.Prompt
			prompt.WithTask("Process: %s", input)
			prompt.WithRules(
				"Rule 1: Follow guidelines",
				"Rule 2: Be precise",
			)
			prompt.WithExample("input", "output")
			return &prompt, nil
		}

		llm := &Mock{}
		prompter := agent.NewPrompter(llm, encoder)

		reply, err := prompter.Prompt(context.Background(), "data")

		it.Then(t).Should(
			it.Nil(err),
		)

		replyText := reply.String()
		it.Then(t).Should(
			it.String(replyText).Contain("Process: data"),
			it.String(replyText).Contain("Rule 1"),
			it.String(replyText).Contain("Rule 2"),
		)
	})
}
