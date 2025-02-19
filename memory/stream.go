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
	"sync"

	"github.com/fogfish/guid/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// The stream memory retains all of the agent's observations in the time ordered sequence.
type Stream struct {
	mu      sync.Mutex
	heap    map[guid.K]*thinker.Observation
	commits []guid.K
	stratum chatter.Stratum
}

// Creates new stream memory that retains all of the agent's observations.
func NewStream(stratum chatter.Stratum) *Stream {
	return &Stream{
		heap:    make(map[guid.K]*thinker.Observation),
		commits: make([]guid.K, 0),
		stratum: stratum,
	}
}

// Commit new observation into memory.
func (s *Stream) Commit(e *thinker.Observation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.heap[e.Created] = e
	s.commits = append(s.commits, e.Created)
}

// Builds the context window for LLM using incoming prompt.
func (s *Stream) Context(prompt chatter.Prompt) []fmt.Stringer {
	seq := make([]fmt.Stringer, 0)
	if len(s.stratum) > 0 {
		seq = append(seq, s.stratum)
	}
	for _, id := range s.commits {
		evidence := s.heap[id]
		evidence.Accessed = guid.G(guid.Clock)

		seq = append(seq, evidence.Query.Content)
		seq = append(seq, evidence.Reply.Content)
	}

	seq = append(seq, prompt)

	return seq
}
