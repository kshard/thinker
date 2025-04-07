//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command

import (
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// TODO: multiline return - codeblock

// A unique name for return command
const RETURN = "return"

// Creates new return command, instructing LLM return results
func Return() thinker.Cmd {
	return thinker.Cmd{
		Cmd:    RETURN,
		Short:  "indicate that workflow is completed, the agent return expected results",
		Syntax: "return <codeblock>value to return</codeblock>",
		Run: func(t chatter.Reply) (float64, thinker.CmdOut, error) {
			code, err := CodeBlock(RETURN, t.Text)
			if err != nil {
				return 0.00, thinker.CmdOut{Cmd: RETURN}, err
			}
			return 1.0, thinker.CmdOut{Cmd: RETURN, Output: code}, nil
		},
	}
}
