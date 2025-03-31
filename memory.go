//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package thinker

import (
	"fmt"

	"github.com/fogfish/guid/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/float8"
)

// Memory is core element of agents behaviour. It is a database that
// maintains a comprehensive record of an agent’s experience.
// It recalls observations and builds the context windows to be used for prompting.
//
// See package `memory` that implements various algorithms
type Memory interface {
	// intentional the loss of memories, including facts, information and experiences
	Purge()

	// Commit new observation into memory.
	Commit(*Observation)

	// Builds the context window for LLM using incoming prompt.
	Context(chatter.Prompt) []fmt.Stringer
}

// The observation made by agent, it contains LLMs prompt, reply, environment
// status and other metadata.
type Observation struct {
	Created  guid.K
	Accessed guid.K
	Query    Input
	Reply    Reply
}

// Create new observation
func NewObservation(query chatter.Prompt, reply chatter.Reply) *Observation {
	return &Observation{
		Created: guid.G(guid.Clock),
		Query:   Input{Content: query},
		Reply:   Reply{Content: reply},
	}
}

// Input and its relevance vector (embeddings) as observed by agent
type Input struct {
	Content   chatter.Prompt
	Relevance []float8.Float8
}

// Reply and its relevance vector (embeddings) as observed by agent
type Reply struct {
	Content    chatter.Reply
	Relevance  []float8.Float8
	Importance float64
}
