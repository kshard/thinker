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

// The void reasoner always sets a new goal to return results.
type Void[A, B any] struct{}

// Creates new void reasoner that always sets a new goal to return results.
func NewVoid[A, B any]() *Void[A, B] { return &Void[A, B]{} }

// Deduct new goal for the agent to pursue.
func (Void[A, B]) Deduct(thinker.State[A, B]) (thinker.Phase, chatter.Prompt, error) {
	return thinker.AGENT_RETURN, chatter.Prompt{}, nil
}
