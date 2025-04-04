//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package reasoner

import (
	"fmt"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/command"
)

// State of Command reasoner
type StateCmd = thinker.State[thinker.CmdOut]

// The cmd (command) reasoner set the goal for agent to execute a single command.
// It returns right after the command return results.
type Cmd struct{}

// Creates new command reasoner.
func NewCmd() *Cmd {
	return &Cmd{}
}

func (task *Cmd) Purge() {}

func (task *Cmd) Deduct(state StateCmd) (thinker.Phase, chatter.Prompt, error) {
	if state.Feedback != nil && state.Confidence < 1.0 {
		var prompt chatter.Prompt
		prompt.WithTask("Refine the previous operation using the feedback below.")
		prompt.With(state.Feedback)

		return thinker.AGENT_REFINE, prompt, nil
	}

	if len(state.Reply.Cmd) != 0 {
		return thinker.AGENT_RETURN, chatter.Prompt{}, nil
	}

	return thinker.AGENT_ABORT, chatter.Prompt{}, thinker.ErrUnknown
}

//------------------------------------------------------------------------------

// The sequence of cmd (commands) reasoner set the goal for agent to execute a sequence of commands.
// The reason returns only after LLM uses return command.
type CmdSeq struct{}

func NewCmdSeq() *CmdSeq {
	return &CmdSeq{}
}

func (task *CmdSeq) Purge() {}

func (task *CmdSeq) Deduct(state StateCmd) (thinker.Phase, chatter.Prompt, error) {
	if state.Feedback != nil && state.Confidence < 1.0 {
		var prompt chatter.Prompt
		prompt.WithTask("Refine the previous workflow step using the feedback below.")
		prompt.With(state.Feedback)

		return thinker.AGENT_REFINE, prompt, nil
	}

	if state.Reply.Cmd == command.RETURN {
		return thinker.AGENT_RETURN, chatter.Prompt{}, nil
	}

	// the workflow step is completed
	if len(state.Reply.Cmd) != 0 {
		var prompt chatter.Prompt
		prompt.WithTask("Continue the workflow execution.")
		prompt.With(
			chatter.Blob(
				fmt.Sprintf("TOOL:%s has returned:\n", state.Reply.Cmd),
				state.Reply.Output,
			),
		)

		return thinker.AGENT_ASK, prompt, nil
	}

	return thinker.AGENT_ABORT, chatter.Prompt{}, thinker.ErrUnknown
}
