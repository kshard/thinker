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

func TestGolang(t *testing.T) {
	cmd := Golang("/tmp/cmd")
	script := `
package main

import "fmt"

func main() {
	fmt.Println("response")
}
`
	reply, _ := json.Marshal(map[string]any{"script": script})
	out, err := cmd.Run(json.RawMessage(reply))

	it.Then(t).Should(
		it.Nil(err),
		it.String(out).Contain("response"),
	)
}
