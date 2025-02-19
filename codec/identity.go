//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package codec

import "github.com/kshard/chatter"

// Identity encoder, passes input string directly to prompt
var EncoderID = FromEncoder(
	func(q string) (prompt chatter.Prompt, err error) {
		prompt.WithTask(q)
		return
	},
)

// Identity decoder, passes LLM reply directly as result
var DecoderID = FromDecoder(
	func(reply chatter.Reply) (float64, string, error) {
		return 1.0, reply.Text, nil
	},
)
