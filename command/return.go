package command

import (
	"encoding/json"

	"github.com/kshard/thinker"
)

// A unique name for return command
const RETURN = "return"

// Creates new return command, instructing LLM return results
func Return() thinker.Cmd {
	return thinker.Cmd{
		Cmd:   RETURN,
		About: "indicate that workflow is completed and returns the expected result.",
		Args: []thinker.Arg{
			{
				Name:  "value",
				Type:  "string",
				About: `value to return as the workflow completion`,
			},
		},
		Run: func(in json.RawMessage) ([]byte, error) {
			var reply replyReturn
			if err := json.Unmarshal(in, &reply); err != nil {
				return nil, err
			}

			return []byte(reply.Value), nil
		},
	}
}

type replyReturn struct {
	Value string `json:"value,omitempty"`
}
