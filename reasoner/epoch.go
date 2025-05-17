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
type Epoch[B any] struct {
	thinker.Reasoner[B]
	max int
}

// Creates new epoch reasoner that limits the number of agent iterations per step.
func NewEpoch[B any](max int, reasoner thinker.Reasoner[B]) Epoch[B] {
	return Epoch[B]{Reasoner: reasoner, max: max}
}

// Deduct new goal for the agent to pursue.
func (epoch Epoch[B]) Deduct(state thinker.State[B]) (thinker.Phase, chatter.Message, error) {
	if state.Epoch >= epoch.max {
		return thinker.AGENT_ABORT, nil, thinker.ErrMaxEpoch.With(thinker.ErrAborted, state.Epoch)
	}

	return epoch.Reasoner.Deduct(state)
}
