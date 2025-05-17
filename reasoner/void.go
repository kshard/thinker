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
type Void[B any] struct{}

// Creates new void reasoner that always sets a new goal to return results.
func NewVoid[B any]() *Void[B] { return &Void[B]{} }

func (Void[B]) Purge() {}

// Deduct new goal for the agent to pursue.
func (Void[B]) Deduct(thinker.State[B]) (thinker.Phase, chatter.Message, error) {
	return thinker.AGENT_RETURN, nil, nil
}
