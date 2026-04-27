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
// TestForEach
// =============================================================================

func TestForEach(t *testing.T) {
	t.Run("ForEachSuccess", func(t *testing.T) {
		// Define a state type with a slice field
		type State struct {
			Items []int
		}

		// Define an Arr[int] that increments each int
		arrInt := func(ctx context.Context, value int, opt ...chatter.Opt) (int, error) {
			return value + 1, nil
		}

		// Create ForEach Arr[State]
		arrState := nanobot.ForEach[State, int](arrInt)

		// Test with initial state
		state := State{Items: []int{1, 2, 3}}
		result, err := arrState(context.Background(), state)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(len(result.Items), 3),
			it.Equal(result.Items[0], 2), // 1 + 1
			it.Equal(result.Items[1], 3), // 2 + 1
			it.Equal(result.Items[2], 4), // 3 + 1
		)
	})

	t.Run("ForEachEmptySlice", func(t *testing.T) {
		type State struct {
			Items []string
		}

		arrString := func(ctx context.Context, value string, opt ...chatter.Opt) (string, error) {
			return value + "_processed", nil
		}

		arrState := nanobot.ForEach[State, string](arrString)

		state := State{Items: []string{}}
		result, err := arrState(context.Background(), state)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(len(result.Items), 0),
		)
	})

	t.Run("ForEachErrorPropagation", func(t *testing.T) {
		type State struct {
			Items []int
		}

		testErr := errors.New("processing error")
		arrInt := func(ctx context.Context, value int, opt ...chatter.Opt) (int, error) {
			if value == 2 {
				return 0, testErr
			}
			return value * 2, nil
		}

		arrState := nanobot.ForEach[State, int](arrInt)

		state := State{Items: []int{1, 2, 3}}
		result, err := arrState(context.Background(), state)

		it.Then(t).Should(
			it.Equal(err, testErr),
			it.Equal(result.Items[0], 2), // 1 * 2 (processed before error)
			it.Equal(result.Items[1], 2), // unchanged
			it.Equal(result.Items[2], 3), // unchanged
		)
	})

	t.Run("ForEachWithStringSlice", func(t *testing.T) {
		type State struct {
			Messages []string
		}

		arrString := func(ctx context.Context, value string, opt ...chatter.Opt) (string, error) {
			return "processed: " + value, nil
		}

		arrState := nanobot.ForEach[State, string](arrString)

		state := State{Messages: []string{"hello", "world"}}
		result, err := arrState(context.Background(), state)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(len(result.Messages), 2),
			it.Equal(result.Messages[0], "processed: hello"),
			it.Equal(result.Messages[1], "processed: world"),
		)
	})

	t.Run("ForEachWithStructSlice", func(t *testing.T) {
		type Item struct {
			Value int
			Name  string
		}
		type State struct {
			Items []Item
		}

		arrItem := func(ctx context.Context, item Item, opt ...chatter.Opt) (Item, error) {
			item.Value += 100
			item.Name = "updated_" + item.Name
			return item, nil
		}

		arrState := nanobot.ForEach[State, Item](arrItem)

		state := State{Items: []Item{
			{Value: 10, Name: "first"},
			{Value: 20, Name: "second"},
		}}
		result, err := arrState(context.Background(), state)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(len(result.Items), 2),
			it.Equal(result.Items[0].Value, 110),
			it.Equal(result.Items[0].Name, "updated_first"),
			it.Equal(result.Items[1].Value, 120),
			it.Equal(result.Items[1].Name, "updated_second"),
		)
	})
}
