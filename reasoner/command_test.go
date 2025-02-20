//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package reasoner_test

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/command"
	"github.com/kshard/thinker/reasoner"
)

func TestCmdTaskDeduct(t *testing.T) {
	r := reasoner.NewCmdTask[string]()

	t.Run("Refine", func(t *testing.T) {
		phase, prompt, err := r.Deduct(thinker.State[string, thinker.CmdOut]{
			Feedback:   chatter.Feedback("feedback"),
			Confidence: 0.1,
		})

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_REFINE),
			it.String(prompt.String()).Contain("Refine the previous workflow step"),
		)
	})

	t.Run("Return", func(t *testing.T) {
		phase, _, err := r.Deduct(thinker.State[string, thinker.CmdOut]{
			Reply: thinker.CmdOut{Cmd: command.RETURN},
		})

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_RETURN),
		)
	})

	t.Run("Continue", func(t *testing.T) {
		phase, prompt, err := r.Deduct(thinker.State[string, thinker.CmdOut]{
			Reply: thinker.CmdOut{Cmd: command.BASH, Output: "Bash Output"},
		})

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_ASK),
			it.String(prompt.String()).Contain("Continue the workflow execution."),
			it.String(prompt.String()).Contain("Bash Output"),
		)
	})

	t.Run("Abort", func(t *testing.T) {
		phase, _, _ := r.Deduct(thinker.State[string, thinker.CmdOut]{})

		it.Then(t).Should(
			it.Equal(phase, thinker.AGENT_ABORT),
		)
	})

}
