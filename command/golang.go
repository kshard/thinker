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
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kshard/thinker"
)

// A unique name for bash (the shell)
const GOLANG = "go"

// Create new Golang command, defining goroot
func Golang(gopath string) thinker.Cmd {
	return thinker.Cmd{
		Cmd:   GOLANG,
		About: "Compiles and runs Golang programs, scripts as package main.",
		Args: []thinker.Arg{
			{
				Name:  "script",
				Type:  "string",
				About: `script is main.go file containing the the program, it is executed with go run main.go.`,
			},
		},
		Run: golang(gopath),
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

func golang(gopath string) func(json.RawMessage) ([]byte, error) {
	return func(command json.RawMessage) ([]byte, error) {
		if err := goSetup(gopath); err != nil {
			return nil, err
		}

		var code script
		if err := json.Unmarshal(command, &code); err != nil {
			err := thinker.Feedback(
				"The input does not contain valid JSON object",
				"JSON parsing has failed with an error "+err.Error(),
			)
			return nil, err
		}

		file, err := gofile(gopath, code.Script)
		if err != nil {
			return nil, err
		}

		if err := goDeps(gopath); err != nil {
			return nil, err
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
				fmt.Sprintf("The tool %s has failed, improve the response based on feedback:", GOLANG),

				"Execution of golang program is failed with the error: "+err.Error(),
				"The error output is "+stderr.String(),
			)
			return nil, err
		}

		return stdout.Bytes(), nil
	}
}
