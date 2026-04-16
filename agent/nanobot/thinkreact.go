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

// Think lifts Bot[S, []A] into Bot[S, []T] by applying a scatter function
// to each element of the task list. The scatter function projects the outer
// state S and each task item A into the inner per-task state T.
//
//	Think(bot, σ) : s ↦ bot(s) >>= λ[a₁,…,aₙ]. η([σ(s,a₁),…,σ(s,aₙ)])
func Think[S, A, T any](bot Bot[S, []A], scatter func(S, A) T) Bot[S, []T] {
	return &botThink[S, A, T]{bot: bot, scatter: scatter}
}

type botThink[S, A, T any] struct {
	bot     Bot[S, []A]
	scatter func(S, A) T
}

func (b *botThink[S, A, T]) Prompt(ctx context.Context, s S, opt ...chatter.Opt) ([]T, error) {
	tasks, err := b.bot.Prompt(ctx, s, opt...)
	if err != nil {
		return nil, err
	}

	result := make([]T, len(tasks))
	for i, a := range tasks {
		result[i] = b.scatter(s, a)
	}
	return result, nil
}

// BotThinkReAct implements the scatter/gather (plan-and-execute) pattern.
//
//	ThinkReAct : Bot[S, []T] × Arr[T] × (S × []T → S)? → Arr[S]
//
// The think Bot[S, []T] produces the task list from the outer state S. Each
// task item is already in the inner per-task state T (use Think to scatter
// from Bot[S, []A] if needed). The react Arr[T] is a full Kleisli arrow —
// a composed pipeline (Seq, Reflect, nested ThinkReAct) in the inner state
// space. The gather fold merges the per-task results []T back into the outer
// state S. If no gather is provided, the default is the auto-derived
// Lens[S, []T].
//
// Unlike Seq (sequential Kleisli composition), each task starts from an
// independent projection of the original input, not from the output of the
// previous task. Results are gathered by the fold function.
//
// Denotation:
//
//	ThinkReAct(p, r, γ)(s) = γ(s, [r(tᵢ) | tᵢ ∈ p(s)])
type BotThinkReAct[S, T any] struct {
	think  Bot[S, []T]
	react  Arr[T]
	gather func(S, []T) S
	chalk  Chalk
}

// ThinkReAct creates a BotThinkReAct. Panics on error. If no gather function
// is provided, the default Lens[S, []T] is used.
func ThinkReAct[S, T any](rt *Runtime, think Bot[S, []T], react Arr[T], gather ...func(S, []T) S) *BotThinkReAct[S, T] {
	bot, err := NewThinkReAct(rt, think, react, gather...)
	if err != nil {
		panic(err)
	}
	return bot
}

// NewThinkReAct creates a BotThinkReAct. If no gather function is provided,
// the default Lens[S, []T] is used.
func NewThinkReAct[S, T any](rt *Runtime, think Bot[S, []T], react Arr[T], gather ...func(S, []T) S) (*BotThinkReAct[S, T], error) {
	var g func(S, []T) S
	if len(gather) > 0 && gather[0] != nil {
		g = gather[0]
	} else {
		lens := mustLens[S, []T]()
		g = func(s S, ts []T) S { return lens.Put(s, ts) }
	}

	return &BotThinkReAct[S, T]{
		think:  think,
		react:  react,
		gather: g,
		chalk:  rt.Chalk,
	}, nil
}

// Prompt runs the full think-then-react cycle: calls the think bot once to
// obtain the task list, then runs the react arrow on each task independently,
// and gathers the results back into the outer state S.
func (bot *BotThinkReAct[S, T]) Prompt(ctx context.Context, input S, opt ...chatter.Opt) (S, error) {
	tasks, err := bot.think.Prompt(ctx, input, opt...)
	if err != nil {
		return *new(S), err
	}

	bot.chalk.Task(ctx, "Execute (%d subtasks)", len(tasks))
	results := make([]T, len(tasks))
	for i, task := range tasks {
		bot.chalk.Task(bot.chalk.Sub(ctx), "%d of %d", i+1, len(tasks))

		t, err := bot.react(ctx, task, opt...)
		if err != nil {
			bot.chalk.Fail(err)
			return *new(S), err
		}
		results[i] = t
		bot.chalk.Done()
	}
	bot.chalk.Done()

	return bot.gather(input, results), nil
}
