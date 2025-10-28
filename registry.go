//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package thinker

import "github.com/kshard/chatter"

type Registry interface {
	// Registry context as LLM embeddable schema
	Context() chatter.Registry

	// Invoke the registry
	Invoke(reply *chatter.Reply) (Phase, chatter.Message, error)
}
