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
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/memory"
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
		memory := memory.NewStream(-1, "")
		manifold := agent.NewManifold(
			llm,
			codec.EncoderID,
			codec.DecoderID,
			registry,
		).WithMemory(memory)

		reply, err := manifold.Prompt(context.Background(), "Hello, World!")
		it.Then(t).Must(it.Nil(err))

		it.Then(t).ShouldNot(
			it.Nil(reply),
		)

		// The reply should contain the echoed prompt
		replyText := reply.String()
		it.Then(t).Should(
			it.Equal(len(memory.Context(nil)), 2),
			it.String(replyText).Contain("Hello, World!"),
		)
	})

	t.Run("StringCodec", func(t *testing.T) {
		llm := &Mock{}
		registry := &MockRegistry{}
		memory := memory.NewStream(-1, "")
		manifold := agent.NewManifold(
			llm,
			codec.String,
			codec.String,
			registry,
		).WithMemory(memory)

		result, err := manifold.Prompt(context.Background(), "Test input")
		it.Then(t).Must(it.Nil(err))

		it.Then(t).Should(
			it.Equal(len(memory.Context(nil)), 2),
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

//------------------------------------------------------------------------------
// Memory mocks
//------------------------------------------------------------------------------

// feedbackErr is an error that also implements chatter.Content so the manifold
// recognises it as decoder feedback and starts a retry loop.
type feedbackErr string

func (f feedbackErr) Error() string  { return string(f) }
func (f feedbackErr) String() string { return string(f) }

// InvokeThenReturnMock returns LLM_INVOKE on the first call, then echoes on subsequent calls.
type InvokeThenReturnMock struct {
	calls int
}

func (m *InvokeThenReturnMock) Usage() chatter.Usage { return chatter.Usage{} }

func (m *InvokeThenReturnMock) Prompt(_ context.Context, prompt []chatter.Message, _ ...chatter.Opt) (*chatter.Reply, error) {
	m.calls++
	if m.calls == 1 {
		return &chatter.Reply{
			Stage:   chatter.LLM_INVOKE,
			Content: []chatter.Content{chatter.Text("invoke request")},
		}, nil
	}
	seq := make([]string, len(prompt))
	for i, msg := range prompt {
		seq[i] = msg.String()
	}
	return &chatter.Reply{
		Stage:   chatter.LLM_RETURN,
		Content: []chatter.Content{chatter.Text(strings.Join(seq, " "))},
	}, nil
}

// IncompleteMock echoes its input but with Stage LLM_INCOMPLETE.
type IncompleteMock struct{}

func (m *IncompleteMock) Usage() chatter.Usage { return chatter.Usage{} }

func (m *IncompleteMock) Prompt(_ context.Context, prompt []chatter.Message, _ ...chatter.Opt) (*chatter.Reply, error) {
	seq := make([]string, len(prompt))
	for i, msg := range prompt {
		seq[i] = msg.String()
	}
	return &chatter.Reply{
		Stage:   chatter.LLM_INCOMPLETE,
		Content: []chatter.Content{chatter.Text(strings.Join(seq, " "))},
	}, nil
}

// ErrorMock always returns an error from Prompt.
type ErrorMock struct{}

func (m *ErrorMock) Usage() chatter.Usage { return chatter.Usage{} }

func (m *ErrorMock) Prompt(_ context.Context, _ []chatter.Message, _ ...chatter.Opt) (*chatter.Reply, error) {
	return nil, errors.New("llm error")
}

// LoopRegistry returns AGENT_ASK so the manifold loops back with the tool answer.
type LoopRegistry struct{}

func (r *LoopRegistry) Context() chatter.Registry { return chatter.Registry{} }

func (r *LoopRegistry) Invoke(_ *chatter.Reply) (thinker.Phase, chatter.Message, error) {
	return thinker.AGENT_ASK, chatter.Text("tool result"), nil
}

// AbortRegistry returns AGENT_ABORT so the manifold stops immediately.
type AbortRegistry struct{}

func (r *AbortRegistry) Context() chatter.Registry { return chatter.Registry{} }

func (r *AbortRegistry) Invoke(_ *chatter.Reply) (thinker.Phase, chatter.Message, error) {
	return thinker.AGENT_ABORT, nil, nil
}

//------------------------------------------------------------------------------
// Test Manifold Memory
//------------------------------------------------------------------------------

func TestManifoldMemory(t *testing.T) {
	// CommitOncePerPrompt verifies that a single Prompt call commits exactly one
	// observation (query + reply), producing two entries in Context.
	t.Run("CommitOncePerPrompt", func(t *testing.T) {
		mem := memory.NewStream(-1, "")
		manifold := agent.NewManifold(&Mock{}, codec.String, codec.String, &MockRegistry{}).WithMemory(mem)

		_, err := manifold.Prompt(context.Background(), "input")
		it.Then(t).Must(it.Nil(err))

		it.Then(t).Should(
			it.Equal(len(mem.Context(nil)), 2),
		)
	})

	// AccumulatesAcrossCalls verifies that each Prompt call appends a new observation
	// so Context grows by two entries per call.
	t.Run("AccumulatesAcrossCalls", func(t *testing.T) {
		mem := memory.NewStream(-1, "")
		manifold := agent.NewManifold(&Mock{}, codec.String, codec.String, &MockRegistry{}).WithMemory(mem)

		for i, input := range []string{"First", "Second", "Third"} {
			_, err := manifold.Prompt(context.Background(), input)
			it.Then(t).Must(it.Nil(err))

			it.Then(t).Should(
				it.Equal(len(mem.Context(nil)), (i+1)*2),
			)
		}
	})

	// CapacityEviction verifies that when the stream capacity is 1, old observations
	// are evicted so Context always contains exactly two entries regardless of how
	// many Prompt calls have been made.
	t.Run("CapacityEviction", func(t *testing.T) {
		mem := memory.NewStream(1, "")
		manifold := agent.NewManifold(&Mock{}, codec.String, codec.String, &MockRegistry{}).WithMemory(mem)

		_, err := manifold.Prompt(context.Background(), "First")
		it.Then(t).Must(it.Nil(err))
		it.Then(t).Should(it.Equal(len(mem.Context(nil)), 2))

		_, err = manifold.Prompt(context.Background(), "Second")
		it.Then(t).Must(it.Nil(err))
		// cap=1: oldest observation evicted, only most-recent retained
		it.Then(t).Should(it.Equal(len(mem.Context(nil)), 2))

		_, err = manifold.Prompt(context.Background(), "Third")
		it.Then(t).Must(it.Nil(err))
		it.Then(t).Should(it.Equal(len(mem.Context(nil)), 2))
	})

	// StratumInContext verifies that a non-empty stratum is prepended to the context
	// window sent to the LLM and counted in Context(nil).
	t.Run("StratumInContext", func(t *testing.T) {
		mem := memory.NewStream(-1, "You are a helpful assistant.")
		manifold := agent.NewManifold(&Mock{}, codec.String, codec.String, &MockRegistry{}).WithMemory(mem)

		// The echo mock includes the stratum in its reply.
		result, err := manifold.Prompt(context.Background(), "Hello")
		it.Then(t).Must(it.Nil(err))
		it.Then(t).Should(
			it.String(result).Contain("You are a helpful assistant"),
			it.String(result).Contain("Hello"),
		)

		// context(nil) = [stratum] + [obs.query, obs.reply] = 3 entries
		it.Then(t).Should(it.Equal(len(mem.Context(nil)), 3))
	})

	// PriorContextPassedToLLM verifies that the context window on the second Prompt
	// call includes the first observation, so the LLM echo carries prior conversation.
	t.Run("PriorContextPassedToLLM", func(t *testing.T) {
		mem := memory.NewStream(-1, "")
		manifold := agent.NewManifold(&Mock{}, codec.String, codec.String, &MockRegistry{}).WithMemory(mem)

		_, err := manifold.Prompt(context.Background(), "First")
		it.Then(t).Must(it.Nil(err))

		// Second call: LLM receives [prior_query, prior_reply, new_prompt] and echoes all.
		result, err := manifold.Prompt(context.Background(), "Second")
		it.Then(t).Must(it.Nil(err))
		it.Then(t).Should(
			it.String(result).Contain("First"),
			it.String(result).Contain("Second"),
		)
	})

	// LLMInvokeLoopCommitsTwice verifies that when LLM_INVOKE causes a second
	// loop iteration each iteration's observation is committed independently.
	t.Run("LLMInvokeLoopCommitsTwice", func(t *testing.T) {
		mem := memory.NewStream(-1, "")
		manifold := agent.NewManifold(
			&InvokeThenReturnMock{},
			codec.String,
			codec.String,
			&LoopRegistry{},
		).WithMemory(mem)

		_, err := manifold.Prompt(context.Background(), "input")
		it.Then(t).Must(it.Nil(err))

		// 2 LLM calls → 2 observations → 4 context entries
		it.Then(t).Should(it.Equal(len(mem.Context(nil)), 4))
	})

	// DecoderFeedbackLoopCommitsTwice verifies that when a decoder returns a
	// chatter.Content feedback error the failed LLM reply is already committed
	// before the loop retries, resulting in two committed observations.
	t.Run("DecoderFeedbackLoopCommitsTwice", func(t *testing.T) {
		callCount := 0
		decoder := codec.FromDecoder(
			func(reply *chatter.Reply) (float64, string, error) {
				callCount++
				if callCount == 1 {
					return 0, "", feedbackErr("Please retry.")
				}
				return 1.0, reply.String(), nil
			},
		)

		mem := memory.NewStream(-1, "")
		manifold := agent.NewManifold(&Mock{}, codec.String, decoder, &MockRegistry{}).WithMemory(mem)

		_, err := manifold.Prompt(context.Background(), "input")
		it.Then(t).Must(it.Nil(err))

		// 2 LLM calls → 2 observations → 4 context entries
		it.Then(t).Should(it.Equal(len(mem.Context(nil)), 4))
	})

	// LLMErrorDoesNotCommit verifies that when the LLM returns an error the failed
	// call is not committed to memory because Commit follows a successful LLM call.
	t.Run("LLMErrorDoesNotCommit", func(t *testing.T) {
		mem := memory.NewStream(-1, "")
		manifold := agent.NewManifold(&ErrorMock{}, codec.String, codec.String, &MockRegistry{}).WithMemory(mem)

		_, err := manifold.Prompt(context.Background(), "input")
		it.Then(t).ShouldNot(it.Nil(err))

		// LLM failed before Commit → memory stays empty
		it.Then(t).Should(it.Equal(len(mem.Context(nil)), 0))
	})

	// LLMIncompleteCommitsOnce verifies that LLM_INCOMPLETE follows the same commit
	// path as LLM_RETURN and produces a single observation.
	t.Run("LLMIncompleteCommitsOnce", func(t *testing.T) {
		mem := memory.NewStream(-1, "")
		manifold := agent.NewManifold(&IncompleteMock{}, codec.String, codec.String, &MockRegistry{}).WithMemory(mem)

		_, err := manifold.Prompt(context.Background(), "input")
		it.Then(t).Must(it.Nil(err))

		it.Then(t).Should(it.Equal(len(mem.Context(nil)), 2))
	})

	// InvokeAbortCommitsBeforeAbort verifies that the LLM_INVOKE observation is
	// committed before the registry's AGENT_ABORT terminates execution.
	t.Run("InvokeAbortCommitsBeforeAbort", func(t *testing.T) {
		mem := memory.NewStream(-1, "")
		manifold := agent.NewManifold(
			&InvokeThenReturnMock{},
			codec.String,
			codec.String,
			&AbortRegistry{},
		).WithMemory(mem)

		_, err := manifold.Prompt(context.Background(), "input")
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.String(err.Error()).Contain("aborted"))

		// The LLM_INVOKE reply was committed before the abort
		it.Then(t).Should(it.Equal(len(mem.Context(nil)), 2))
	})
}
