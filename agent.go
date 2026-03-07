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
	"log/slog"

	"github.com/kshard/chatter"
	"github.com/kshard/chatter/provider/autoconfig"
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
	Feedback chatter.Content
}

// LLMs configurations availabe to agents
type LLM struct {
	Base   chatter.Chatter
	Micro  chatter.Chatter
	Small  chatter.Chatter
	Medium chatter.Chatter
	Large  chatter.Chatter
	XLarge chatter.Chatter
}

func ConfigLLM(rc string) LLM {
	var (
		llm LLM
		err error
	)

	if llm.Base, err = autoconfig.FromFile(rc, "base"); err != nil {
		panic(fmt.Errorf("unable to config base llm: %w", err))
	}

	if llm.Micro, err = autoconfig.FromFile(rc, "micro"); err != nil {
		slog.Warn("unable to config micro llm", "error", err)
		llm.Micro = llm.Base
	}

	if llm.Small, err = autoconfig.FromFile(rc, "small"); err != nil {
		slog.Warn("unable to config small llm", "error", err)
		llm.Small = llm.Base
	}

	if llm.Medium, err = autoconfig.FromFile(rc, "medium"); err != nil {
		slog.Warn("unable to config medium llm", "error", err)
		llm.Medium = llm.Base
	}

	if llm.Large, err = autoconfig.FromFile(rc, "large"); err != nil {
		slog.Warn("unable to config large llm", "error", err)
		llm.Large = llm.Base
	}

	if llm.XLarge, err = autoconfig.FromFile(rc, "xlarge"); err != nil {
		slog.Warn("unable to config xlarge llm", "error", err)
		llm.XLarge = llm.Base
	}

	return llm
}
