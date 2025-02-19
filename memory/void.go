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
type Void struct{}

// Create the void memory that does not retain any observations.
func NewVoid() *Void { return &Void{} }

// Commit new observation into memory.
func (s *Void) Commit(e *thinker.Observation) {}

// Builds the context window for LLM using incoming prompt.
func (s *Void) Context(prompt chatter.Prompt) []fmt.Stringer { return prompt.ToSeq() }
