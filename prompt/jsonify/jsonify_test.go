//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package jsonify_test

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/prompt/jsonify"
)

func TestStrings(t *testing.T) {
	t.Run("Harden", func(t *testing.T) {
		var prompt chatter.Prompt
		jsonify.Strings.Harden(&prompt)

		str := prompt.String()

		it.Then(t).Should(
			it.String(str).Contain("Strictly adhere to the following requirements"),
			it.String(str).Contain("JSON list of strings"),
			it.String(str).Contain("reply [] if you do not know the answer"),
		)
	})

	t.Run("Decode", func(t *testing.T) {
		var seq []string
		reply := &chatter.Reply{
			Content: []chatter.Content{
				chatter.Text(` ["a", "b", "c"] `),
			},
		}
		err := jsonify.Strings.Decode(reply, &seq)

		it.Then(t).Should(
			it.Nil(err),
			it.Seq(seq).Equal("a", "b", "c"),
		)
	})

	t.Run("DecodeErrors", func(t *testing.T) {
		for in, ex := range map[string]string{
			"abc":       "does not contain valid JSON list of strings",
			"[a, b, c]": "JSON parsing of included list of strings has failed",
		} {
			var seq []string
			reply := &chatter.Reply{
				Content: []chatter.Content{
					chatter.Text(in),
				},
			}
			err := jsonify.Strings.Decode(reply, &seq)

			it.Then(t).Should(
				it.String(err.Error()).Contain(ex),
			)
		}
	})
}
