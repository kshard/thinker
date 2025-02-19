package reasoner

import (
	"fmt"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/command"
)

type CmdTask[A any] struct{}

func NewCmdTask[A any]() *CmdTask[A] {
	return &CmdTask[A]{}
}

func (task *CmdTask[A]) Deduct(state thinker.State[A, thinker.CmdOut]) (thinker.Phase, chatter.Prompt, error) {
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
