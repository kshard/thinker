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

type botMap[S, A, B any] struct {
	bot Bot[S, A]
	fn  func(S, A) B
}

func (m *botMap[S, A, B]) Prompt(ctx context.Context, s S, opt ...chatter.Opt) (B, error) {
	a, err := m.bot.Prompt(ctx, s, opt...)
	if err != nil {
		return *new(B), err
	}
	return m.fn(s, a), nil
}

// Map transforms the output type of a Bot[S, A] into Bot[S, B].
// f receives the current blackboard S and the bot's output A, and returns B.
//
// The denotation is:
//
//	Map(bot, f) : s ↦ f(s, bot.Prompt(s))
func Map[S, A, B any](b Bot[S, A], f func(S, A) B) Bot[S, B] {
	return &botMap[S, A, B]{bot: b, fn: f}
}
