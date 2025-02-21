//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command

import (
	"fmt"
	"strings"
	"sync"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// The command registry, used by the application to define available tools and
// commands for workflows. It automates the advertisement of registered commands
// and their usage rules.
type Registry struct {
	mu       sync.Mutex
	registry map[string]thinker.Cmd
}

var _ thinker.Decoder[thinker.CmdOut] = (*Registry)(nil)

// Creates new command registry.
func NewRegistry() *Registry {
	return &Registry{
		registry: make(map[string]thinker.Cmd),
	}
}

// Register new command
func (r *Registry) Register(cmd thinker.Cmd) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, has := r.registry[cmd.Cmd]; has {
		return thinker.ErrCmdConflict
	}

	if !cmd.IsValid() {
		return thinker.ErrCmdInvalid
	}

	r.registry[cmd.Cmd] = cmd
	return nil
}

// Injects requirments for LLM about available tooling
func (r *Registry) Harden(prompt *chatter.Prompt) {
	prompt.With(
		chatter.Rules(
			`Strictly adhere to the following requirements when generating a response.
			Do not deviate, ignore, or modify any aspect of them:`,

			"When you need to execute a command, output a structured command using the syntax defined by the commands registry.",
			"Implement the sequential workflow, output single command only and wait for result to decide next action.",
			"Do not assume availability of any command.",
			"Do not invent commands that are not explicitly allowed.",
		),
	)

	seq := make([]string, 0)
	for _, app := range r.registry {
		cmd := fmt.Sprintf("%s: %s, use the syntax to invoke: TOOL:%s", app.Cmd, app.Short, app.Syntax)
		seq = append(seq, cmd)
	}

	prompt.With(
		chatter.Context("Allowed commands registry:", seq...),
	)

	for _, app := range r.registry {
		if len(app.Long) != 0 {
			seq := make([]string, 0)
			if len(app.Args) != 0 {
				seq = append(seq, fmt.Sprintf("TOOL:%s takes parameters: %s", app.Cmd, strings.Join(app.Args, ",")))
			}
			seq = append(seq, app.Long)

			prompt.With(
				chatter.Context(
					fmt.Sprintf("Detailed instructions about the TOOL:%s", app.Cmd),
					seq...,
				),
			)
		}
	}
}

// Transform LLM response into the command invokation, returns the result of command.
func (r *Registry) Decode(reply chatter.Reply) (float64, thinker.CmdOut, error) {
	s := string(reply.Text)
	at := strings.Index(s, "TOOL:")
	if at > -1 {
		for name, cmd := range r.registry {
			if strings.HasPrefix(s[at+5:], name) {
				in := chatter.Reply{
					Text:            s[5+len(name):],
					UsedInputTokens: reply.UsedInputTokens,
					UsedReplyTokens: reply.UsedReplyTokens,
				}
				return cmd.Run(in)
			}
		}
	}

	err := thinker.Feedback(
		`Improve the response based on feedback:`,
		"The output does not contain valid reference to the tool.",
		"No pattern TOOL:... is found in the output.",
	)

	return 0.0, thinker.CmdOut{}, err
}
