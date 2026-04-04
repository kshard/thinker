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

// Seq pipelines two bots in sequence over a shared state S. Bot a runs first
// and produces an intermediate output A; that output is merged into S via the
// apply function, the optional effect runs, and then bot b receives the
// updated S and returns the final output B.
//
//	Seq : (S -> A) ∘ (S, A -> S) ∘ (S -> S) ∘ (S -> B)
//
// The apply function (S, A -> S) folds A into S.
// The default apply is a Lens[S, A].
//
// The effect function (S -> S) captures side effects between the two steps.
// The default effect is the identity function.
type Seq[S, A, B any] struct {
	a      Bot[S, A]
	b      Bot[S, B]
	apply  func(S, A) S
	effect func(context.Context, S) (S, error)
}

// MustSeq is like NewSeq but panics on error.
func MustSeq[S, A, B any](rt *Runtime, a Bot[S, A], b Bot[S, B]) *Seq[S, A, B] {
	seq, err := NewSeq(rt, a, b)
	if err != nil {
		panic(err)
	}
	return seq
}

// NewSeq creates a two-step sequential pipeline. Bot a runs on the initial
// state, its output is applied to produce a new state, and then bot b runs on
// that state.
func NewSeq[S, A, B any](rt *Runtime, a Bot[S, A], b Bot[S, B]) (*Seq[S, A, B], error) {
	return &Seq[S, A, B]{
		a:      a,
		b:      b,
		apply:  mustApply[S, A]().Put,
		effect: func(ctx context.Context, s S) (S, error) { return s, nil },
	}, nil
}

// WithApply overrides the default lens-based merge with a custom function
// that folds the first bot's output A into the shared state S.
func (bot *Seq[S, A, B]) WithApply(apply func(S, A) S) *Seq[S, A, B] {
	bot.apply = apply
	return bot
}

// WithEffect registers a side-effect function that runs between the two bot
// steps, for example to validate or enrich the intermediate state.
func (bot *Seq[S, A, B]) WithEffect(effect func(context.Context, S) (S, error)) *Seq[S, A, B] {
	bot.effect = effect
	return bot
}

// Prompt runs the pipeline: invokes bot a, merges the result into the state,
// runs the effect, then invokes bot b and returns its output.
func (bot *Seq[S, A, B]) Prompt(ctx context.Context, input S, opt ...chatter.Opt) (B, error) {
	a, err := bot.a.Prompt(ctx, input, opt...)
	if err != nil {
		return *new(B), err
	}

	s := bot.apply(input, a)
	s, err = bot.effect(ctx, s)
	if err != nil {
		return *new(B), err
	}

	b, err := bot.b.Prompt(ctx, s, opt...)
	if err != nil {
		return *new(B), err
	}
	return b, nil
}
