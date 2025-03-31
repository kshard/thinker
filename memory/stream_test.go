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
	s := NewStream(2, "role.")

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

	t.Run("Evict", func(t *testing.T) {
		a := thinker.NewObservation(chatter.Prompt{Task: "0."}, chatter.Reply{Text: "0."})
		s.Commit(a)

		b := thinker.NewObservation(chatter.Prompt{Task: "a."}, chatter.Reply{Text: "a."})
		s.Commit(b)

		c := thinker.NewObservation(chatter.Prompt{Task: "b."}, chatter.Reply{Text: "b."})
		s.Commit(c)

		seq := make([]string, 0)
		for _, x := range s.Context(chatter.Prompt{Task: "c."}) {
			seq = append(seq, x.String())
		}

		it.Then(t).Should(
			it.Seq(seq).Equal("role.", "a.", "a.", "b.", "b.", "c."),
		)
	})

	t.Run("Purge", func(t *testing.T) {
		a := thinker.NewObservation(chatter.Prompt{Task: "a."}, chatter.Reply{Text: "a."})
		s.Commit(a)

		b := thinker.NewObservation(chatter.Prompt{Task: "b."}, chatter.Reply{Text: "b."})
		s.Commit(b)

		seq := make([]string, 0)
		for _, x := range s.Context(chatter.Prompt{Task: "c."}) {
			seq = append(seq, x.String())
		}

		it.Then(t).Should(
			it.Seq(seq).Equal("role.", "a.", "a.", "b.", "b.", "c."),
		)

		s.Purge()

		seq = make([]string, 0)
		for _, x := range s.Context(chatter.Prompt{Task: "c."}) {
			seq = append(seq, x.String())
		}
		it.Then(t).Should(
			it.Seq(seq).Equal("role.", "c."),
		)
	})

}
