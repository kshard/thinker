//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package memory

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

func TestStream(t *testing.T) {
	s := NewStream("role.")

	t.Run("Commit", func(t *testing.T) {
		var prompt chatter.Prompt
		prompt.WithTask("prompt.")

		f := thinker.NewObservation(prompt, chatter.Reply{Text: "reply."})
		s.Commit(f)

		it.Then(t).Should(
			it.Seq(s.commits).Equal(f.Created),
			it.Map(s.heap).Have(f.Created, f),
		)
	})

	t.Run("Context", func(t *testing.T) {
		var prompt chatter.Prompt
		prompt.WithTask("ask.")

		seq := make([]string, 0)
		for _, x := range s.Context(prompt) {
			seq = append(seq, x.String())
		}

		it.Then(t).Should(
			it.Seq(seq).Equal("role.", "prompt.", "reply.", "ask."),
		)
	})
}
