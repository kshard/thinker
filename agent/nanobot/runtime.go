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
	"io/fs"

	"github.com/fogfish/golem/optics"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/command"
)

// LLMs is the registry of available language models. Callers look up a model
// by name; the second return value reports whether the name was found.
type LLMs interface {
	Model(string) (chatter.Chatter, bool)
}

// Chalk is a structured progress-reporting sink. Implementations can write to
// a terminal, a log file, or any other destination. The no-op default is used
// when no output is configured.
type Chalk interface {
	Sub(context.Context) context.Context
	Task(context.Context, string, ...any)
	Done(...string)
	Fail(error)
	Printf(format string, args ...any)
}

// Bot is the core building block of the nanobot package. Any agent that
// accepts an input of type S and returns a result of type A satisfies this
// interface, making different agent implementations composable.
type Bot[S, A any] interface {
	Prompt(ctx context.Context, input S, opt ...chatter.Opt) (A, error)
}

type devnull struct{}

func (d devnull) Sub(ctx context.Context) context.Context              { return ctx }
func (d devnull) Task(ctx context.Context, format string, args ...any) {}
func (d devnull) Done(...string)                                       {}
func (d devnull) Fail(error)                                           {}
func (d devnull) Printf(format string, args ...any)                    {}

// Runtime is the shared execution environment threaded through every agent
// constructor. It bundles the file system (for prompt templates), the LLM
// registry, an optional command registry for tool use, and the progress
// reporter.
type Runtime struct {
	FileSystem fs.FS
	LLMs       LLMs
	Registry   *command.Registry
	Chalk      Chalk
}

// NewRuntime creates a Runtime with the given file system and LLM registry.
// Progress output is silenced by default; call WithStdout to enable it.
func NewRuntime(fs fs.FS, llms LLMs) *Runtime {
	var chalk Chalk = devnull{}

	return &Runtime{
		FileSystem: fs,
		LLMs:       llms,
		Chalk:      chalk,
	}
}

// WithRegistry returns a copy of the runtime that uses the given command
// registry for tool-call dispatch. Each ReAct agent will inherit it unless
// the prompt file declares its own servers.
func (rt *Runtime) WithRegistry(r *command.Registry) *Runtime {
	return &Runtime{
		FileSystem: rt.FileSystem,
		LLMs:       rt.LLMs,
		Registry:   r,
		Chalk:      rt.Chalk,
	}
}

// WithStdout returns a copy of the runtime that reports progress to c.
func (rt *Runtime) WithStdout(c Chalk) *Runtime {
	return &Runtime{
		FileSystem: rt.FileSystem,
		LLMs:       rt.LLMs,
		Registry:   rt.Registry,
		Chalk:      c,
	}
}

// Workspace holds the two lifecycle hooks—apply and effect—that are shared
// across composite bot patterns (Seq, ThinkReAct, Reflect). Keeping them in
// one place avoids repeating the same configuration on every bot.
type Workspace[S, A any] struct {
	apply  func(S, A) S
	effect func(context.Context, S) (S, error)
}

// NewWorkspace creates a Workspace whose apply function is a type-derived
// lens (the default when S contains exactly one field of type A) and whose
// effect is the identity.
func NewWorkspace[S, A any]() *Workspace[S, A] {
	return &Workspace[S, A]{
		apply:  mustApply[S, A]().Put,
		effect: func(ctx context.Context, s S) (S, error) { return s, nil },
	}
}

// WithApply overrides the default lens-based merge with a custom function
// that folds the agent output A back into the shared state S.
func (w *Workspace[S, A]) WithApply(apply func(S, A) S) *Workspace[S, A] {
	w.apply = apply
	return w
}

// WithEffect registers a side-effect function that runs after each apply
// step, for example to persist state, emit events, or enforce guardrails.
func (w *Workspace[S, A]) WithEffect(effect func(context.Context, S) (S, error)) *Workspace[S, A] {
	w.effect = effect
	return w
}

type apply[S, A any] struct {
	lens optics.Lens[S, A]
}

func mustApply[S, A any]() apply[S, A] {
	return apply[S, A]{
		lens: optics.ForProduct1[S, A](),
	}
}

func (f apply[S, A]) Put(s S, a A) S {
	return *f.lens.Put(&s, a)
}
