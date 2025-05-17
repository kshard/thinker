//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package softcmd_test

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/command/softcmd"
)

func TestCmdDeduct(t *testing.T) {
	r := softcmd.NewReasonerCmd()

	t.Run("Refine", func(t *testing.T) {
		phase, prompt, err := r.Deduct(softcmd.StateCmd{
			Feedback:   &chatter.Feedback{Note: "feedback"},
			Confidence: 0.1,
		})

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_REFINE),
			it.String(prompt.String()).Contain("Refine the previous operation"),
		)
	})

	t.Run("Return", func(t *testing.T) {
		phase, _, err := r.Deduct(softcmd.StateCmd{
			Reply: softcmd.CmdOut{Cmd: softcmd.BASH, Output: "Bash Output"},
		})

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_RETURN),
		)
	})

	t.Run("Abort", func(t *testing.T) {
		phase, _, _ := r.Deduct(softcmd.StateCmd{})

		it.Then(t).Should(
			it.Equal(phase, thinker.AGENT_ABORT),
		)
	})
}

func TestCmdSeqDeduct(t *testing.T) {
	r := softcmd.NewReasonerCmdSeq()

	t.Run("Refine", func(t *testing.T) {
		phase, prompt, err := r.Deduct(softcmd.StateCmd{
			Feedback:   &chatter.Feedback{Note: "feedback"},
			Confidence: 0.1,
		})

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_REFINE),
			it.String(prompt.String()).Contain("Refine the previous workflow step"),
		)
	})

	t.Run("Return", func(t *testing.T) {
		phase, _, err := r.Deduct(softcmd.StateCmd{
			Reply: softcmd.CmdOut{Cmd: softcmd.RETURN},
		})

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_RETURN),
		)
	})

	t.Run("Continue", func(t *testing.T) {
		phase, prompt, err := r.Deduct(softcmd.StateCmd{
			Reply: softcmd.CmdOut{Cmd: softcmd.BASH, Output: "Bash Output"},
		})

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_ASK),
			it.String(prompt.String()).Contain("Continue the workflow execution."),
			it.String(prompt.String()).Contain("Bash Output"),
		)
	})

	t.Run("Abort", func(t *testing.T) {
		phase, _, _ := r.Deduct(softcmd.StateCmd{})

		it.Then(t).Should(
			it.Equal(phase, thinker.AGENT_ABORT),
		)
	})
}
