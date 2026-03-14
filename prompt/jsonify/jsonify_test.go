//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package jsonify_test

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/prompt/jsonify"
)

func TestStrings(t *testing.T) {
	t.Run("Harden", func(t *testing.T) {
		var prompt chatter.Prompt
		jsonify.Strings.Harden(&prompt, nil)

		str := prompt.String()

		it.Then(t).Should(
			it.String(str).Contain("Strictly adhere to the following requirements"),
			it.String(str).Contain("ALWAYS reply with valid JSON only"),
			it.String(str).Contain("The JSON must exactly match the schema"),
		)
	})

	t.Run("Decode", func(t *testing.T) {
		var seq []string
		reply := &chatter.Reply{
			Content: []chatter.Content{
				chatter.Text(` ["a", "b", "c"] `),
			},
		}
		err := jsonify.Strings.Decode(reply, nil, &seq)

		it.Then(t).Should(
			it.Nil(err),
			it.Seq(seq).Equal("a", "b", "c"),
		)
	})

	t.Run("DecodeErrors", func(t *testing.T) {
		for in, ex := range map[string]string{
			"abc":       "The output does not contain valid JSON array or object.",
			"[a, b, c]": "JSON parsing of included list of strings has failed",
		} {
			var seq []string
			reply := &chatter.Reply{
				Content: []chatter.Content{
					chatter.Text(in),
				},
			}
			err := jsonify.Strings.Decode(reply, nil, &seq)

			it.Then(t).Should(
				it.String(err.Error()).Contain(ex),
			)
		}
	})

	t.Run("HardenWithSchema", func(t *testing.T) {
		schema := &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {Type: "string"},
				"age":  {Type: "integer"},
			},
			Required: []string{"name", "age"},
		}

		var prompt chatter.Prompt
		jsonify.Strings.Harden(&prompt, schema)

		str := prompt.String()

		it.Then(t).Should(
			it.String(str).Contain("Strictly adhere to the following requirements"),
			it.String(str).Contain("ALWAYS reply with valid JSON only"),
			it.String(str).Contain("Produce JSON object with the exact fields below"),
			it.String(str).Contain("Expected JSON schema:"),
			it.String(str).Contain(`"type": "object"`),
		)
	})

	t.Run("DecodeWithSchemaValid", func(t *testing.T) {
		schema := &jsonschema.Schema{
			Type: "array",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		}

		var seq []string
		reply := &chatter.Reply{
			Content: []chatter.Content{
				chatter.Text(`["valid", "strings", "array"]`),
			},
		}
		err := jsonify.Strings.Decode(reply, schema, &seq)

		it.Then(t).Should(
			it.Nil(err),
			it.Seq(seq).Equal("valid", "strings", "array"),
		)
	})

	// t.Run("DecodeWithSchemaInvalid", func(t *testing.T) {
	// 	minItems := 5
	// 	schema := &jsonschema.Schema{
	// 		Type: "array",
	// 		Items: &jsonschema.Schema{
	// 			Type: "string",
	// 		},
	// 		MinItems: &minItems,
	// 	}

	// 	var seq []string
	// 	reply := &chatter.Reply{
	// 		Content: []chatter.Content{
	// 			chatter.Text(`["only", "three", "items"]`),
	// 		},
	// 	}
	// 	err := jsonify.Strings.Decode(reply, schema, &seq)
	//
	// 	it.Then(t).ShouldNot(
	// 		it.Nil(err),
	// 	)
	// 	it.Then(t).Should(
	// 		it.String(err.Error()).Contain("JSON has failed schema validation"),
	// 	)
	// })
}
