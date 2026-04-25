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
	"reflect"

	"github.com/fogfish/golem/optics"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/command"
)

// LLMs is the registry of available language models. Callers look up a model
// by name; the second return value reports whether the name was found.
type LLMs interface {
	Model(string) (chatter.Chatter, bool)
}

const chalkboard = "io.console.chalkboard"

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

// Runtime is the shared execution environment threaded through every agent
// constructor. It bundles the file system (for prompt templates), the LLM
// registry, an optional command registry for tool use, and the progress
// reporter.
type Runtime struct {
	FileSystem fs.FS
	LLMs       LLMs
	Registry   *command.Registry
}

// NewRuntime creates a Runtime with the given file system and LLM registry.
// Progress output is silenced by default; call WithStdout to enable it.
func NewRuntime(fs fs.FS, llms LLMs) *Runtime {
	return &Runtime{
		FileSystem: fs,
		LLMs:       llms,
	}
}

// WithFileSystem returns a copy of the runtime that uses the given file system
// for prompt loading. Each agent will inherit it unless the prompt file declares
// its own file system.
func (rt *Runtime) WithFileSystem(fs fs.FS) *Runtime {
	return &Runtime{
		FileSystem: fs,
		LLMs:       rt.LLMs,
		Registry:   rt.Registry,
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
	}
}

// =============================================================================
// Kleisli State Algebra
//
// The core abstraction is the Kleisli category Kl(M) over the error monad
//
//	M A = Context → Either(Error, A)
//
// Objects are types; morphisms S ⇝ S are Kleisli endomorphisms on the shared
// blackboard S. The StateT S M monad transformer gives the full picture:
//
//	StateT S M A = S → M (A × S)
//
// Three type-level building blocks parameterise a state transition:
//
//	Lens[S, A] — the "put" half of a lens: folds output A from a bot into
//	             shared state S.
//
//	Eval[S]    — Kleisli endomorphism S ⇝ S: side-effectful transformation of S
//	             (persistence, validation, guardrails).
//
//	Eff[S,A]   — the composite effect: bundles Lens and Eval; when either field
//	             is nil the defaults are used: a type-derived lens for Lens,
//	             the identity for Eval.
//
// Arrow lifts any Bot[S, A] into the Kleisli arrow Arr[S]:
//
//	Arrow(bot, eff) : S ⇝ S
//	  s ↦ let a = bot.Prompt(s)
//	        s' = eff.Lens(s, a)
//	    in eff.Eval(s')
//
// Arr[S] itself satisfies Bot[S, S] via its Prompt method, making every
// Kleisli arrow directly composable as a bot.
//
// Seq is monoid composition in End_Kl(M)(S):
//
//	Seq(f, g, …) = f >=> g >=> …
//
// =============================================================================

// Eval is the Kleisli endomorphism type S ⇝ S: receives the current blackboard
// S and returns an updated blackboard, possibly with an error.
type Eval[S any] = func(context.Context, S) (S, error)

// Lens is the "put" half of a lens: folds a bot output A into the shared
// blackboard S, returning the updated state.
type Lens[S, A any] = func(S, A) S

// Eff is the composite effect that bridges Bot[S, A] to Arr[S]. It bundles
// a Lens (pure state setter) and an Eval (side-effectful post-processing).
// Both fields are optional: a nil Lens triggers automatic lens derivation via
// optics.ForProduct1; a nil Eval is treated as the identity.
type Eff[S, A any] struct {
	Lens Lens[S, A]
	Eval Eval[S]
}

// EffLens constructs an Eff with only a custom Lens setter; Eval defaults to
// the identity. Both type parameters are inferred from lens.
func EffLens[S, A any](lens Lens[S, A]) Eff[S, A] {
	return Eff[S, A]{Lens: lens}
}

// EffEval constructs an Eff with only a custom Eval function; Lens defaults to
// the auto-derived lens. S is inferred from eval; A must be specified:
//
//	EffEval[string](func(ctx context.Context, s State) (State, error) { … })
func EffEval[A, S any](eval Eval[S]) Eff[S, A] {
	return Eff[S, A]{Eval: eval}
}

// Arr is the Kleisli endomorphism S ⇝ S modelling a single step in a
// stateful pipeline.  It satisfies Bot[S, S] through its Prompt method.
type Arr[S any] func(context.Context, S, ...chatter.Opt) (S, error)

// Prompt makes Arr[S] satisfy Bot[S, S]; the step is invoked directly.
func (f Arr[S]) Prompt(ctx context.Context, s S, opt ...chatter.Opt) (S, error) {
	return f(ctx, s, opt...)
}

// When wraps the arrow with a run predicate. The arrow runs only when pred(s)
// returns true; otherwise the current state is returned unchanged (identity
// Kleisli arrow). This lifts an imperative "if not already done" guard into a
// composable Arr[S] that fits directly in a Seq pipeline:
//
//	Seq(
//	    Arrow(botEvidence, eff).When(func(s S) bool { return len(s.Field) == 0 }),
//	    Arrow(botTriage,   eff).When(func(s S) bool { return len(s.Urls)  == 0 }),
//	)
//
// Denotation:
//
//	arr.When(pred) : s ↦ if pred(s) then arr(s) else s
func (f Arr[S]) When(pred func(S) bool) Arr[S] {
	return func(ctx context.Context, s S, opt ...chatter.Opt) (S, error) {
		if !pred(s) {
			return s, nil
		}
		return f(ctx, s, opt...)
	}
}

// WithTask wraps the arrow with a progress report. The task name is fixed const string.
func (f Arr[S]) WithTask(name string) Arr[S] { return f.WithTaskf(func(S) string { return name }) }

// WithTaskf wraps the arrow with a progress report.
// The task name is generated by applying fn to the current state S at the time of execution.
// The task is automatically marked done when the arrow returns, even if it returns an error.
func (f Arr[S]) WithTaskf(fn func(S) string) Arr[S] {
	return func(ctx context.Context, s S, opt ...chatter.Opt) (S, error) {
		c, ok := ctx.Value(chalkboard).(Chalk)
		if !ok || c == nil {
			return f(ctx, s, opt...)
		}

		c.Task(ctx, fn(s))
		defer c.Done()
		return f(c.Sub(ctx), s, opt...)
	}
}

// Arrow lifts Bot[S, A] into the Kleisli arrow Arr[S] using the supplied Eff.
// If no Eff is given, Lens is derived automatically from the first field of S
// that has type A (via optics.ForProduct1); Eval defaults to the identity.
// If an Eff is given but either field is nil, the same defaults apply.
//
//	Arrow(bot, eff) : s ↦ eff.Eval(eff.Lens(s, bot.Prompt(s)))
func Arrow[S, A any](bot Bot[S, A], eff ...Eff[S, A]) Arr[S] {
	var lens func(S, A) S
	var eval func(context.Context, S) (S, error)

	if len(eff) > 0 {
		lens = eff[0].Lens
		eval = eff[0].Eval
	}
	if lens == nil {
		lens = mustLens[S, A]().Put
	}
	if eval == nil {
		eval = func(_ context.Context, s S) (S, error) { return s, nil }
	}

	return func(ctx context.Context, s S, opt ...chatter.Opt) (S, error) {
		a, err := bot.Prompt(ctx, s, opt...)
		if err != nil {
			return *new(S), err
		}
		return eval(ctx, lens(s, a))
	}
}

// Bind lifts Bot[S, A] into Arr[S] using only the Eval side-effect; Lens is
// auto-derived from the unique field of type A in S. This is the short form of
// Arrow(bot, Eff{Eval: eval}) for the common case where only a side-effect
// (e.g. persistence, chalk commit) is needed after the bot runs.
//
//	Bind(bot, eval) : s ↦ eval(lens.Put(s, bot.Prompt(s)))
func Bind[S, A any](bot Bot[S, A], eval Eval[S]) Arr[S] {
	lens := mustLens[S, A]().Put

	return func(ctx context.Context, s S, opt ...chatter.Opt) (S, error) {
		a, err := bot.Prompt(ctx, s, opt...)
		if err != nil {
			return *new(S), err
		}
		return eval(ctx, lens(s, a))
	}
}

// Lift injects a plain Eval[S] function into Arr[S] as a pipeline step with no
// bot call. Use it for pure computations, validation, or transformation steps
// that operate directly on the blackboard S.
//
//	Lift(f) : s ↦ f(s)
func Lift[S any](eval Eval[S]) Arr[S] {
	return func(ctx context.Context, s S, opt ...chatter.Opt) (S, error) {
		return eval(ctx, s)
	}
}

// Pure lifts a deterministic function func(context.Context, S) (A, error) into
// Bot[S, A] with no LLM call. This is the unit/return of the Bot abstraction:
// it bridges pure blackboard reads into any position that expects a Bot[S, A],
// most notably the think argument of ThinkReAct.
//
//	Pure(f) : Bot[S, A]  where  Prompt(s) = f(s)
//
// Typical use — read an existing []A field from S and pass it to ThinkReAct:
//
//	ThinkReAct(rt,
//	    Think(
//	        Pure(func(_ context.Context, s S) ([]A, error) { return s.Items, nil }),
//	        func(s S, a A) T { return T{Item: a, Ctx: s.SharedCtx} },
//	    ),
//	    react,
//	    gather,
//	)
func Pure[S, A any](f func(context.Context, S) (A, error)) Bot[S, A] {
	return &botPure[S, A]{f: f}
}

type botPure[S, A any] struct {
	f func(context.Context, S) (A, error)
}

func (b *botPure[S, A]) Prompt(ctx context.Context, s S, opt ...chatter.Opt) (A, error) {
	return b.f(ctx, s)
}

// =============================================================================
// Internal lens helpers
// =============================================================================

type lens[S, A any] struct {
	l optics.Lens[S, A]
	s bool
}

func mustLens[S, A any]() lens[S, A] {
	if reflect.TypeOf((*S)(nil)) == reflect.TypeOf((*A)(nil)) {
		return lens[S, A]{s: true}
	}

	return lens[S, A]{
		l: optics.ForProduct1[S, A](),
	}
}

func (f lens[S, A]) Put(s S, a A) S {
	if f.s {
		return s
	}

	return *f.l.Put(&s, a)
}
