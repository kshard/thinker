//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package thinker

import "github.com/kshard/chatter"

// Used by agtent to converts structured object into LLM prompt.
type Encoder[T any] interface {
	// Encodes type T into LLM prompt.
	FMap(T) (chatter.Prompt, error)
}

// Used by agent to converts LLM's reply into structured object.
type Decoder[T any] interface {
	// The transformer "parses" reply into type T and return the confidence about
	// the result on the interval [0, 1]. The function should return the feedback
	// to LLM if reply cannot be processed.
	FMap(chatter.Reply) (float64, T, error)
}

// Creates feedback for the LLM, packaging it as an error for use by the Decoder and Reasoner.
func Feedback(note string, text ...string) error {
	return feedback{chatter.Feedback(note, text...)}
}

type feedback struct{ chatter.Snippet }

func (s feedback) Error() string { return s.String() }
