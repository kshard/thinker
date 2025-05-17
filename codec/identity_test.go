//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package codec_test

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/codec"
)

func TestEncoderID(t *testing.T) {
	input := "prompt."
	prompt, err := codec.EncoderID.Encode(input)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(prompt.String(), input),
	)
}

func TestDecoderID(t *testing.T) {
	input := "prompt"
	reply := &chatter.Reply{Content: []chatter.Content{chatter.Text(input)}}
	conf, text, err := codec.DecoderID.Decode(reply)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(conf, 1.0),
		it.Equal(text.String(), input),
	)
}
