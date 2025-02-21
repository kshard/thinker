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

// From is helper to build Encoder[A] interface from pure function.
func FromEncoder[A any](f func(A) (chatter.Prompt, error)) thinker.Encoder[A] {
	return fEncoder[A](f)
}

type fEncoder[T any] func(T) (chatter.Prompt, error)

func (f fEncoder[T]) Encode(t T) (chatter.Prompt, error) { return f(t) }

// From is helper to build Decoder[B] interface from pure function.
func FromDecoder[B any](f func(chatter.Reply) (float64, B, error)) thinker.Decoder[B] {
	return fDecoder[B](f)
}

type fDecoder[T any] func(chatter.Reply) (float64, T, error)

func (f fDecoder[T]) Decode(t chatter.Reply) (float64, T, error) { return f(t) }
