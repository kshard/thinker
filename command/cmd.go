package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"text/template"

	"github.com/kshard/thinker"
)

func Cmd(dir string, spec thinker.Cmd) thinker.Cmd {
	spec.Run = cmd(dir, spec)
	return spec
}

func cmd(dir string, spec thinker.Cmd) func(json.RawMessage) ([]byte, error) {
	t := template.Must(template.New(spec.Cmd).Parse(spec.Syntax))

	return func(command json.RawMessage) ([]byte, error) {
		var in map[string]any
		if err := json.Unmarshal(command, &in); err != nil {
			err := thinker.Feedback(
				"The input does not contain valid JSON object",
				"JSON parsing has failed with an error "+err.Error(),
			)
			return []byte(err.Error()), nil
		}

		var sb strings.Builder
		if err := t.Execute(&sb, in); err != nil {
			return nil, err
		}

		cmd := exec.Command("bash", "-c", sb.String())
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Dir = dir
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			err = thinker.Feedback(
				fmt.Sprintf("The tool %s has failed, improve the response based on feedback:", spec.Cmd),

				"Execution of the tool is failed with the error: "+err.Error(),
				"The stderr is "+stderr.String(),
			)

			return nil, err
		}

		return stdout.Bytes(), nil
	}
}
