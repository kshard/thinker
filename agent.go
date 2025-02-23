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

// Execution phase of the agent
type Phase int

const (
	// Agent is asking for new facts from LLM
	AGENT_ASK Phase = iota
	// Agent has a final result to return
	AGENT_RETURN
	// Agent should retry with the same context
	AGENT_RETRY
	// Agent should refine the prompt based on feedback
	AGENT_REFINE
	// Agent aborts processing due to unrecoverable error
	AGENT_ABORT
)

// State of the agent, maintained by the agent and used by Reasoner.
type State[B any] struct {
	// Execution phase of the agent
	Phase Phase

	// Current epoch of execution phase
	Epoch int

	// Reply from LLM
	Reply B

	// Confidence level of obtained results
	Confidence float64

	// Feedback to LLM
	Feedback chatter.Section
}
