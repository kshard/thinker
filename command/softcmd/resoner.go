//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package softcmd

import (
	"fmt"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// State of Command reasoner
type StateCmd = thinker.State[CmdOut]

// The cmd (command) reasoner set the goal for agent to execute a single command.
// It returns right after the command return results.
type CmdOne struct{}

var _ thinker.Reasoner[CmdOut] = (*CmdOne)(nil)

// Creates new command reasoner.
func NewReasonerCmd() *CmdOne {
	return &CmdOne{}
}

func (task *CmdOne) Purge() {}

func (task *CmdOne) Deduct(state StateCmd) (thinker.Phase, chatter.Message, error) {
	if state.Feedback != nil && state.Confidence < 1.0 {
		var prompt chatter.Prompt
		prompt.WithTask("Refine the previous operation using the feedback below.")
		prompt.With(state.Feedback)

		return thinker.AGENT_REFINE, &prompt, nil
	}

	if len(state.Reply.Cmd) != 0 {
		return thinker.AGENT_RETURN, nil, nil
	}

	return thinker.AGENT_ABORT, nil, thinker.ErrUnknown
}

//------------------------------------------------------------------------------

// The sequence of cmd (commands) reasoner set the goal for agent to execute a sequence of commands.
// The reason returns only after LLM uses return command.
type CmdSeq struct{}

var _ thinker.Reasoner[CmdOut] = (*CmdSeq)(nil)

func NewReasonerCmdSeq() *CmdSeq {
	return &CmdSeq{}
}

func (task *CmdSeq) Purge() {}

func (task *CmdSeq) Deduct(state StateCmd) (thinker.Phase, chatter.Message, error) {
	if state.Feedback != nil && state.Confidence < 1.0 {
		var prompt chatter.Prompt
		prompt.WithTask("Refine the previous workflow step using the feedback below.")
		prompt.With(state.Feedback)

		return thinker.AGENT_REFINE, &prompt, nil
	}

	if state.Reply.Cmd == RETURN {
		return thinker.AGENT_RETURN, nil, nil
	}

	// the workflow step is completed
	if len(state.Reply.Cmd) != 0 {
		var prompt chatter.Prompt
		prompt.WithTask("Continue the workflow execution.")
		prompt.WithBlob(
			fmt.Sprintf("TOOL:%s has returned:\n", state.Reply.Cmd),
			state.Reply.Output,
		)

		return thinker.AGENT_ASK, &prompt, nil
	}

	return thinker.AGENT_ABORT, nil, thinker.ErrUnknown
}
