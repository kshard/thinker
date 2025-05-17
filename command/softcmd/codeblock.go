//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package softcmd

import (
	"fmt"
	"strings"

	"github.com/kshard/thinker"
)

func CodeBlock(tool string, str string) (string, error) {
	const btag = "<codeblock>"
	const etag = "</codeblock>"

	b := strings.Index(str, btag)
	e := strings.Index(str, etag)
	if b == -1 || e == -1 || b >= e {
		err := thinker.Feedback(
			fmt.Sprintf("The TOOL:%s has failed, improve the response based on feedback:", tool),

			fmt.Sprintf(`Strictly adhere %s code formatting and enclose the %s code in <codeblock> tags`, tool, tool),
		)
		return "", err
	}

	code := str[b+len(btag) : e]
	return code, nil
}
