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

	"github.com/fogfish/golem/optics"
	"github.com/kshard/chatter"
)

// ForEach applies an Arr[T] to each element of a sequence in the state S.
// It is a special case of ThinkReAct where the scatter function is the identity
// and the gather function is the auto-derived Lens[S, []T].
func ForEach[S, T any](arrT Arr[T]) Arr[S] {
	st := optics.ForProduct1[S, []T]()

	return func(ctx context.Context, s S, opt ...chatter.Opt) (S, error) {
		seq := st.Get(&s)

		for i := range seq {
			t, err := arrT(ctx, seq[i], opt...)
			if err != nil {
				return s, err
			}
			seq[i] = t
		}

		st.Put(&s, seq)
		return s, nil
	}
}
