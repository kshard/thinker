//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package nanobot_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/agent/nanobot"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
)

// =============================================================================
// Mocks
// =============================================================================

// MockBot is a configurable mock for the Bot[S, A] interface.
type MockBot[S, A any] struct {
	fn func(context.Context, S, ...chatter.Opt) (A, error)
}

func (m *MockBot[S, A]) Prompt(ctx context.Context, input S, opt ...chatter.Opt) (A, error) {
	return m.fn(ctx, input, opt...)
}

// MockChatter is a configurable mock for the chatter.Chatter interface used by Jsonify.
type MockChatter struct {
	response string
	err      error
}

func (m *MockChatter) Usage() chatter.Usage { return chatter.Usage{} }

func (m *MockChatter) Prompt(_ context.Context, _ []chatter.Message, _ ...chatter.Opt) (*chatter.Reply, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &chatter.Reply{
		Stage:   chatter.LLM_RETURN,
		Content: []chatter.Content{chatter.Text(m.response)},
	}, nil
}

// MockLLMs is a mock for the LLMs registry.
type MockLLMs struct {
	models map[string]chatter.Chatter
}

func (m *MockLLMs) Model(name string) (chatter.Chatter, bool) {
	c, ok := m.models[name]
	return c, ok
}

// MockChalk is a no-op mock for the Chalk interface that records calls.
type MockChalk struct {
	tasks  []string
	dones  int
	failed []error
}

func (c *MockChalk) Sub(ctx context.Context) context.Context { return ctx }
func (c *MockChalk) Task(_ context.Context, format string, _ ...any) {
	c.tasks = append(c.tasks, format)
}
func (c *MockChalk) Done(...string)        { c.dones++ }
func (c *MockChalk) Fail(err error)        { c.failed = append(c.failed, err) }
func (c *MockChalk) Printf(string, ...any) {}

// =============================================================================
// Test state type
//
// Work has a single string field so that optics.ForProduct1[Work, string]
// can resolve the lens automatically (used by default apply in Seq/Reflect/ThinkReAct).
// =============================================================================

type Work struct{ Result string }

// =============================================================================
// TestRuntime
// =============================================================================

func TestRuntime(t *testing.T) {
	t.Run("NewRuntime", func(t *testing.T) {
		rt := nanobot.NewRuntime(nil, nil)
		it.Then(t).ShouldNot(it.Nil(rt))
	})

	t.Run("WithRegistry", func(t *testing.T) {
		rt := nanobot.NewRuntime(nil, nil)
		reg := command.NewRegistry()
		rt2 := rt.WithRegistry(reg)

		it.Then(t).ShouldNot(it.Nil(rt2))
		it.Then(t).ShouldNot(it.Equal(rt, rt2))
	})

	t.Run("WithStdout", func(t *testing.T) {
		rt := nanobot.NewRuntime(nil, nil)
		chalk := &MockChalk{}
		rt2 := rt.WithStdout(chalk)

		it.Then(t).ShouldNot(it.Nil(rt2))
		it.Then(t).ShouldNot(it.Equal(rt, rt2))
	})
}

// =============================================================================
// TestSeq
// =============================================================================

func TestSeq(t *testing.T) {
	rt := nanobot.NewRuntime(nil, nil)

	t.Run("Success", func(t *testing.T) {
		botA := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return "from-a", nil
			},
		}
		botB := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result + "-from-b", nil
			},
		}

		seq, err := nanobot.NewSeq(rt, botA, botB)
		it.Then(t).Should(it.Nil(err))

		result, err := seq.Prompt(context.Background(), Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result, "from-a-from-b"),
		)
	})

	t.Run("BotAFails", func(t *testing.T) {
		errA := errors.New("bot-a error")
		botA := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "", errA
			},
		}
		botB := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "from-b", nil
			},
		}

		seq, err := nanobot.NewSeq(rt, botA, botB)
		it.Then(t).Should(it.Nil(err))

		_, err = seq.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errA)))
	})

	t.Run("BotBFails", func(t *testing.T) {
		errB := errors.New("bot-b error")
		botA := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "from-a", nil
			},
		}
		botB := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "", errB
			},
		}

		seq, err := nanobot.NewSeq(rt, botA, botB)
		it.Then(t).Should(it.Nil(err))

		_, err = seq.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errB)))
	})

	t.Run("WithApply", func(t *testing.T) {
		type StateAB struct {
			A string
			B string
		}

		applied := ""
		botA := &MockBot[StateAB, string]{
			fn: func(_ context.Context, _ StateAB, _ ...chatter.Opt) (string, error) {
				return "output-a", nil
			},
		}
		botB := &MockBot[StateAB, string]{
			fn: func(_ context.Context, s StateAB, _ ...chatter.Opt) (string, error) {
				return s.A, nil
			},
		}

		seq, err := nanobot.NewSeq(rt, botA, botB)
		it.Then(t).Should(it.Nil(err))

		seq = seq.WithApply(func(s StateAB, a string) StateAB {
			applied = a
			return StateAB{A: a, B: s.B}
		})

		result, err := seq.Prompt(context.Background(), StateAB{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result, "output-a"),
			it.Equal(applied, "output-a"),
		)
	})

	t.Run("WithEffect", func(t *testing.T) {
		effectCalled := false
		botA := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "a", nil
			},
		}
		botB := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}

		seq, err := nanobot.NewSeq(rt, botA, botB)
		it.Then(t).Should(it.Nil(err))

		seq = seq.WithEffect(func(ctx context.Context, w Work) (Work, error) {
			effectCalled = true
			return w, nil
		})

		_, err = seq.Prompt(context.Background(), Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.True(effectCalled),
		)
	})

	t.Run("EffectFails", func(t *testing.T) {
		errEffect := errors.New("effect error")
		botA := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "a", nil
			},
		}
		botB := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "b", nil
			},
		}

		seq, err := nanobot.NewSeq(rt, botA, botB)
		it.Then(t).Should(it.Nil(err))

		seq = seq.WithEffect(func(_ context.Context, w Work) (Work, error) {
			return w, errEffect
		})

		_, err = seq.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errEffect)))
	})
}

// =============================================================================
// TestReflect
// =============================================================================

func TestReflect(t *testing.T) {
	rt := nanobot.NewRuntime(nil, nil)

	t.Run("AcceptOnFirstAttempt", func(t *testing.T) {
		judge := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return "verdict", nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "corrected", nil
			},
		}

		bot, err := nanobot.NewReflect(rt, judge, react)
		it.Then(t).Should(it.Nil(err))

		// default accept always accepts (returns +1)
		result, err := bot.Prompt(context.Background(), Work{Result: "initial"})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "initial"),
		)
	})

	t.Run("ImmediateReject", func(t *testing.T) {
		judge := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "bad", nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "fix", nil
			},
		}

		bot, err := nanobot.NewReflect(rt, judge, react)
		it.Then(t).Should(it.Nil(err))

		// accept function returns -1 to immediately reject
		bot = bot.WithAccept(func(s Work, verdict string) (Work, int) {
			return s, -1
		})

		_, err = bot.Prompt(context.Background(), Work{Result: "initial"})
		it.Then(t).ShouldNot(it.Nil(err))
	})

	t.Run("RetryThenAccept", func(t *testing.T) {
		call := 0
		judge := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "corrected", nil
			},
		}

		bot, err := nanobot.NewReflect(rt, judge, react)
		it.Then(t).Should(it.Nil(err))

		bot = bot.
			WithAttempts(3).
			WithAccept(func(s Work, verdict string) (Work, int) {
				call++
				if call < 2 {
					// neutral → continue retrying
					return s, 0
				}
				// accept on 2nd call
				return Work{Result: "accepted"}, 1
			})

		result, err := bot.Prompt(context.Background(), Work{Result: "initial"})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "accepted"),
		)
	})

	t.Run("ExhaustsAttempts", func(t *testing.T) {
		judge := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "fix", nil
			},
		}

		bot, err := nanobot.NewReflect(rt, judge, react)
		it.Then(t).Should(it.Nil(err))

		// always neutral → exhaust all 2 attempts
		bot = bot.
			WithAttempts(2).
			WithAccept(func(s Work, verdict string) (Work, int) {
				return s, 0
			})

		_, err = bot.Prompt(context.Background(), Work{Result: "initial"})
		it.Then(t).ShouldNot(it.Nil(err))
	})

	t.Run("JudgeError", func(t *testing.T) {
		errJudge := errors.New("judge error")
		judge := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "", errJudge
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "fix", nil
			},
		}

		bot, err := nanobot.NewReflect(rt, judge, react)
		it.Then(t).Should(it.Nil(err))
		bot = bot.WithAccept(func(s Work, verdict string) (Work, int) {
			return s, 0
		})

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errJudge)))
	})

	t.Run("ReactError", func(t *testing.T) {
		errReact := errors.New("react error")
		judge := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "", errReact
			},
		}

		bot, err := nanobot.NewReflect(rt, judge, react)
		it.Then(t).Should(it.Nil(err))

		// neutral → triggers react bot
		bot = bot.WithAttempts(2).WithAccept(func(s Work, verdict string) (Work, int) {
			return s, 0
		})

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errReact)))
	})

	t.Run("EffectError", func(t *testing.T) {
		errEffect := errors.New("effect error")
		judge := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "fix", nil
			},
		}

		bot, err := nanobot.NewReflect(rt, judge, react)
		it.Then(t).Should(it.Nil(err))

		bot = bot.
			WithAttempts(2).
			WithAccept(func(s Work, _ string) (Work, int) { return s, 0 }).
			WithEffect(func(_ context.Context, w Work) (Work, error) {
				return w, errEffect
			})

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errEffect)))
	})

	t.Run("WithChalk", func(t *testing.T) {
		chalk := &MockChalk{}
		rt2 := nanobot.NewRuntime(nil, nil).WithStdout(chalk)

		judge := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "fix", nil
			},
		}

		bot, err := nanobot.NewReflect(rt2, judge, react)
		it.Then(t).Should(it.Nil(err))
		// default accept → accept on first attempt
		_, err = bot.Prompt(context.Background(), Work{Result: "state"})
		it.Then(t).Should(it.Nil(err))
		it.Then(t).Should(it.True(chalk.dones >= 2))
	})
}

// =============================================================================
// TestThinkReAct
// =============================================================================

func TestThinkReAct(t *testing.T) {
	rt := nanobot.NewRuntime(nil, nil)

	t.Run("Success", func(t *testing.T) {
		think := &MockBot[Work, []string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]string, error) {
				return []string{"task-1", "task-2"}, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return "done:" + w.Result, nil
			},
		}

		bot, err := nanobot.NewThinkReAct(rt, think, react)
		it.Then(t).Should(it.Nil(err))

		results, err := bot.Prompt(context.Background(), Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(len(results), 2),
		)
	})

	t.Run("ThinkFails", func(t *testing.T) {
		errThink := errors.New("think error")
		think := &MockBot[Work, []string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]string, error) {
				return nil, errThink
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "done", nil
			},
		}

		bot, err := nanobot.NewThinkReAct(rt, think, react)
		it.Then(t).Should(it.Nil(err))

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errThink)))
	})

	t.Run("ReactFails", func(t *testing.T) {
		errReact := errors.New("react error")
		think := &MockBot[Work, []string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]string, error) {
				return []string{"task-1", "task-2"}, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "", errReact
			},
		}

		bot, err := nanobot.NewThinkReAct(rt, think, react)
		it.Then(t).Should(it.Nil(err))

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errReact)))
	})

	t.Run("WithApply", func(t *testing.T) {
		applied := []string{}
		think := &MockBot[Work, []string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]string, error) {
				return []string{"t1", "t2"}, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}

		bot, err := nanobot.NewThinkReAct(rt, think, react)
		it.Then(t).Should(it.Nil(err))

		bot = bot.WithApply(func(s Work, task string) Work {
			applied = append(applied, task)
			return Work{Result: task}
		})

		results, err := bot.Prompt(context.Background(), Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(len(results), 2),
			it.Equal(results[0], "t1"),
			it.Equal(results[1], "t2"),
			it.Equal(len(applied), 2),
		)
	})

	t.Run("WithEffect", func(t *testing.T) {
		effectCount := 0
		think := &MockBot[Work, []string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]string, error) {
				return []string{"t1", "t2", "t3"}, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}

		bot, err := nanobot.NewThinkReAct(rt, think, react)
		it.Then(t).Should(it.Nil(err))

		bot = bot.WithEffect(func(_ context.Context, w Work) (Work, error) {
			effectCount++
			return w, nil
		})

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(effectCount, 3),
		)
	})

	t.Run("EffectFails", func(t *testing.T) {
		errEffect := errors.New("effect error")
		think := &MockBot[Work, []string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]string, error) {
				return []string{"t1"}, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "done", nil
			},
		}

		bot, err := nanobot.NewThinkReAct(rt, think, react)
		it.Then(t).Should(it.Nil(err))

		bot = bot.WithEffect(func(_ context.Context, w Work) (Work, error) {
			return w, errEffect
		})

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errEffect)))
	})
}

// =============================================================================
// TestJsonify
// =============================================================================

func TestJsonify(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		llm := &MockChatter{response: `["apple", "banana", "cherry"]`}

		bot := nanobot.NewJsonify[string](
			llm,
			3,
			codec.FromEncoder(func(in string) (chatter.Message, error) {
				var p chatter.Prompt
				p.WithTask(in)
				return &p, nil
			}),
			func(seq []string) error { return nil },
		)

		result, err := bot.Prompt(context.Background(), "list fruits")
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(len(result), 3),
			it.Equal(result[0], "apple"),
			it.Equal(result[1], "banana"),
			it.Equal(result[2], "cherry"),
		)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		llm := &MockChatter{response: "this is not json at all"}

		bot := nanobot.NewJsonify[string](
			llm,
			2,
			codec.FromEncoder(func(in string) (chatter.Message, error) {
				var p chatter.Prompt
				p.WithTask(in)
				return &p, nil
			}),
			func(seq []string) error { return nil },
		)

		_, err := bot.Prompt(context.Background(), "list fruits")
		it.Then(t).ShouldNot(it.Nil(err))
	})

	t.Run("ValidatorRejects", func(t *testing.T) {
		llm := &MockChatter{response: `["only-one"]`}

		bot := nanobot.NewJsonify[string](
			llm,
			2,
			codec.FromEncoder(func(in string) (chatter.Message, error) {
				var p chatter.Prompt
				p.WithTask(in)
				return &p, nil
			}),
			func(seq []string) error {
				if len(seq) < 2 {
					return errors.New("need at least 2 items")
				}
				return nil
			},
		)

		_, err := bot.Prompt(context.Background(), "list fruits")
		it.Then(t).ShouldNot(it.Nil(err))
	})

	t.Run("LLMError", func(t *testing.T) {
		errLLM := errors.New("llm failure")
		llm := &MockChatter{err: errLLM}

		bot := nanobot.NewJsonify[string](
			llm,
			3,
			codec.FromEncoder(func(in string) (chatter.Message, error) {
				var p chatter.Prompt
				p.WithTask(in)
				return &p, nil
			}),
			func(seq []string) error { return nil },
		)

		_, err := bot.Prompt(context.Background(), "list fruits")
		it.Then(t).ShouldNot(it.Nil(err))
	})
}
