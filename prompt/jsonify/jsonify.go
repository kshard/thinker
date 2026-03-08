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
	"fmt"
	"regexp"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
)

var re = regexp.MustCompile(`(?s)(\{.*\}|\[.*\])`)

// Helper to prompt and decode json array of strings
const Strings = strings("thinker.prompt.jsonify")

type strings string

// Injects requirments for LLM to return json array of strings
func (strings) Harden(prompt *chatter.Prompt, schema *jsonschema.Schema) {
	prompt.WithRules(
		`Strictly adhere to the following requirements when generating a response. You must not deviate, ignore, or modify any aspect of them:`,

		`(1) You are a JSON-only responder.`,
		`(2) ALWAYS reply with valid JSON only. No explanation, no commentary, no markdown, no code fences, no extra characters before/after the JSON.`,
		`(3) The JSON must exactly match the schema or structure requested by the user prompt.`,
		`(4) If any required field cannot be determined, set it to null (do not omit it).`,
		`(5) If an error occurs or the request cannot be completed, return {"error": "<explain briefly why>", "partial": <true|false>}.`,
		`(6) Do NOT reveal chain-of-thought or reasoning.`,
		`(7) Do NOT include trailing commas or comments.`,
		`(8) Output must be UTF-8 and must be parsable by a standard JSON.parse().`,
	)

	if schema != nil {
		if schemaJSON, err := json.MarshalIndent(schema, "", "  "); err == nil {
			prompt.WithRules(
				"Produce JSON object with the exact fields below (names and types must match exactly). Return only the JSON object with no extra text.",
				fmt.Sprintf("Expected JSON schema:\n%s", string(schemaJSON)),
			)
		}
	}
}

// Parse LLMs response, searching for json array of strings.
// It returns the feedback to LLM if response does not contain valid json.
func (strings) Decode(reply *chatter.Reply, schema *jsonschema.Schema, seq any) error {
	matches := re.FindStringSubmatch(string(reply.String()))
	if len(matches) == 0 {
		return thinker.Feedback(
			`Improve the response based on feedback:`,
			"The output does not contain valid JSON array or object.",
			// "No pattern [ \"string\", \"string\", ... ] is found in the output.",
		)
	}

	if err := json.Unmarshal([]byte(matches[0]), &seq); err != nil {
		return thinker.Feedback(
			`Improve the response based on feedback.`,
			"The output does not contain valid JSON array or object.",
			"JSON parsing of included list of strings has failed with an error  "+err.Error(),
		)
	}

	if schema != nil {
		resolved, err := schema.Resolve(nil)
		if err != nil {
			return err
		}

		if err := resolved.Validate(seq); err != nil {
			return thinker.Feedback(
				`Improve the response based on feedback.`,
				"The output does not contain required JSON format.",
				"JSON has failed schema validation: "+err.Error(),
			)
		}
	}

	return nil
}
