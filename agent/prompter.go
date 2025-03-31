package agent

import (
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/reasoner"
)

// Prompter is memoryless and stateless agent, implementing request/response to LLMs
type Prompter[A any] struct {
	*Automata[A, string]
}

func NewPrompter[A any](llm chatter.Chatter, f func(A) (chatter.Prompt, error)) *Prompter[A] {
	w := &Prompter[A]{}
	w.Automata = NewAutomata(
		llm,
		memory.NewVoid(""),
		reasoner.NewVoid[string](),
		codec.FromEncoder(f),
		codec.DecoderID,
	)

	return w
}
