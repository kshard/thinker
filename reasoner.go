//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package thinker

import (
	"github.com/kshard/chatter"
)

// The Reasoner serves as the goal-setting component in the architecture.
// It evaluates the agent's current state, performing either deterministic
// or non-deterministic analysis of immediate results and past experiences.
// Based on this assessment, it determines whether the goal has been achieved
// and, if not, suggests the best new goal for the agent to pursue.
type Reasoner[A, B any] interface {
	// Deduct new goal for the agent to pursue.
	Deduct(State[A, B]) (Phase, chatter.Prompt, error)
}
