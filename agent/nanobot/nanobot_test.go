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
	"testing/fstest"

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
	subs   int
}

type chalkCtxKey struct{}

func (c *MockChalk) Sub(ctx context.Context) context.Context {
	c.subs++
	return context.WithValue(ctx, chalkCtxKey{}, true)
}
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
}

// =============================================================================
// TestSeq
// =============================================================================

func TestSeq(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		botA := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "from-a", nil
			},
		}
		botB := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result + "-from-b", nil
			},
		}

		seq := nanobot.Seq(
			nanobot.Arrow(botA),
			nanobot.Arrow(botB),
		)

		result, err := seq.Prompt(context.Background(), Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "from-a-from-b"),
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

		seq := nanobot.Seq(
			nanobot.Arrow[Work, string](botA),
			nanobot.Arrow[Work, string](botB),
		)

		_, err := seq.Prompt(context.Background(), Work{})
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

		seq := nanobot.Seq(
			nanobot.Arrow[Work, string](botA),
			nanobot.Arrow[Work, string](botB),
		)

		_, err := seq.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errB)))
	})

	t.Run("WithApply", func(t *testing.T) {
		type A string
		type B string
		type StateAB struct {
			A A
			B B
		}

		applied := ""
		botA := &MockBot[StateAB, A]{
			fn: func(_ context.Context, _ StateAB, _ ...chatter.Opt) (A, error) {
				return "output-a", nil
			},
		}
		botB := &MockBot[StateAB, B]{
			fn: func(_ context.Context, s StateAB, _ ...chatter.Opt) (B, error) {
				return B(s.A), nil
			},
		}

		arrA := nanobot.Arrow(botA, nanobot.Eff[StateAB, A]{
			Lens: func(s StateAB, a A) StateAB {
				applied = string(a)
				return StateAB{A: a, B: s.B}
			},
		})
		seq := nanobot.Seq(arrA, nanobot.Arrow(botB))

		result, err := seq.Prompt(context.Background(), StateAB{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.A, A("output-a")),
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

		arrA := nanobot.Arrow[Work, string](botA, nanobot.Eff[Work, string]{
			Eval: func(ctx context.Context, w Work) (Work, error) {
				effectCalled = true
				return w, nil
			},
		})
		seq := nanobot.Seq(arrA, nanobot.Arrow[Work, string](botB))

		_, err := seq.Prompt(context.Background(), Work{})
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

		arrA := nanobot.Arrow(botA, nanobot.Eff[Work, string]{
			Eval: func(ctx context.Context, w Work) (Work, error) {
				return w, errEffect
			},
		})
		seq := nanobot.Seq(arrA, nanobot.Arrow(botB))

		_, err := seq.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errEffect)))
	})
}

// =============================================================================
// TestWhen
// =============================================================================

func TestWhen(t *testing.T) {
	t.Run("SkipWhenFalse", func(t *testing.T) {
		called := false
		bot := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				called = true
				return "new", nil
			},
		}
		arr := nanobot.Arrow[Work, string](bot).When(func(w Work) bool { return len(w.Result) == 0 })

		result, err := arr(context.Background(), Work{Result: "already-set"})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "already-set"),
			it.True(!called),
		)
	})

	t.Run("RunWhenTrue", func(t *testing.T) {
		bot := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "filled", nil
			},
		}
		arr := nanobot.Arrow[Work, string](bot).When(func(w Work) bool { return len(w.Result) == 0 })

		result, err := arr(context.Background(), Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "filled"),
		)
	})

	t.Run("InSeq", func(t *testing.T) {
		calls := []string{}
		botA := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				calls = append(calls, "A")
				return "from-a", nil
			},
		}
		botB := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				calls = append(calls, "B")
				return "from-b", nil
			},
		}

		// first step already done (Result non-empty); second step runs
		seq := nanobot.Seq(
			nanobot.Arrow(botA).When(func(w Work) bool { return len(w.Result) == 0 }),
			nanobot.Arrow(botB).When(func(w Work) bool { return true }),
		)

		result, err := seq(context.Background(), Work{Result: "pre-filled"})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "from-b"),
			it.Equal(len(calls), 1),
			it.Equal(calls[0], "B"),
		)
	})
}

// =============================================================================
// TestWithTask
// =============================================================================

func TestArrWithTask(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		chalk := &MockChalk{}
		ctx := context.WithValue(context.Background(), "io.console.chalkboard", chalk)

		arr := nanobot.Lift(func(ctx context.Context, w Work) (Work, error) {
			if ok, _ := ctx.Value(chalkCtxKey{}).(bool); !ok {
				t.Fatalf("expected chalk sub-context to be used")
			}
			w.Result = "done"
			return w, nil
		}).WithTask("arr-step")

		result, err := arr.Prompt(ctx, Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "done"),
			it.Equal(len(chalk.tasks), 1),
			it.Equal(chalk.tasks[0], "arr-step"),
			it.Equal(chalk.dones, 1),
			it.Equal(len(chalk.failed), 0),
			it.Equal(chalk.subs, 1),
		)
	})

	t.Run("ErrorStillDone", func(t *testing.T) {
		errArr := errors.New("arr error")
		chalk := &MockChalk{}
		ctx := context.WithValue(context.Background(), "io.console.chalkboard", chalk)

		arr := nanobot.Lift(func(ctx context.Context, w Work) (Work, error) {
			if ok, _ := ctx.Value(chalkCtxKey{}).(bool); !ok {
				t.Fatalf("expected chalk sub-context to be used")
			}
			return w, errArr
		}).WithTask("arr-step")

		_, err := arr.Prompt(ctx, Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(
			it.True(errors.Is(err, errArr)),
			it.Equal(len(chalk.tasks), 1),
			it.Equal(chalk.tasks[0], "arr-step"),
			it.Equal(chalk.dones, 1),
			it.Equal(len(chalk.failed), 0),
			it.Equal(chalk.subs, 1),
		)
	})
}

// =============================================================================
// TestReflect
// =============================================================================

func TestReflect(t *testing.T) {
	rt := nanobot.NewRuntime(nil, nil)

	t.Run("AcceptOnFirstAttempt", func(t *testing.T) {
		rawJudge := &MockBot[Work, string]{
			// echo the input — auto-derived lens puts it back, preserving Result
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "corrected", nil
			},
		}

		// no eff: Lens=auto-lens, Eval=always-accept(+1)
		judge := nanobot.Judge(rawJudge)

		bot, err := nanobot.NewReflect(rt, judge, nanobot.Arrow(react))
		it.Then(t).Should(it.Nil(err))

		result, err := bot.Prompt(context.Background(), Work{Result: "initial"})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "initial"),
		)
	})

	t.Run("ImmediateReject", func(t *testing.T) {
		rawJudge := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "bad", nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "fix", nil
			},
		}

		judge := nanobot.Judge(rawJudge,
			nanobot.Eff[nanobot.Vote[Work], string]{
				Eval: func(_ context.Context, v nanobot.Vote[Work]) (nanobot.Vote[Work], error) {
					v.Reject()
					return v, nil
				},
			},
		)

		bot, err := nanobot.NewReflect(rt, judge, nanobot.Arrow(react))
		it.Then(t).Should(it.Nil(err))

		_, err = bot.Prompt(context.Background(), Work{Result: "initial"})
		it.Then(t).ShouldNot(it.Nil(err))
	})

	t.Run("RetryThenAccept", func(t *testing.T) {
		call := 0
		rawJudge := &MockBot[Work, string]{
			// returns different verdicts per call; auto-lens writes them into Work.Result
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				call++
				if call < 2 {
					return "needs-work", nil
				}
				return "accepted", nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "corrected", nil
			},
		}

		judge := nanobot.Judge(rawJudge, nanobot.Eff[nanobot.Vote[Work], string]{
			Eval: func(_ context.Context, v nanobot.Vote[Work]) (nanobot.Vote[Work], error) {
				if v.State.Result == "accepted" {
					v.Accept()
				}
				return v, nil
			},
		})

		bot, err := nanobot.NewReflect(rt, judge, nanobot.Arrow[Work, string](react))
		it.Then(t).Should(it.Nil(err))

		bot = bot.WithAttempts(3)

		result, err := bot.Prompt(context.Background(), Work{Result: "initial"})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "accepted"),
		)
	})

	t.Run("ExhaustsAttempts", func(t *testing.T) {
		rawJudge := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "fix", nil
			},
		}

		// always neutral → exhaust all 2 attempts
		judge := nanobot.Judge[Work, string](rawJudge, nanobot.Eff[nanobot.Vote[Work], string]{
			Eval: func(_ context.Context, v nanobot.Vote[Work]) (nanobot.Vote[Work], error) {
				return v, nil // Accepted stays 0
			},
		})

		bot, err := nanobot.NewReflect(rt, judge, nanobot.Arrow[Work, string](react))
		it.Then(t).Should(it.Nil(err))

		bot = bot.WithAttempts(2)

		_, err = bot.Prompt(context.Background(), Work{Result: "initial"})
		it.Then(t).ShouldNot(it.Nil(err))
	})

	t.Run("JudgeError", func(t *testing.T) {
		errJudge := errors.New("judge error")
		judge := &MockBot[Work, nanobot.Vote[Work]]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (nanobot.Vote[Work], error) {
				return nanobot.Vote[Work]{}, errJudge
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "fix", nil
			},
		}

		bot, err := nanobot.NewReflect(rt, judge, nanobot.Arrow[Work, string](react))
		it.Then(t).Should(it.Nil(err))

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errJudge)))
	})

	t.Run("ReactError", func(t *testing.T) {
		errReact := errors.New("react error")
		rawJudge := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "", errReact
			},
		}

		// neutral → triggers correct arrow
		judge := nanobot.Judge[Work, string](rawJudge, nanobot.Eff[nanobot.Vote[Work], string]{
			Eval: func(_ context.Context, v nanobot.Vote[Work]) (nanobot.Vote[Work], error) {
				return v, nil // Accepted stays 0
			},
		})

		bot, err := nanobot.NewReflect(rt, judge, nanobot.Arrow[Work, string](react))
		it.Then(t).Should(it.Nil(err))

		bot = bot.WithAttempts(2)

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errReact)))
	})

	t.Run("EffectError", func(t *testing.T) {
		errEffect := errors.New("effect error")
		rawJudge := &MockBot[Work, string]{
			fn: func(_ context.Context, w Work, _ ...chatter.Opt) (string, error) {
				return w.Result, nil
			},
		}
		react := &MockBot[Work, string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) (string, error) {
				return "fix", nil
			},
		}

		// neutral → correct arrow runs; effect inside correct Arrow fails
		judge := nanobot.Judge[Work, string](rawJudge, nanobot.Eff[nanobot.Vote[Work], string]{
			Eval: func(_ context.Context, v nanobot.Vote[Work]) (nanobot.Vote[Work], error) {
				return v, nil // Accepted stays 0
			},
		})
		correct := nanobot.Arrow[Work, string](react, nanobot.Eff[Work, string]{
			Eval: func(_ context.Context, w Work) (Work, error) {
				return w, errEffect
			},
		})

		bot, err := nanobot.NewReflect(rt, judge, correct)
		it.Then(t).Should(it.Nil(err))

		bot = bot.WithAttempts(2)

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errEffect)))
	})
}

// =============================================================================
// TestThinkReAct
// =============================================================================

// TaskState is the inner per-task state for ThinkReAct tests.
type TaskState struct{ Value string }

func TestThinkReAct(t *testing.T) {
	rt := nanobot.NewRuntime(nil, nil)

	t.Run("Success", func(t *testing.T) {
		think := &MockBot[Work, []TaskState]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]TaskState, error) {
				return []TaskState{{Value: "task-1"}, {Value: "task-2"}}, nil
			},
		}
		react := nanobot.Arr[TaskState](func(_ context.Context, t TaskState, _ ...chatter.Opt) (TaskState, error) {
			return TaskState{Value: "done:" + t.Value}, nil
		})
		gather := func(s Work, ts []TaskState) Work {
			for _, t := range ts {
				s.Result += t.Value + ";"
			}
			return s
		}

		bot, err := nanobot.NewThinkReAct(rt, think, react, gather)
		it.Then(t).Should(it.Nil(err))

		result, err := bot.Prompt(context.Background(), Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "done:task-1;done:task-2;"),
		)
	})

	t.Run("ThinkFails", func(t *testing.T) {
		errThink := errors.New("think error")
		think := &MockBot[Work, []TaskState]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]TaskState, error) {
				return nil, errThink
			},
		}
		react := nanobot.Arr[TaskState](func(_ context.Context, t TaskState, _ ...chatter.Opt) (TaskState, error) {
			return t, nil
		})
		gather := func(s Work, _ []TaskState) Work { return s }

		bot, err := nanobot.NewThinkReAct(rt, think, react, gather)
		it.Then(t).Should(it.Nil(err))

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errThink)))
	})

	t.Run("ReactFails", func(t *testing.T) {
		errReact := errors.New("react error")
		think := &MockBot[Work, []TaskState]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]TaskState, error) {
				return []TaskState{{Value: "task-1"}}, nil
			},
		}
		react := nanobot.Arr[TaskState](func(_ context.Context, _ TaskState, _ ...chatter.Opt) (TaskState, error) {
			return TaskState{}, errReact
		})
		gather := func(s Work, _ []TaskState) Work { return s }

		bot, err := nanobot.NewThinkReAct(rt, think, react, gather)
		it.Then(t).Should(it.Nil(err))

		_, err = bot.Prompt(context.Background(), Work{})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(it.True(errors.Is(err, errReact)))
	})

	t.Run("WithThink", func(t *testing.T) {
		// Test the Think combinator that scatters Bot[S, []A] into Bot[S, []T]
		planBot := &MockBot[Work, []string]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]string, error) {
				return []string{"t1", "t2"}, nil
			},
		}
		think := nanobot.Think(planBot, func(s Work, task string) TaskState {
			return TaskState{Value: task}
		})
		react := nanobot.Arr[TaskState](func(_ context.Context, t TaskState, _ ...chatter.Opt) (TaskState, error) {
			return TaskState{Value: "done:" + t.Value}, nil
		})
		gather := func(s Work, ts []TaskState) Work {
			for _, t := range ts {
				s.Result += t.Value + ";"
			}
			return s
		}

		bot, err := nanobot.NewThinkReAct(rt, think, react, gather)
		it.Then(t).Should(it.Nil(err))

		result, err := bot.Prompt(context.Background(), Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "done:t1;done:t2;"),
		)
	})

	t.Run("ReactIsComposedPipeline", func(t *testing.T) {
		// react is Arr[T], so it can be a Seq of multiple steps
		think := &MockBot[Work, []TaskState]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]TaskState, error) {
				return []TaskState{{Value: "x"}}, nil
			},
		}
		step1 := nanobot.Arr[TaskState](func(_ context.Context, t TaskState, _ ...chatter.Opt) (TaskState, error) {
			return TaskState{Value: t.Value + "+step1"}, nil
		})
		step2 := nanobot.Arr[TaskState](func(_ context.Context, t TaskState, _ ...chatter.Opt) (TaskState, error) {
			return TaskState{Value: t.Value + "+step2"}, nil
		})
		react := nanobot.Seq(step1, step2)

		gather := func(s Work, ts []TaskState) Work {
			s.Result = ts[0].Value
			return s
		}

		bot, err := nanobot.NewThinkReAct(rt, think, react, gather)
		it.Then(t).Should(it.Nil(err))

		result, err := bot.Prompt(context.Background(), Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "x+step1+step2"),
		)
	})

	t.Run("ReturnsArrS", func(t *testing.T) {
		// ThinkReAct returns Bot[S, S] ≅ Arr[S], so it nests in Seq
		think := &MockBot[Work, []TaskState]{
			fn: func(_ context.Context, _ Work, _ ...chatter.Opt) ([]TaskState, error) {
				return []TaskState{{Value: "a"}}, nil
			},
		}
		react := nanobot.Arr[TaskState](func(_ context.Context, t TaskState, _ ...chatter.Opt) (TaskState, error) {
			return t, nil
		})
		gather := func(s Work, ts []TaskState) Work {
			s.Result = ts[0].Value
			return s
		}

		thinkreact, err := nanobot.NewThinkReAct(rt, think, react, gather)
		it.Then(t).Should(it.Nil(err))

		// BotThinkReAct satisfies Bot[S, S], compose directly via Seq
		suffix := nanobot.Lift(func(_ context.Context, s Work) (Work, error) {
			s.Result += "!"
			return s, nil
		})
		pipeline := nanobot.Seq(
			nanobot.Lift(func(ctx context.Context, s Work) (Work, error) {
				return thinkreact.Prompt(ctx, s)
			}),
			suffix,
		)

		result, err := pipeline(context.Background(), Work{})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.Result, "a!"),
		)
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

// =============================================================================
// TestReActWithTask
// =============================================================================

func TestReActWithTask(t *testing.T) {
	newBot := func(t *testing.T, llm chatter.Chatter) *nanobot.BotReAct[Work, string] {
		t.Helper()

		fs := fstest.MapFS{
			"react.prompt": &fstest.MapFile{
				Data: []byte("Return {{.Result}}"),
			},
		}

		bot, err := nanobot.NewReAct[Work, string](nanobot.NewRuntime(fs, &MockLLMs{
			models: map[string]chatter.Chatter{"base": llm},
		}), "react.prompt")
		it.Then(t).Should(it.Nil(err))

		return bot.WithTask("react-step")
	}

	t.Run("Success", func(t *testing.T) {
		chalk := &MockChalk{}
		ctx := context.WithValue(context.Background(), "io.console.chalkboard", chalk)
		bot := newBot(t, &MockChatter{response: "final answer"})

		result, err := bot.Prompt(ctx, Work{Result: "input"})
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result, "final answer"),
			it.Equal(len(chalk.tasks), 1),
			it.Equal(chalk.tasks[0], "react-step"),
			it.Equal(chalk.dones, 1),
			it.Equal(len(chalk.failed), 0),
		)
	})

	t.Run("FailureCallsFail", func(t *testing.T) {
		errLLM := errors.New("llm failure")
		chalk := &MockChalk{}
		ctx := context.WithValue(context.Background(), "io.console.chalkboard", chalk)
		bot := newBot(t, &MockChatter{err: errLLM})

		_, err := bot.Prompt(ctx, Work{Result: "input"})
		it.Then(t).ShouldNot(it.Nil(err))
		it.Then(t).Should(
			it.True(errors.Is(err, errLLM)),
			it.Equal(len(chalk.tasks), 1),
			it.Equal(chalk.tasks[0], "react-step"),
			it.Equal(chalk.dones, 0),
			it.Equal(len(chalk.failed), 1),
			it.True(errors.Is(chalk.failed[0], errLLM)),
		)
	})
}
