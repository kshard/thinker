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

// The epoch reasoner limits the number of agent iterations per step.
// It aborts execution if the maximum limit is reached.
type Epoch[A, B any] struct {
	thinker.Reasoner[A, B]
	max int
}

// Creates new epoch reasoner that limits the number of agent iterations per step.
func NewEpoch[A, B any](max int, reasoner thinker.Reasoner[A, B]) Epoch[A, B] {
	return Epoch[A, B]{Reasoner: reasoner, max: max}
}

// Deduct new goal for the agent to pursue.
func (epoch Epoch[A, B]) Deduct(state thinker.State[A, B]) (thinker.Phase, chatter.Prompt, error) {
	if state.Epoch >= epoch.max {
		return thinker.AGENT_ABORT, chatter.Prompt{}, thinker.ErrMaxEpoch.With(thinker.ErrAbout, state.Epoch)
	}

	return epoch.Reasoner.Deduct(state)
}
