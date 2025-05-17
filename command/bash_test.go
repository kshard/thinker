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
)

func TestBash(t *testing.T) {
	cmd := Bash("", "/tmp")
	reply := json.RawMessage(`{"script": "ls"}`)
	out, err := cmd.Run(reply)

	it.Then(t).Should(
		it.Nil(err),
		it.Greater(len(out), 0),
	)
}
