//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package nanobot

import (
	"context"

	"github.com/kshard/chatter"
)

// Seq composes Kleisli arrows left-to-right over a shared state S.
//
//	Seq(f, g, h) = f >=> g >=> h
//	  : s ↦ let s₁ = f(s), s₂ = g(s₁) in h(s₂)
//
// Each step receives the state produced by the previous one; the first error
// short-circuits the pipeline and the state accumulated thus far is returned.
// Because Arr[S] satisfies Bot[S, S], the returned Arr[S] composes freely
// with any other Arr or Bot in the algebra.
//
// Individual arrows are constructed with Arrow, which lifts a Bot[S, A] into
// Arr[S] by applying an Eff (Lens setter + Eval side-effect).
func Seq[S any](steps ...Arr[S]) Arr[S] {
	return func(ctx context.Context, s S, opt ...chatter.Opt) (S, error) {
		for _, step := range steps {
			var err error
			s, err = step(ctx, s, opt...)
			if err != nil {
				return s, err
			}
		}
		return s, nil
	}
}

// NewSeq is like Seq but returns an error for API consistency with other
// compositors.  The error is always nil.
func NewSeq[S any](steps ...Arr[S]) (Arr[S], error) {
	return Seq(steps...), nil
}
