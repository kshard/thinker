package thinker

import "github.com/kshard/chatter"

type Registry interface {
	// Register a new command to the registry
	Register(Cmd) error

	// Registry context as LLM embeddable schema
	Context() chatter.Registry

	// Invoke the registry
	Invoke(reply *chatter.Reply) (Phase, chatter.Message, error)
}
