//
// Copyright (C) 2026 Dmitry Kolesnikov
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
	"testing/fstest"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
)

//------------------------------------------------------------------------------
// Test fixtures (in-memory FS)
//------------------------------------------------------------------------------

var nanobotFS = fstest.MapFS{
	"text_prompt.md": {
		Data: []byte("---\nformat: text\n---\nHello {{.}}!\n"),
	},
	"json_prompt.md": {
		Data: []byte("---\nformat: json\n---\nAnalyze: {{.}}\n"),
	},
	"invalid_template.md": {
		Data: []byte("---\nformat: text\n---\nHello {{.Invalid\n"),
	},
	"runs_on_micro.md": {
		Data: []byte("---\nformat: text\nruns-on: micro\n---\nHello {{.}}!\n"),
	},
	"input_schema.md": {
		Data: []byte("---\nformat: json\nschema:\n  input:\n    type: object\n    properties:\n      name:\n        type: string\n    required:\n      - name\n---\nProcess {{.}}\n"),
	},
}

//------------------------------------------------------------------------------
// Helpers: TextUnmarshaler output type
//------------------------------------------------------------------------------

type TextOut struct {
	Value string
}

func (t *TextOut) UnmarshalText(data []byte) error {
	t.Value = string(data)
	return nil
}

//------------------------------------------------------------------------------
// Helpers: unsupported output type for text format
//------------------------------------------------------------------------------

type unsupportedOut int

//------------------------------------------------------------------------------
// Helpers: JSON output type
//------------------------------------------------------------------------------

type JSONOut struct {
	Items []string `json:"items"`
}

//------------------------------------------------------------------------------
// Additional mocks used only in nanobot tests
//------------------------------------------------------------------------------

// MockFixed always returns a fixed response string.
type MockFixed struct {
	response string
}

func (m *MockFixed) Usage() chatter.Usage { return chatter.Usage{} }

func (m *MockFixed) Prompt(_ context.Context, _ []chatter.Message, _ ...chatter.Opt) (*chatter.Reply, error) {
	return &chatter.Reply{
		Stage:   chatter.LLM_RETURN,
		Content: []chatter.Content{chatter.Text(m.response)},
	}, nil
}

// MockSequence returns responses in order; after the list is exhausted it
// returns an error so tests don't block.
type MockSequence struct {
	responses []string
	index     int
}

func (m *MockSequence) Usage() chatter.Usage { return chatter.Usage{} }

func (m *MockSequence) Prompt(_ context.Context, _ []chatter.Message, _ ...chatter.Opt) (*chatter.Reply, error) {
	if m.index >= len(m.responses) {
		return nil, errors.New("MockSequence: no more responses")
	}
	resp := m.responses[m.index]
	m.index++
	return &chatter.Reply{
		Stage:   chatter.LLM_RETURN,
		Content: []chatter.Content{chatter.Text(resp)},
	}, nil
}

// MockError always returns an LLM-level error.
type MockError struct{}

func (m *MockError) Usage() chatter.Usage { return chatter.Usage{} }

func (m *MockError) Prompt(_ context.Context, _ []chatter.Message, _ ...chatter.Opt) (*chatter.Reply, error) {
	return nil, errors.New("llm unavailable")
}

// MockTagged records whether it was called and echoes the prompt content.
type MockTagged struct {
	tag    string
	called bool
}

func (m *MockTagged) Usage() chatter.Usage { return chatter.Usage{} }

func (m *MockTagged) Prompt(_ context.Context, msgs []chatter.Message, _ ...chatter.Opt) (*chatter.Reply, error) {
	m.called = true
	var sb strings.Builder
	for _, msg := range msgs {
		sb.WriteString(msg.String())
	}
	return &chatter.Reply{
		Stage:   chatter.LLM_RETURN,
		Content: []chatter.Content{chatter.Text(sb.String())},
	}, nil
}

//------------------------------------------------------------------------------
// TestNewNanoBot
//------------------------------------------------------------------------------

func TestNewNanoBot(t *testing.T) {
	t.Run("FileNotFound", func(t *testing.T) {
		llm := thinker.LLM{Base: &Mock{}}
		_, err := agent.NewNanoBot[string, TextOut](llm, nanobotFS, "missing.md")

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})

	t.Run("InvalidTemplate", func(t *testing.T) {
		llm := thinker.LLM{Base: &Mock{}}
		_, err := agent.NewNanoBot[string, TextOut](llm, nanobotFS, "invalid_template.md")

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})

	t.Run("ValidTextPrompt", func(t *testing.T) {
		llm := thinker.LLM{Base: &Mock{}}
		bot, err := agent.NewNanoBot[string, TextOut](llm, nanobotFS, "text_prompt.md")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).ShouldNot(
			it.Nil(bot),
		)
	})

	t.Run("ValidJsonPrompt", func(t *testing.T) {
		llm := thinker.LLM{Base: &Mock{}}
		bot, err := agent.NewNanoBot[string, JSONOut](llm, nanobotFS, "json_prompt.md")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).ShouldNot(
			it.Nil(bot),
		)
	})
}

//------------------------------------------------------------------------------
// TestMakeNanoBot
//------------------------------------------------------------------------------

func TestMakeNanoBot(t *testing.T) {
	t.Run("PanicsOnMissingFile", func(t *testing.T) {
		defer func() {
			r := recover()
			it.Then(t).ShouldNot(
				it.Nil(r),
			)
		}()

		llm := thinker.LLM{Base: &Mock{}}
		agent.MakeNanoBot[string, TextOut](llm, nanobotFS, "missing.md")
	})

	t.Run("SucceedsOnValidFile", func(t *testing.T) {
		llm := thinker.LLM{Base: &Mock{}}
		bot := agent.MakeNanoBot[string, TextOut](llm, nanobotFS, "text_prompt.md")

		it.Then(t).ShouldNot(
			it.Nil(bot),
		)
	})
}

//------------------------------------------------------------------------------
// TestNanoBot_TextFormat
//------------------------------------------------------------------------------

func TestNanoBot_TextFormat(t *testing.T) {
	t.Run("TextUnmarshaler", func(t *testing.T) {
		llm := thinker.LLM{Base: &Mock{}}
		bot, err := agent.NewNanoBot[string, TextOut](llm, nanobotFS, "text_prompt.md")
		it.Then(t).Should(it.Nil(err))

		result, err := bot.Prompt(context.Background(), "World")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.String(result.Value).Contain("World"),
		)
	})

	t.Run("UnsupportedOutputType", func(t *testing.T) {
		llm := thinker.LLM{Base: &Mock{}}
		bot, err := agent.NewNanoBot[string, unsupportedOut](llm, nanobotFS, "text_prompt.md")
		it.Then(t).Should(it.Nil(err))

		_, err = bot.Prompt(context.Background(), "World")

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.String(err.Error()).Contain("nanobot unable to handle type"),
		)
	})
}

//------------------------------------------------------------------------------
// TestNanoBot_JsonFormat
//------------------------------------------------------------------------------

func TestNanoBot_JsonFormat(t *testing.T) {
	t.Run("DecodesValidJSON", func(t *testing.T) {
		mock := &MockFixed{response: `{"items":["a","b","c"]}`}
		llm := thinker.LLM{Base: mock}
		bot, err := agent.NewNanoBot[string, JSONOut](llm, nanobotFS, "json_prompt.md")
		it.Then(t).Should(it.Nil(err))

		result, err := bot.Prompt(context.Background(), "test")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.Seq(result.Items).Contain("a", "b", "c"),
		)
	})

	t.Run("RetriesOnInvalidJSONThenSucceeds", func(t *testing.T) {
		// First call returns no JSON (triggers feedback/retry).
		// Second call returns valid JSON.
		mock := &MockSequence{responses: []string{
			"not json yet",
			`{"items":["x"]}`,
		}}
		llm := thinker.LLM{Base: mock}
		bot, err := agent.NewNanoBot[string, JSONOut](llm, nanobotFS, "json_prompt.md")
		it.Then(t).Should(it.Nil(err))

		result, err := bot.Prompt(context.Background(), "test")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.Seq(result.Items).Contain("x"),
		)
		// LLM was called twice: initial request + one retry after feedback.
		it.Then(t).Should(
			it.Equal(mock.index, 2),
		)
	})

	t.Run("LLMError", func(t *testing.T) {
		mock := &MockError{}
		llm := thinker.LLM{Base: mock}
		bot, err := agent.NewNanoBot[string, JSONOut](llm, nanobotFS, "json_prompt.md")
		it.Then(t).Should(it.Nil(err))

		_, err = bot.Prompt(context.Background(), "test")

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})
}

//------------------------------------------------------------------------------
// TestNanoBot_RunsOn
//------------------------------------------------------------------------------

func TestNanoBot_RunsOn(t *testing.T) {
	t.Run("DefaultUsesBase", func(t *testing.T) {
		base := &MockTagged{tag: "base"}
		llm := thinker.LLM{Base: base}
		bot, err := agent.NewNanoBot[string, TextOut](llm, nanobotFS, "text_prompt.md")
		it.Then(t).Should(it.Nil(err))

		_, err = bot.Prompt(context.Background(), "test")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.True(base.called),
		)
	})

	t.Run("MicroTier", func(t *testing.T) {
		micro := &MockTagged{tag: "micro"}
		llm := thinker.LLM{Base: &Mock{}, Micro: micro}
		bot, err := agent.NewNanoBot[string, TextOut](llm, nanobotFS, "runs_on_micro.md")
		it.Then(t).Should(it.Nil(err))

		_, err = bot.Prompt(context.Background(), "test")

		it.Then(t).Should(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.True(micro.called),
		)
	})
}

//------------------------------------------------------------------------------
// TestNanoBot_InputValidation
//------------------------------------------------------------------------------

func TestNanoBot_InputValidation(t *testing.T) {
	t.Run("ValidInput", func(t *testing.T) {
		mock := &MockFixed{response: `{"items":[]}`}
		llm := thinker.LLM{Base: mock}
		bot, err := agent.NewNanoBot[map[string]any, JSONOut](llm, nanobotFS, "input_schema.md")
		it.Then(t).Should(it.Nil(err))

		_, err = bot.Prompt(context.Background(), map[string]any{"name": "Alice"})

		// Valid input should not produce an input-validation error.
		if err != nil {
			it.Then(t).ShouldNot(
				it.String(err.Error()).Contain("input validation failed"),
			)
		}
	})

	// t.Run("InvalidInput", func(t *testing.T) {
	// 	mock := &MockFixed{response: `{"items":[]}`}
	// 	llm := thinker.LLM{Base: mock}
	// 	bot, err := agent.NewNanoBot[map[string]any, JSONOut](llm, nanobotFS, "input_schema.md")
	// 	it.Then(t).Should(it.Nil(err))

	// 	_, err = bot.Prompt(context.Background(), map[string]any{"other": "value"})

	// 	it.Then(t).ShouldNot(
	// 		it.Nil(err),
	// 	)
	// 	it.Then(t).Should(
	// 		it.String(err.Error()).Contain("input validation failed"),
	// 	)
	// })
}
