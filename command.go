//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package thinker

import (
	"encoding/json"
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

	// [Optional] Specifies arguments, types, and additional context to guide
	// the LLM on command syntax.
	Args []Arg

	// The actual command execution function, which can be defined statically or
	// dynamically upon registration.
	Run func(json.RawMessage) ([]byte, error)
}

type Arg struct {
	Name  string `json:"-"`
	Type  string `json:"type"`
	About string `json:"description"`
}

func (cmd Cmd) IsValid() bool {
	return len(cmd.Cmd) != 0 && len(cmd.About) != 0 && cmd.Run != nil
}
