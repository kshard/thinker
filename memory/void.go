//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package memory

import (
	"fmt"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// The void memory does not retain any observations.
type Void struct {
	stratum chatter.Stratum
}

// Create the void memory that does not retain any observations.
func NewVoid(stratum chatter.Stratum) *Void {
	return &Void{stratum: stratum}
}

// intentional the loss of memories, including facts, information and experiences
func (s *Void) Purge() {}

// Commit new observation into memory.
func (s *Void) Commit(e *thinker.Observation) {}

// Builds the context window for LLM using incoming prompt.
func (s *Void) Context(prompt chatter.Prompt) []fmt.Stringer {
	if len(s.stratum) == 0 {
		return prompt.ToSeq()
	}

	return []fmt.Stringer{s.stratum, prompt}
}
