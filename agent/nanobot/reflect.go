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
	"fmt"

	"github.com/kshard/chatter"
)

// Reflect implements a guard/review loop that forces a "Review" state before
// a "Done" transition (Reflection or Guard Loop). Content is assumed to be
// produced upstream and arrives as the initial state S. The judge bot evaluates
// S first; if the review function signals acceptance the loop returns the
// (optionally enriched) state. Otherwise, the review function also returns the
// feedback-enriched state S' which is passed to the react bot (corrector), and
// the loop retries, up to attempts times.
//
//	Reflect : (S -> A) ∘ review(S,A)→(S,int) ∘ (S -> B) ∘ (S, B -> S) ∘ (S -> S) ∘ retry
//
// The review function (S, A) -> (S, int) combines acceptance check and feedback
// in a single call. Returning a positive int means the verdict is accepted and the returned
// S is the final output. Returning a negative int means the returned S carries the judge's
// critique so the corrector can act on it.
// The default review always accepts and returns S unchanged.
//
// The apply function (S, B -> S) merges the corrector output B into state S.
// The default apply is a Lens[S, B].
//
// The effect function (S -> S) captures side effects after each correction step.
// The default effect is the identity function.
type Reflect[S, A, B any] struct {
	judge    Bot[S, A]
	react    Bot[S, B]
	accept   func(S, A) (S, int)
	apply    func(S, B) S
	effect   func(context.Context, S) (S, error)
	attempts int
	chalk    Chalk
}

// MustReflect is like NewReflect but panics on error.
func MustReflect[S, A, B any](rt *Runtime, judge Bot[S, A], react Bot[S, B]) *Reflect[S, A, B] {
	reflect, err := NewReflect(rt, judge, react)
	if err != nil {
		panic(err)
	}
	return reflect
}

// NewReflect creates a Reflect agent that uses judge to evaluate state S
// and react to correct it when the judge rejects. The default accept
// function always accepts, and the default apply is a type-derived lens.
func NewReflect[S, A, B any](rt *Runtime, judge Bot[S, A], react Bot[S, B]) (*Reflect[S, A, B], error) {
	return &Reflect[S, A, B]{
		react:    react,
		judge:    judge,
		accept:   func(s S, _ A) (S, int) { return s, 1 },
		apply:    mustApply[S, B]().Put,
		effect:   func(ctx context.Context, s S) (S, error) { return s, nil },
		attempts: 1,
		chalk:    rt.Chalk,
	}, nil
}

// WithAccept sets the combined acceptance-and-feedback function. It receives the
// current state S and the judge's verdict A. Return (s, positive int) to accept — s is
// the final output. Return (s, negative int) to reject — s carries the critique so the
// corrector (react) knows why it was rejected.
func (bot *Reflect[S, A, B]) WithAccept(accept func(S, A) (S, int)) *Reflect[S, A, B] {
	bot.accept = accept
	return bot
}

// WithApply overrides the default lens-based merge with a custom function
// that folds the corrector's output B into state S after each rejection.
func (bot *Reflect[S, A, B]) WithApply(apply func(S, B) S) *Reflect[S, A, B] {
	bot.apply = apply
	return bot
}

// WithEffect registers a side-effect function that runs after each
// correction step, for example to persist intermediate state or enforce
// invariants before the next judge evaluation.
func (bot *Reflect[S, A, B]) WithEffect(effect func(context.Context, S) (S, error)) *Reflect[S, A, B] {
	bot.effect = effect
	return bot
}

// WithAttempts sets the maximum number of judge→correct iterations.
func (bot *Reflect[S, A, B]) WithAttempts(attempts int) *Reflect[S, A, B] {
	bot.attempts = attempts
	return bot
}

// Prompt runs the reflection loop. The judge evaluates the current state on
// every iteration; if accepted the final state is returned, if rejected the
// react bot corrects it and the loop retries. An error is returned if the
// state is still rejected after all attempts are exhausted.
func (bot *Reflect[S, A, B]) Prompt(ctx context.Context, input S, opt ...chatter.Opt) (S, error) {
	bot.chalk.Task(ctx, "Reflect (%d attempts)", bot.attempts)

	s := input
	for i := range bot.attempts {
		bot.chalk.Task(bot.chalk.Sub(ctx), "attempt %d of %d", i+1, bot.attempts)

		b, err := bot.judge.Prompt(bot.chalk.Sub(ctx), s, opt...)
		if err != nil {
			bot.chalk.Fail(err)
			return *new(S), err
		}

		s, accepted := bot.accept(s, b)
		switch {
		case accepted > 0:
			bot.chalk.Done()
			bot.chalk.Done()
			return s, nil

		case accepted < 0:
			bot.chalk.Done()
			err := fmt.Errorf("rejected")
			bot.chalk.Fail(err)
			return s, err
		}

		a, err := bot.react.Prompt(bot.chalk.Sub(ctx), s, opt...)
		if err != nil {
			bot.chalk.Done()
			bot.chalk.Fail(err)
			return s, err
		}

		s = bot.apply(s, a)
		s, err = bot.effect(ctx, s)
		if err != nil {
			bot.chalk.Done()
			bot.chalk.Fail(err)
			return s, err
		}

		bot.chalk.Done()
	}

	err := fmt.Errorf("rejected")
	bot.chalk.Fail(err)
	return s, err
}
