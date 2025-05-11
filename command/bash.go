//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// A unique name for bash (the shell)
const BASH = "bash"

// Create new bash command, defining the os variant and working dir
func Bash(os, dir string) thinker.Cmd {
	return thinker.Cmd{
		Cmd:    BASH,
		Short:  fmt.Sprintf("executes shell command, strictly adhere shell command syntaxt to %s. Enclose the bash commands in <codeblock> tags.", os),
		Syntax: "bash <codeblock>source code</codeblock>",
		Run:    bash(os, dir),
	}
}

func bash(os string, dir string) func(*chatter.Reply) (float64, thinker.CmdOut, error) {
	return func(command *chatter.Reply) (float64, thinker.CmdOut, error) {
		code, err := CodeBlock(BASH, command.String())
		if err != nil {
			return 0.00, thinker.CmdOut{Cmd: BASH}, err
		}

		cmd := exec.Command("bash", "-c", code)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Dir = dir
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			err = thinker.Feedback(
				fmt.Sprintf("The TOOL:%s has failed, improve the response based on feedback:", BASH),

				fmt.Sprintf("Strictly adhere shell command syntaxt to %s.", os),
				"Execution of the tool is failed with the error: "+err.Error(),
				"The error output is "+stderr.String(),
			)
			return 0.05, thinker.CmdOut{Cmd: BASH}, err
		}

		return 1.0, thinker.CmdOut{Cmd: BASH, Output: stdout.String()}, nil
	}
}
