//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package softcmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// A unique name for bash (the shell)
const PYTHON = "python"

// Create new python command, defining working dir
func Python(dir string) Cmd {
	return Cmd{
		Cmd:    PYTHON,
		About:  "Use python to execute scripts that help you complete your task. Enclose the python code in <codeblock> tags.",
		Syntax: `python <codeblock>source code</codeblock>`,
		Run:    python(dir),
	}
}

func pySetup(dir string) error {
	pyenv := filepath.Join(dir, ".venv")
	if _, err := os.Stat(pyenv); err != nil {
		setup := exec.Command("python3", "-m", "venv", ".venv")
		setup.Dir = dir

		_, err := setup.Output()
		if err != nil {
			return err
		}

		pipreqs := exec.Command(filepath.Join(pyenv, "bin/python"), "-m", "pip", "install", "pigar")
		pipreqs.Dir = dir

		_, err = pipreqs.Output()
		if err != nil {
			return err
		}
	}

	return nil
}

func pyDeps(dir string) error {
	pyenv := filepath.Join(dir, ".venv")

	deps := exec.Command(filepath.Join(pyenv, "bin/pigar"), "generate", "--question-answer", "yes", "--auto-select")
	deps.Dir = dir

	_, err := deps.Output()
	if err != nil {
		return err
	}

	pip := exec.Command(filepath.Join(pyenv, "bin/python"), "-m", "pip", "install", "-r", "requirements.txt")
	pip.Dir = dir

	_, err = pip.Output()
	if err != nil {
		return err
	}

	return nil
}

func pyfile(dir, code string) (string, error) {
	fd, err := os.CreateTemp(dir, "job-*.py")
	if err != nil {
		return "", err
	}
	defer fd.Close()

	_, err = fd.WriteString(code)
	if err != nil {
		return "", err
	}

	return fd.Name(), nil
}

func python(dir string) func(*chatter.Reply) (float64, CmdOut, error) {
	return func(command *chatter.Reply) (float64, CmdOut, error) {
		if err := pySetup(dir); err != nil {
			return 0.0, CmdOut{}, err
		}

		code, err := CodeBlock(PYTHON, command.String())
		if err != nil {
			return 0.00, CmdOut{Cmd: PYTHON}, err
		}

		file, err := pyfile(dir, code)
		if err != nil {
			return 0.00, CmdOut{Cmd: PYTHON}, err
		}

		if err := pyDeps(dir); err != nil {
			return 0.0, CmdOut{}, err
		}

		pyenv := filepath.Join(dir, ".venv")
		cmd := exec.Command(filepath.Join(pyenv, "bin/python"), file)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Dir = dir
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			err = thinker.Feedback(
				fmt.Sprintf("The TOOL:%s has failed, improve the response based on feedback:", PYTHON),

				`Strictly adhere python code formatting, use \t, \n where is needed`,
				"Execution of python script is failed with the error: "+err.Error(),
				"The error output is "+stderr.String(),
			)
			return 0.05, CmdOut{Cmd: PYTHON}, err
		}

		return 1.0, CmdOut{Cmd: PYTHON, Output: stdout.String()}, nil
	}
}
