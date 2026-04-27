//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package nanobot_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/agent/nanobot"
)

// =============================================================================
// TestHoist
// =============================================================================

func TestHoist(t *testing.T) {
	t.Run("BiMapHoistSuccess", func(t *testing.T) {
		// Define source and target types with matching field types
		type Source struct {
			FieldA int
			FieldB string
		}
		type Target struct {
			FieldA int
			FieldB string
		}

		// Create a hoister using BiMap
		hoister := nanobot.BiMap[Source, int, Target, string]()

		// Define an Arr[Target] that modifies the target
		arrTarget := func(ctx context.Context, target Target, opt ...chatter.Opt) (Target, error) {
			target.FieldA += 10
			target.FieldB += "_modified"
			return target, nil
		}

		// Hoist the Arr[Target] to Arr[Source]
		arrSource := hoister.Hoist(arrTarget)

		// Test the hoisted function
		source := Source{FieldA: 5, FieldB: "hello"}
		result, err := arrSource(context.Background(), source)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.FieldA, 5), // A field unchanged
			it.Equal(result.FieldB, "_modified"),
		)
	})

	t.Run("BiMapHoistErrorPropagation", func(t *testing.T) {
		type Source struct {
			FieldA int
			FieldB string
		}
		type Target struct {
			FieldA int
			FieldB string
		}

		hoister := nanobot.BiMap[Source, int, Target, string]()

		// Arr[Target] that returns an error
		testErr := errors.New("test error")
		arrTarget := func(ctx context.Context, target Target, opt ...chatter.Opt) (Target, error) {
			return target, testErr
		}

		arrSource := hoister.Hoist(arrTarget)

		source := Source{FieldA: 1, FieldB: "test"}
		result, err := arrSource(context.Background(), source)

		it.Then(t).Should(
			it.Equal(err, testErr),
			it.Equal(result.FieldA, 1),      // unchanged
			it.Equal(result.FieldB, "test"), // unchanged
		)
	})

	t.Run("BiMapHoistWithDifferentTypes", func(t *testing.T) {
		// Test with different field types but same structure
		type Source struct {
			A float64
			B bool
		}
		type Target struct {
			A float64
			B bool
		}

		hoister := nanobot.BiMap[Source, float64, Target, bool]()

		arrTarget := func(ctx context.Context, target Target, opt ...chatter.Opt) (Target, error) {
			target.A *= 2
			target.B = !target.B
			return target, nil
		}

		arrSource := hoister.Hoist(arrTarget)

		source := Source{A: 3.5, B: true}
		result, err := arrSource(context.Background(), source)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(result.A, 3.5),  // A field unchanged
			it.Equal(result.B, true), // !false
		)
	})
}
