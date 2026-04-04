//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package nanobot

import (
	"context"

	"github.com/kshard/chatter"
)

// Think and then execute the subtasks is a pattern where the agent first plans
// a sequence of actions based on the current state and then executes those
// actions, potentially updating the state after each action.
//
// The pattern is composition of type safe planner and executor agents performed
// over a global state S (e.g. blackboard). The pattern supports side effects on
// the global state S after each action is applied. The pattern is useful for
// implementation of guardrails and determenstic behavior in the agent.
//
//	ThinkReAct : (S -> []A) ∘ (S, A -> S) ∘ (S -> S) ∘ (S -> []B)
//
// The apply function (S, A -> S) modifies S with respect of A.
// The default apply is a Lens[S, A]
//
// The effect function (S -> S) modifies S with respect of the side effects
// of the action A. The default effect is identity function.
type ThinkReAct[S, A, B any] struct {
	think  Bot[S, []A]
	react  Bot[S, B]
	apply  func(S, A) S
	effect func(context.Context, S) (S, error)
	chalk  Chalk
}

// MustThinkReAct is like NewThinkReAct but panics on error.
func MustThinkReAct[S, A, B any](rt *Runtime, think Bot[S, []A], react Bot[S, B]) *ThinkReAct[S, A, B] {
	pte, err := NewThinkReAct(rt, think, react)
	if err != nil {
		panic(err)
	}
	return pte
}

// NewThinkReAct creates a ThinkReAct agent that uses the provided think bot
// to produce the task list and the react bot to execute each task.
func NewThinkReAct[S, A, B any](rt *Runtime, think Bot[S, []A], react Bot[S, B]) (*ThinkReAct[S, A, B], error) {
	return &ThinkReAct[S, A, B]{
		think:  think,
		react:  react,
		apply:  mustApply[S, A]().Put,
		effect: func(ctx context.Context, s S) (S, error) { return s, nil },
		chalk:  rt.Chalk,
	}, nil
}

// WithApply overrides the default lens-based merge with a custom function
// that folds each planned action A into the shared state S before the react
// bot is invoked.
func (bot *ThinkReAct[S, A, B]) WithApply(apply func(S, A) S) *ThinkReAct[S, A, B] {
	bot.apply = apply
	return bot
}

// WithEffect registers a side-effect function that runs after each apply
// step, allowing callers to validate, persist, or enrich the state between
// consecutive react invocations.
func (bot *ThinkReAct[S, A, B]) WithEffect(effect func(context.Context, S) (S, error)) *ThinkReAct[S, A, B] {
	bot.effect = effect
	return bot
}

// Prompt runs the full think-then-react cycle: calls the think bot once to
// obtain the task list, then calls the react bot once per task and returns
// all results in order.
func (bot *ThinkReAct[S, A, B]) Prompt(ctx context.Context, input S, opt ...chatter.Opt) ([]B, error) {
	seq, err := bot.think.Prompt(ctx, input, opt...)
	if err != nil {
		return nil, err
	}

	bot.chalk.Task(ctx, "Execute (%d subtasks)", len(seq))
	ret := make([]B, len(seq))
	for i, item := range seq {
		bot.chalk.Task(bot.chalk.Sub(ctx), "%v", item)

		s := bot.apply(input, item)
		s, err = bot.effect(ctx, s)
		if err != nil {
			bot.chalk.Fail(err)
			return ret, err
		}

		result, err := bot.react.Prompt(bot.chalk.Sub(ctx), s, opt...)
		if err != nil {
			bot.chalk.Fail(err)
			return ret, err
		}
		ret[i] = result
		bot.chalk.Done()
	}
	bot.chalk.Done()

	return ret, nil
}
