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
	"encoding"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/template"

	"github.com/kshard/chatter"
	"github.com/kshard/chatter/aio"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/prompt"
	"github.com/kshard/thinker/prompt/jsonify"
)

// BotReAct is a prompt-file-driven agent that implements the
// Reason-and-Act pattern via a Manifold loop. It loads a prompt template
// from the file system, selects the appropriate LLM from the runtime, wires
// up any MCP tool servers declared in the prompt file, and exposes a single
// Prompt method that drives the full BotReAct cycle.
type BotReAct[A, B any] struct {
	manifold *agent.Manifold[A, B]
	attempt  int
	external bool
	memory   thinker.Memory
	registry *command.SeqRegistry
	prompt   *prompt.Prompt
	t        *template.Template
	taskf    func(A) string
}

// ReAct is like NewReAct but panics on error.
func ReAct[A, B any](rt *Runtime, file string) *BotReAct[A, B] {
	bot, err := NewReAct[A, B](rt, file)
	if err != nil {
		panic(err)
	}
	return bot
}

// NewReAct creates a ReAct agent from the prompt file at the given path
// within the runtime's file system. The prompt file's front-matter controls
// the model name, output format, retry budget, and any tool servers to
// connect.
func NewReAct[A, B any](rt *Runtime, file string) (*BotReAct[A, B], error) {
	prompt, err := prompt.ParseFile(rt.FileSystem, file)
	if err != nil {
		return nil, err
	}

	t, err := template.New("").Parse(prompt.Prompt)
	if err != nil {
		return nil, err
	}

	const base = "base"
	runner, ok := rt.LLMs.Model(prompt.RunsOn)
	if !ok && prompt.RunsOn != base {
		runner, _ = rt.LLMs.Model(base)
	}

	if runner == nil {
		return nil, fmt.Errorf("no model found for prompt: %s", prompt.RunsOn)
	}

	registry := command.NewRegistry()
	for _, server := range prompt.Servers {
		switch {
		case len(server.Url) > 0:
			err := registry.ConnectUrl(server.Name, server.Url)
			if err != nil {
				return nil, err
			}

		case len(server.Command) > 0:
			err := registry.ConnectCmd(server.Name, server.Command)
			if err != nil {
				return nil, err
			}
		}
	}

	if prompt.Debug {
		runner = aio.NewJsonLogger(os.Stderr, runner)
	}

	bot := &BotReAct[A, B]{prompt: prompt, t: t}

	bot.registry = command.NewSeqRegistry()
	bot.registry.Bind(registry)
	bot.registry.Bind(rt.Registry)

	bot.memory = memory.NewStream(-1, "")

	bot.manifold = agent.NewManifold(
		runner,
		codec.FromEncoder(bot.encode),
		codec.FromDecoder(bot.decode),
		bot.registry,
	).WithMemory(bot.memory)

	return bot, nil
}

func (bot *BotReAct[A, B]) WithMemory(memory thinker.Memory) *BotReAct[A, B] {
	bot.memory, bot.external = memory, true
	bot.manifold = bot.manifold.WithMemory(memory)
	return bot
}

func (bot *BotReAct[A, B]) WithRegistry(r *command.Registry) *BotReAct[A, B] {
	bot.registry.Bind(r)
	return bot
}

func (bot *BotReAct[A, B]) WithTask(name string) *BotReAct[A, B] {
	return bot.WithTaskf(func(A) string { return name })
}

func (bot *BotReAct[A, B]) WithTaskf(fn func(A) string) *BotReAct[A, B] {
	bot.taskf = fn
	return bot
}

// Prompt encodes the input using the prompt template, runs the Manifold
// ReAct loop until the model returns a final answer, and decodes the result
// into B. Progress is reported via the Chalk sink when the prompt file
// declares a name.
func (bot *BotReAct[A, B]) Prompt(ctx context.Context, input A, opt ...chatter.Opt) (B, error) {
	if !bot.external {
		bot.memory.Reset()
	}

	chalk, ok := ctx.Value(chalkboard).(Chalk)
	if !ok || chalk == nil || bot.taskf == nil {
		return bot.manifold.Prompt(ctx, input, opt...)
	}

	chalk.Task(ctx, bot.taskf(input))
	val, err := bot.manifold.Prompt(ctx, input, opt...)
	if err != nil {
		chalk.Fail(err)
		return val, err
	}
	chalk.Done()

	return val, nil
}

func (bot *BotReAct[A, B]) encode(in A) (chatter.Message, error) {
	// see https://github.com/google/jsonschema-go/issues/23 for details
	// if bot.prompt.Schema.Input != nil {
	// 	if err := bot.validateSchema(in, bot.prompt.Schema.Input); err != nil {
	// 		return nil, fmt.Errorf("input validation failed for agent: %w", err)
	// 	}
	// }

	var sb strings.Builder
	err := bot.t.Execute(&sb, in)
	if err != nil {
		return nil, err
	}

	var prompt chatter.Prompt
	prompt.WithBlob("", sb.String())

	if bot.prompt.Schema.Format == "json" {
		jsonify.Strings.Harden(&prompt, bot.prompt.Schema.Reply)
	}

	bot.attempt = 0
	return &prompt, nil
}

func (bot *BotReAct[A, B]) decode(reply *chatter.Reply) (float64, B, error) {
	if bot.prompt.Schema.Format == "text" {
		out := new(B)

		switch v := any(out).(type) {
		case *string:
			return 1.0, any(reply.String()).(B), nil
		case encoding.TextUnmarshaler:
			if err := v.UnmarshalText([]byte(reply.String())); err != nil {
				return 0.0, *out, err
			}
			return 1.0, *out, nil
		default:
			rv := reflect.ValueOf(out)
			if rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.String {
				rv.Elem().SetString(reply.String())
				return 1.0, *out, nil
			}
			return 0.0, *out, fmt.Errorf("nanobot unable to handle type: %T as string", out)
		}
	}

	var out B
	if err := jsonify.Strings.Decode(reply, bot.prompt.Schema.Reply, &out); err != nil {
		bot.attempt++
		if bot.attempt >= bot.prompt.Retry {
			return 0.0, out, fmt.Errorf("unable to reply with JSON after %d attempts: %w", bot.attempt, err)
		}
		return 0.0, out, err
	}

	return 1.0, out, nil
}

// see https://github.com/google/jsonschema-go/issues/23 for details
// func (bot *NanoBot[A, B]) validateSchema(obj any, schema *jsonschema.Schema) error {
// 	resolved, err := schema.Resolve(nil)
// 	if err != nil {
// 		return fmt.Errorf("failed to resolve schema: %w", err)
// 	}
//
// 	if err := resolved.Validate(obj); err != nil {
// 		return fmt.Errorf("invalid object: %w", err)
// 	}
//
// 	return nil
// }
