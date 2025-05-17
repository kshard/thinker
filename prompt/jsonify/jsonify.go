//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package jsonify

import (
	"encoding/json"
	"regexp"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

var re = regexp.MustCompile(`(?s)(\{.*\}|\[.*\])`)

// Helper to prompt and decode json array of strings
const Strings = strings("thinker.prompt.jsonify")

type strings string

// Injects requirments for LLM to return json array of strings
func (strings) Harden(prompt *chatter.Prompt) {
	prompt.WithRules(
		`Strictly adhere to the following requirements when generating a response.
			Do not deviate, ignore, or modify any aspect of them:`,

		"The output should be JSON list of strings.",
		"Do not generate unknowns, reply [] if you do not know the answer.",
	)
}

// Parse LLMs response, searching for json array of strings.
// It returns the feedback to LLM if response does not contain valid json.
func (strings) Decode(reply *chatter.Reply, seq any) error {
	matches := re.FindStringSubmatch(string(reply.String()))
	if len(matches) == 0 {
		return thinker.Feedback(
			`Improve the response based on feedback:`,
			"The output does not contain valid JSON list of strings.",
			"No pattern [ \"string\", \"string\", ... ] is found in the output.",
		)
	}

	if err := json.Unmarshal([]byte(matches[0]), &seq); err != nil {
		return thinker.Feedback(
			`Improve the response based on feedback.`,
			"The output does not contain valid JSON list of strings.",
			"JSON parsing of included list of strings has failed with an error  "+err.Error(),
		)
	}

	return nil
}
