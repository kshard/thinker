//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package softcmd

import (
	"testing"

	"github.com/fogfish/it/v2"
)

func TestCodeBlock(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		for in, ex := range map[string]string{
			"<codeblock>a</codeblock>":       "a",
			"xxx<codeblock>a</codeblock>":    "a",
			"<codeblock>a</codeblock>xxx":    "a",
			"xxx<codeblock>a</codeblock>xxx": "a",
		} {
			code, err := CodeBlock(BASH, in)
			it.Then(t).Should(
				it.Nil(err),
				it.Equal(code, ex),
			)
		}
	})

	t.Run("Failed", func(t *testing.T) {
		for _, in := range []string{
			"<codeblock>a",
			"a</codeblock>",
			"</codeblock>a<codeblock>",
			"xxxcodeblock>a</codeblock>",
			"<codeblock>a<codeblock>xxx",
			"xxx<codeblock>acodeblockxxx",
		} {
			_, err := CodeBlock(BASH, in)
			it.Then(t).ShouldNot(
				it.Nil(err),
			)
		}
	})
}
