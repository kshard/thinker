//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package reasoner

import (
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// From is helper to build Reasoner[A, B any] interface from pure function.
func From[B any](f func(thinker.State[B]) (thinker.Phase, chatter.Prompt, error)) thinker.Reasoner[B] {
	return fromReasoner[B](f)
}

type fromReasoner[B any] func(thinker.State[B]) (thinker.Phase, chatter.Prompt, error)

func (f fromReasoner[B]) Purge() {}

// Deduct new goal for the agent to pursue.
func (f fromReasoner[B]) Deduct(s thinker.State[B]) (thinker.Phase, chatter.Prompt, error) {
	return f(s)
}
