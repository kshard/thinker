//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package codec

import (
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// Identity encoder, passes input string directly to prompt
var EncoderID = FromEncoder(
	func(q string) (chatter.Message, error) {
		var prompt chatter.Prompt
		prompt.WithTask(q)
		return &prompt, nil
	},
)

// Identity decoder, passes LLM reply directly as result
var DecoderID = FromDecoder(
	func(reply *chatter.Reply) (float64, *chatter.Reply, error) {
		return 1.0, reply, nil
	},
)

// String Identity codec
const String = s("string")

var (
	_ thinker.Decoder[string] = String
	_ thinker.Encoder[string] = String
)

type s string

func (s s) Encode(q string) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask(q)
	return &prompt, nil
}

func (s s) Decode(reply *chatter.Reply) (float64, string, error) {
	return 1.0, reply.String(), nil
}
