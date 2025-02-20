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
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

// A unique name for bash (the shell)
const GOLANG = "go"

// Create new Golang command, defining goroot
func Golang(gopath string) thinker.Cmd {
	return thinker.Cmd{
		Cmd:    GOLANG,
		Short:  "Use golang to execute scripts (package main) that help you complete your task. Enclose the golang code in <codeblock> tags.",
		Syntax: `go  <codeblock>source code</codeblock>`,
		Run:    golang(gopath),
	}
}

func goSourceCode(gopath string) string {
	return filepath.Join(gopath, "src", "github.com", "kshard", "jobs")
}

func goSetup(gopath string) error {
	gojob := goSourceCode(gopath)
	if err := os.MkdirAll(gojob, 0755); err != nil {
		return err
	}

	gomod := filepath.Join(gojob, "go.mod")

	if _, err := os.Stat(gomod); err != nil {
		setup := exec.Command("go", "mod", "init")
		setup.Dir = gojob
		setup.Env = []string{
			"GOPATH=" + gopath,
		}

		_, err := setup.Output()
		if err != nil {
			return err
		}
	}

	return nil
}

func goDeps(gopath string) error {
	gojob := goSourceCode(gopath)

	deps := exec.Command("go", "mod", "tidy")
	deps.Dir = gojob
	deps.Env = []string{
		"GOPATH=" + gopath,
	}

	_, err := deps.Output()
	if err != nil {
		return err
	}

	return nil
}

func gofile(gopath, code string) (string, error) {
	gojob := goSourceCode(gopath)
	dir, err := os.MkdirTemp(gojob, "job*")
	if err != nil {
		return "", err
	}

	fd, err := os.Create(filepath.Join(dir, "main.go"))
	if err != nil {
		return "", err
	}
	defer fd.Close()

	_, err = fd.WriteString(code)
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Base(dir), "main.go"), nil
}

func golang(gopath string) func(chatter.Reply) (float64, thinker.CmdOut, error) {
	return func(command chatter.Reply) (float64, thinker.CmdOut, error) {
		if err := goSetup(gopath); err != nil {
			return 0.0, thinker.CmdOut{}, err
		}

		code, err := CodeBlock(GOLANG, command.Text)
		if err != nil {
			return 0.00, thinker.CmdOut{Cmd: GOLANG}, err
		}

		file, err := gofile(gopath, code)
		if err != nil {
			return 0.00, thinker.CmdOut{Cmd: GOLANG}, err
		}

		if err := goDeps(gopath); err != nil {
			return 0.0, thinker.CmdOut{}, err
		}

		cmd := exec.Command("go", "run", file)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Dir = goSourceCode(gopath)
		cmd.Env = []string{
			"GOPATH=" + gopath,
			"GOCACHE=" + filepath.Join(gopath, "cache"),
		}
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			err = thinker.Feedback(
				fmt.Sprintf("The TOOL:%s has failed, improve the response based on feedback:", GOLANG),

				"Execution of golang program is failed with the error: "+err.Error(),
				"The error output is "+stderr.String(),
			)
			return 0.05, thinker.CmdOut{Cmd: GOLANG}, err
		}

		return 1.0, thinker.CmdOut{Cmd: GOLANG, Output: stdout.String()}, nil
	}
}
