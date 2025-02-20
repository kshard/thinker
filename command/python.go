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
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// A unique name for bash (the shell)
const PYTHON = "python"

// Create new bash command, defining the os variant and working dir
func Python(dir string) thinker.Cmd {
	return thinker.Cmd{
		Cmd:    PYTHON,
		Short:  "Use Python REPL to execute scripts that help you complete your task (format python code with \\t, \\n). Declare dependencies to python modules.",
		Syntax: `python -c """source code"""`,
		Run:    python(dir),
	}
}

func pySetup(dir string) error {
	pyenv := filepath.Join(dir, ".venv")
	if _, err := os.Stat(pyenv); err != nil {
		slog.Info("Setup python")
		setup := exec.Command("python3", "-m", "venv", ".venv")
		setup.Dir = dir

		_, err := setup.Output()
		if err != nil {
			return err
		}

		slog.Info("Setup pigar")
		pipreqs := exec.Command(filepath.Join(pyenv, "bin/python"), "-m", "pip", "install", "pigar")
		pipreqs.Dir = dir

		_, err = pipreqs.Output()
		if err != nil {
			return err
		}
	}

	return nil
}

func pyDeps(dir, scode string) error {
	pyenv := filepath.Join(dir, ".venv")
	pysrc := filepath.Join(dir, "main.py")

	slog.Info("Write source code")
	if err := os.WriteFile(pysrc, []byte(scode), 0666); err != nil {
		return err
	}

	slog.Info("Generate deps")
	deps := exec.Command(filepath.Join(pyenv, "bin/pigar"), "generate", "--question-answer", "yes", "--auto-select")
	deps.Dir = dir

	_, err := deps.Output()
	if err != nil {
		return err
	}

	slog.Info("Install deps")
	pip := exec.Command(filepath.Join(pyenv, "bin/python"), "-m", "pip", "install", "-r", "requirements.txt")
	pip.Dir = dir

	_, err = pip.Output()
	if err != nil {
		return err
	}

	return nil
}

func python(dir string) func(chatter.Reply) (float64, thinker.CmdOut, error) {
	return func(command chatter.Reply) (float64, thinker.CmdOut, error) {
		if err := pySetup(dir); err != nil {
			return 0.0, thinker.CmdOut{}, err
		}

		scode := strings.TrimSpace(command.Text)
		scode = strings.TrimPrefix(scode, `-c """`)
		scode = strings.TrimSuffix(scode, `"""`)

		if err := pyDeps(dir, scode); err != nil {
			return 0.0, thinker.CmdOut{}, err
		}

		pyenv := filepath.Join(dir, ".venv")
		cmd := exec.Command(filepath.Join(pyenv, "bin/python"), "-c", scode)
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
			return 0.05, thinker.CmdOut{Cmd: PYTHON}, err
		}

		return 1.0, thinker.CmdOut{Cmd: PYTHON, Output: stdout.String()}, nil
	}
}
