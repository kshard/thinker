//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	t.Run("Register", func(t *testing.T) {
		cmd := Return()
		err := r.Register(cmd)

		it.Then(t).Should(
			it.Nil(err),
		)
	})

	t.Run("Conflict", func(t *testing.T) {
		cmd := Return()
		err := r.Register(cmd)

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})

	t.Run("Harden", func(t *testing.T) {
		var prompt chatter.Prompt
		r.Harden(&prompt)

		str := prompt.String()
		cmd := Return()

		it.Then(t).Should(
			it.String(str).Contain("TOOL:"+cmd.Cmd),
			it.String(str).Contain(cmd.Short),
			it.String(str).Contain(cmd.Syntax),
		)
	})

	t.Run("FMap", func(t *testing.T) {
		conf, out, err := r.FMap(chatter.Reply{Text: "TOOL:return hello world\n"})

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(conf, 1.0),
			it.Equal(out.Cmd, "return"),
			it.Equal(out.Output, "hello world"),
		)
	})

	t.Run("FMapNoTool", func(t *testing.T) {
		conf, _, err := r.FMap(chatter.Reply{Text: "TOOL:foo\n"})

		it.Then(t).ShouldNot(
			it.Nil(err),
		)

		it.Then(t).Should(
			it.Equal(conf, 0.0),
			it.String(err.Error()).Contain("The output does not contain valid reference to the tool."),
		)
	})
}
