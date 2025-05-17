//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command

import (
	"encoding/json"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	t.Run("Register", func(t *testing.T) {
		cmd := Bash("MacOS", "/tmp")
		err := r.Register(cmd)

		it.Then(t).Should(
			it.Nil(err),
		)
	})

	t.Run("Conflict", func(t *testing.T) {
		cmd := Return()
		err := r.Register(cmd)

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})

	t.Run("Context", func(t *testing.T) {
		reg := r.Context()

		it.Then(t).Should(
			it.Seq(reg).Contain(
				chatter.Cmd{
					Cmd:    RETURN,
					About:  "indicate that workflow is completed and returns the expected result.",
					Schema: json.RawMessage(`{"properties":{"value":{"type":"string","description":"value to return as the workflow completion"}},"required":["value"],"type":"object"}`),
				},
			),
		)
	})

}
