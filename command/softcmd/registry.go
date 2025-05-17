//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package softcmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// A Command defines an external tool or utility available to the agent for task-solving.
// To ensure usability, each command must include a usage definition and description.
type Cmd struct {
	// [Required] A unique name for the command, used as a reference by LLMs (e.g., "bash").
	Cmd string

	// [Required] A description of the command and its purpose.
	// Used to define the command registry for LLMs.
	About string

	// [Required] Concise instructions on the command's syntax.
	// For example: "bash <command>".
	Syntax string

	// The actual command execution function, which can be defined statically or
	// dynamically upon registration.
	Run func(*chatter.Reply) (float64, CmdOut, error)
}

type Arg struct {
	Name  string `json:"-"`
	Type  string `json:"type"`
	About string `json:"description"`
}

func (cmd Cmd) IsValid() bool {
	return len(cmd.Cmd) != 0 && len(cmd.About) != 0 && cmd.Run != nil
}

// Container for command results.
type CmdOut struct {
	// A unique name of the command, used to getnerate output.
	Cmd string

	// Output of the command.
	Output string
}

// The command registry, used by the application to define available tools and
// commands for workflows. It automates the advertisement of registered commands
// and their usage rules.
type Registry struct {
	mu       sync.Mutex
	registry map[string]Cmd
}

var _ thinker.Decoder[CmdOut] = (*Registry)(nil)

// Creates new command registry.
func NewRegistry() *Registry {
	return &Registry{
		registry: make(map[string]Cmd),
	}
}

// Register new command
func (r *Registry) Register(cmd Cmd) error {
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
	prompt.WithRules(
		`Strictly adhere to the following requirements when generating a response.
			Do not deviate, ignore, or modify any aspect of them:`,

		"When you need to execute a command, output a structured command using the syntax defined by the commands registry.",
		"Implement the sequential workflow, output single command only and wait for result to decide next action.",
		"Do not assume availability of any command.",
		"Do not invent commands that are not explicitly allowed.",
	)

	seq := make([]string, 0)
	for _, app := range r.registry {
		cmd := fmt.Sprintf("%s: %s, use the syntax to invoke: TOOL:%s", app.Cmd, app.About, app.Syntax)
		seq = append(seq, cmd)
	}

	prompt.WithContext("Allowed commands registry:", seq...)
}

// Transform LLM response into the command invokation, returns the result of command.
func (r *Registry) Decode(reply *chatter.Reply) (float64, CmdOut, error) {
	s := reply.String()
	at := strings.Index(s, "TOOL:")
	if at > -1 {
		for name, cmd := range r.registry {
			if strings.HasPrefix(s[at+5:], name) {
				in := &chatter.Reply{
					Stage: chatter.LLM_RETURN,
					Content: []chatter.Content{
						chatter.Text(s[5+len(name):]),
					},
					Usage: reply.Usage,
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

	return 0.0, CmdOut{}, err
}
