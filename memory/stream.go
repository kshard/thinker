//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package memory

import (
	"sync"

	"github.com/fogfish/guid/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

const INFINITE = -1

// The stream memory retains all of the agent's observations in the time ordered sequence.
type Stream struct {
	mu      sync.Mutex
	heap    map[guid.K]*thinker.Observation
	commits []guid.K
	stratum chatter.Stratum
	cap     int
}

var _ thinker.Memory = (*Stream)(nil)

// Creates new stream memory that retains all of the agent's observations.
func NewStream(cap int, stratum chatter.Stratum) *Stream {
	return &Stream{
		heap:    make(map[guid.K]*thinker.Observation),
		commits: make([]guid.K, 0),
		stratum: stratum,
		cap:     cap,
	}
}

func (s *Stream) Purge() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.heap = make(map[guid.K]*thinker.Observation)
	s.commits = make([]guid.K, 0)
}

// Commit new observation into memory.
func (s *Stream) Commit(e *thinker.Observation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.heap[e.Created] = e
	s.commits = append(s.commits, e.Created)

	if s.cap > 0 && len(s.commits) > s.cap {
		head := s.commits[0]
		s.commits = s.commits[1:]
		delete(s.heap, head)
	}
}

// Builds the context window for LLM using incoming prompt.
func (s *Stream) Context(prompt chatter.Message) []chatter.Message {
	seq := make([]chatter.Message, 0)
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
