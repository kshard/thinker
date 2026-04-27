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
	"reflect"
	"sync"

	"github.com/fogfish/golem/optics"
	"github.com/kshard/chatter"
)

// Hoist lifts an Arr[T] into Arr[S] by applying an isomorphism between S and T.
type Hoister[S, T any] interface{ Hoist(Arr[T]) Arr[S] }

// Codec creates a partial morphism between S and T.
// It allows us to treat S and T as interchangeable for the purpose of applying Arr[T] to S.
func Codec[S, T, A, B any]() Hoister[S, T] {
	return iso[S, T]{iso: optics.CodecS1T1[S, T, A, B]()}
}

// Morph creates a partial morphism between S and T using a custom isomorphism.
func Morph[S, T any](m optics.Isomorphism[S, T]) Hoister[S, T] {
	return iso[S, T]{iso: m}
}

type iso[S, T any] struct {
	iso optics.Isomorphism[S, T]
}

// Hoist lifts an Arr[T] into Arr[S] using the isomorphism defined by the iso struct.
func (iso iso[S, T]) Hoist(arrT Arr[T]) Arr[S] {
	return func(ctx context.Context, s S, opt ...chatter.Opt) (S, error) {
		t := alloc[T]()

		iso.iso.Forward(&s, &t)

		t, err := arrT(ctx, t, opt...)
		if err != nil {
			return s, err
		}

		iso.iso.Inverse(&t, &s)
		return s, nil
	}
}

var allocCache sync.Map

func alloc[A any](ctor ...func() A) A {
	if len(ctor) > 0 && ctor[0] != nil {
		return ctor[0]()
	}

	var zero A
	t := reflect.TypeOf(zero)

	if fn, ok := allocCache.Load(t); ok {
		return fn.(func() A)()
	}

	var fn func() A

	if t.Kind() == reflect.Ptr {
		elem := t.Elem()
		fn = func() A {
			return reflect.New(elem).Interface().(A)
		}
	} else {
		fn = func() A {
			return reflect.New(t).Elem().Interface().(A)
		}
	}

	allocCache.Store(t, fn)
	return fn()
}
