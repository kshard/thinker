//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package agent

import (
	"encoding"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/kshard/chatter"
	"github.com/kshard/chatter/aio"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/kshard/thinker/prompt"
	"github.com/kshard/thinker/prompt/jsonify"
)

type NanoBot[A, B any] struct {
	*Manifold[A, B]
	prompt *prompt.Prompt
	t      *template.Template
}

func MakeNanoBot[A, B any](llm thinker.LLM, fs fs.FS, file string) *NanoBot[A, B] {
	bot, err := NewNanoBot[A, B](llm, fs, file)
	if err != nil {
		panic(err)
	}
	return bot
}

func NewNanoBot[A, B any](llm thinker.LLM, fs fs.FS, file string) (*NanoBot[A, B], error) {
	prompt, err := prompt.ParseFile(fs, file)
	if err != nil {
		return nil, err
	}

	t, err := template.New("").Parse(prompt.Prompt)
	if err != nil {
		return nil, err
	}

	ml := llm.Base
	switch prompt.RunsOn {
	case "micro":
		ml = llm.Micro
	case "small":
		ml = llm.Small
	case "medium":
		ml = llm.Medium
	case "large":
		ml = llm.Large
	case "xlarge":
		ml = llm.XLarge
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
		ml = aio.NewJsonLogger(os.Stderr, ml)
	}

	bot := &NanoBot[A, B]{prompt: prompt, t: t}
	bot.Manifold = NewManifold(
		ml,
		codec.FromEncoder(bot.encode),
		codec.FromDecoder(bot.decode),
		registry,
	)

	return bot, nil
}

func (bot *NanoBot[A, B]) encode(in A) (chatter.Message, error) {
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

	return &prompt, nil
}

func (bot *NanoBot[A, B]) decode(reply *chatter.Reply) (float64, B, error) {
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
			return 0.0, *out, fmt.Errorf("nanobot unable to handle type: %T as string", out)
		}
	}

	var out B
	if err := jsonify.Strings.Decode(reply, bot.prompt.Schema.Reply, &out); err != nil {
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
