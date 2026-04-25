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

// Vote is the result type for the judge bot in BotReflect. It carries both
// the updated blackboard S — with any feedback from the judge injected — and
// the accept/reject decision signal.
//
// Build a Bot[S, Vote[S]] from an ordinary Bot[S, A] using Accept:
//
//	Accept(judge, Eff[Vote[S], A]{
//	    Eval: func(_ context.Context, v Vote[S]) (Vote[S], error) {
//	        v.Accepted = signal(v.State)
//	        return v, nil
//	    },
//	})
//
// where signal is +1 (accept), 0 (retry), or -1 (hard reject).
type Vote[S any] struct {
	// State is the blackboard updated with the judge's feedback. On a neutral
	// verdict (Accepted == 0) this state — carrying the critique — is forwarded
	// to the corrector so it knows why the previous attempt was rejected.
	State S

	// Accepted is the decision signal produced by the judge:
	//   +1  accept  — State is returned as the final blackboard.
	//    0  retry   — State carries critique; the corrector runs next.
	//   -1  reject  — hard reject; error is returned immediately.
	accepted int
}

func (v *Vote[S]) Accept() { v.accepted = 1 }
func (v *Vote[S]) Reject() { v.accepted = -1 }
func (v *Vote[S]) Revise() { v.accepted = 0 }

type botAccept[S, A any] struct {
	bot  Bot[S, A]
	lens func(Vote[S], A) Vote[S]
	eval Eval[Vote[S]]
}

func (m *botAccept[S, A]) Prompt(ctx context.Context, s S, opt ...chatter.Opt) (Vote[S], error) {
	a, err := m.bot.Prompt(ctx, s, opt...)
	if err != nil {
		return Vote[S]{}, err
	}
	return m.eval(ctx, m.lens(Vote[S]{State: s}, a))
}

// Judge lifts Bot[S, A] into Bot[S, Vote[S]] using Eff[Vote[S], A] — the
// same bundle used by Arrow, instantiated at the product type Vote[S]:
//
//   - eff.Lens injects A into Vote[S]. The default is the composed lens
//     Vote[S].State ∘ forProduct1[S, A], which writes A into the S field of
//     Vote and leaves Vote.Accepted at its zero value.
//   - eff.Eval sets Vote.Accepted (+1 accept, 0 retry, -1 reject). The default
//     is always-accept (+1).
//
// Denotation:
//
//	Judge(bot, eff) : s ↦ eff.Eval(eff.Lens(Vote[S]{State: s}, bot.Prompt(s)))
func Judge[S, A any](b Bot[S, A], eff ...Eff[Vote[S], A]) Bot[S, Vote[S]] {
	var lens func(Vote[S], A) Vote[S]
	var eval Eval[Vote[S]]

	if len(eff) > 0 {
		lens = eff[0].Lens
		eval = eff[0].Eval
	}
	if lens == nil {
		l := mustLens[S, A]()
		lens = func(v Vote[S], a A) Vote[S] {
			v.State = l.Put(v.State, a)
			return v
		}
	}
	if eval == nil {
		eval = func(_ context.Context, v Vote[S]) (Vote[S], error) {
			v.accepted = 1
			return v, nil
		}
	}

	return &botAccept[S, A]{bot: b, lens: lens, eval: eval}
}

// BotReflect implements a guard/review loop that forces a "Review" state before
// a "Done" transition (reflection or guard loop). Content arrives as the initial
// state S. The judge bot evaluates S and returns a Verdict[S] that combines
// the accept/reject signal with the state updated to include any judge feedback.
// On acceptance the Verdict.State is returned. On a neutral verdict the correct
// Arr[S] is invoked on Verdict.State and the loop retries, up to attempts times.
//
//	BotReflect : (S → Verdict[S]) ∘ correct(S ⇝ S) ∘ retry
//
// correct is an Arr[S] (Kleisli endomorphism S ⇝ S) that carries its own
// Lens setter and Eval side-effect, constructed via Arrow.
//
// The judge Bot[S, Verdict[S]] is typically built from a raw Bot[S, A] via Map.
type BotReflect[S any] struct {
	judge    Bot[S, Vote[S]]
	correct  Arr[S]
	attempts int
}

// Reflect creates a BotReflect. Panics on error.
func Reflect[S any](rt *Runtime, judge Bot[S, Vote[S]], correct Arr[S]) *BotReflect[S] {
	bot, err := NewReflect(rt, judge, correct)
	if err != nil {
		panic(err)
	}
	return bot
}

// NewReflect creates a BotReflect that uses judge to evaluate state S and
// correct to fix it on rejection. The default attempt count is 1.
func NewReflect[S any](rt *Runtime, judge Bot[S, Vote[S]], correct Arr[S]) (*BotReflect[S], error) {
	return &BotReflect[S]{
		judge:    judge,
		correct:  correct,
		attempts: 1,
	}, nil
}

// WithAttempts sets the maximum number of judge→correct iterations.
func (bot *BotReflect[S]) WithAttempts(attempts int) *BotReflect[S] {
	bot.attempts = attempts
	return bot
}

// Prompt runs the reflection loop. The judge evaluates the current state on
// every iteration. On acceptance Verdict.State is returned. On a neutral
// verdict Verdict.State (carrying the judge's critique) is forwarded to the
// correct arrow and the loop retries. An error is returned on a hard reject
// or when the retry budget is exhausted.
func (bot *BotReflect[S]) Prompt(ctx context.Context, input S, opt ...chatter.Opt) (S, error) {
	s := input
	for range bot.attempts {
		v, err := bot.judge.Prompt(ctx, s, opt...)
		if err != nil {
			return *new(S), err
		}

		switch {
		case v.accepted > 0:
			return v.State, nil

		case v.accepted < 0:
			err := fmt.Errorf("rejected")
			return v.State, err
		}

		// neutral: v.State carries the critique — forward to corrector
		s, err = bot.correct(ctx, v.State, opt...)
		if err != nil {
			return s, err
		}
	}

	err := fmt.Errorf("rejected")
	return s, err
}
