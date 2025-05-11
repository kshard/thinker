//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package thinker

import "github.com/kshard/chatter"

// A Command defines an external tool or utility available to the agent for task-solving.
// To ensure usability, each command must include a usage definition and description.
type Cmd struct {
	// [Required] A unique name for the command, used as a reference by LLMs (e.g., "bash").
	Cmd string

	// [Required] A concise, one-line description of the command and its purpose.
	// Used to define the command registry for LLMs.
	Short string

	// [Required] Concise instructions on the command's syntax.
	// For example: "bash <command>".
	Syntax string

	// [Optional] A detailed, multi-line description to educate the LLM on command usage.
	// Provides contextual information on how and when to use the command.
	Long string

	// [Optional] Specifies arguments, types, and additional context to guide
	// the LLM on command syntax.
	Args []string

	// The actual command execution function, which can be defined statically or
	// dynamically upon registration.
	Run func(*chatter.Reply) (float64, CmdOut, error)
}

func (cmd Cmd) IsValid() bool {
	return len(cmd.Cmd) != 0 && len(cmd.Short) != 0 && len(cmd.Syntax) != 0 && cmd.Run != nil
}

// Container for command results.
type CmdOut struct {
	// A unique name of the command, used to getnerate output.
	Cmd string

	// Output of the command.
	Output string
}
