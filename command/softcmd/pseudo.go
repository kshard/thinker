//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package softcmd

import (
	"github.com/kshard/chatter"
)

// A unique name for return command
const RETURN = "return"

// Creates new return command, instructing LLM return results
func Return() Cmd {
	return Cmd{
		Cmd:    RETURN,
		About:  "indicate that workflow is completed, the agent return expected results",
		Syntax: "return <codeblock>value to return</codeblock>",
		Run: func(t *chatter.Reply) (float64, CmdOut, error) {
			code, err := CodeBlock(RETURN, t.String())
			if err != nil {
				return 0.00, CmdOut{Cmd: RETURN}, err
			}
			return 1.0, CmdOut{Cmd: RETURN, Output: code}, nil
		},
	}
}
