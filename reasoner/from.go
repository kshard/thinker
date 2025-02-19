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
func From[A, B any](f func(thinker.State[A, B]) (thinker.Phase, chatter.Prompt, error)) thinker.Reasoner[A, B] {
	return fromReasoner[A, B](f)
}

type fromReasoner[A, B any] func(thinker.State[A, B]) (thinker.Phase, chatter.Prompt, error)

// Deduct new goal for the agent to pursue.
func (f fromReasoner[A, B]) Deduct(s thinker.State[A, B]) (thinker.Phase, chatter.Prompt, error) {
	return f(s)
}
