//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command

import (
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
)

func TestPython(t *testing.T) {
	cmd := Python("/tmp")
	reply := &chatter.Reply{
		Content: []chatter.Content{
			chatter.ContentText{Text: `<codeblock>
import requests
response = requests.get('https://example.com/')
print(response)
</codeblock>
`},
		},
	}
	conf, out, err := cmd.Run(reply)

	it.Then(t).Should(
		it.Nil(err),
		it.Equal(conf, 1.0),
		it.Equal(out.Cmd, cmd.Cmd),
		it.String(out.Output).Contain("Response [200]"),
	)
}
