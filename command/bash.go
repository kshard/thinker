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
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/kshard/thinker"
)

// A unique name for bash (the shell)
const BASH = "bash"

// Create new bash command, defining the os variant and working dir
func Bash(os, dir string) thinker.Cmd {
	return thinker.Cmd{
		Cmd:   BASH,
		About: fmt.Sprintf("Executes shell command on %s operating system.", os),
		Args: []thinker.Arg{
			{
				Name:  "script",
				Type:  "string",
				About: `script is single or sequence of bash commands passed to bash -c "script".`,
			},
		},
		Run: bash(os, dir),
	}
}

type script struct {
	Script string `json:"script"`
}

func bash(os string, dir string) func(json.RawMessage) ([]byte, error) {
	return func(command json.RawMessage) ([]byte, error) {
		var code script
		if err := json.Unmarshal(command, &code); err != nil {
			err := thinker.Feedback(
				"The input does not contain valid JSON object",
				"JSON parsing has failed with an error "+err.Error(),
			)
			return nil, err
		}

		cmd := exec.Command("bash", "-c", code.Script)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Dir = dir
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			err = thinker.Feedback(
				fmt.Sprintf("The tool %s has failed, improve the response based on feedback:", BASH),

				fmt.Sprintf("Strictly adhere shell command syntaxt to %s.", os),
				"Execution of the tool is failed with the error: "+err.Error(),
				"The error output is "+stderr.String(),
			)

			return nil, err
		}

		return stdout.Bytes(), nil
	}
}
