//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command

import (
	"strings"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// A unique name for return command
const RETURN = "return"

// Creates new return command, instructing LLM return results
func Return() thinker.Cmd {
	return thinker.Cmd{
		Cmd:    RETURN,
		Short:  "indicate that workflow is completed, the agent return expected results",
		Syntax: "return <value>",
		Run: func(t chatter.Reply) (float64, thinker.CmdOut, error) {
			s := strings.TrimSpace(t.Text)
			return 1.0, thinker.CmdOut{Cmd: RETURN, Output: s}, nil
		},
	}
}
