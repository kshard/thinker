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

func TestPython(t *testing.T) {
	cmd := Python("/tmp/cmd")
	script := `
import requests
response = requests.get('https://example.com/')
print(response)
`
	reply, _ := json.Marshal(map[string]any{"script": script})
	out, err := cmd.Run(json.RawMessage(reply))

	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("Response [200]"),
	)
}
