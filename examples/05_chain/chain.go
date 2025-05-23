//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kshard/chatter"
	"github.com/kshard/chatter/llm/autoconfig"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/agent/worker"
	"github.com/kshard/thinker/command/softcmd"
)

//------------------------------------------------------------------------------

// The agent creates short-stoty, see Hello World example for details
type AgentA struct {
	*agent.Prompter[string]
}

func NewAgentA(llm chatter.Chatter) *AgentA {
	agt := &AgentA{}
	agt.Prompter = agent.NewPrompter(llm, agt.story)
	return agt
}

func (AgentA) story(subj string) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask("Create a short story about 140 characters about %s.", subj)
	return &prompt, nil
}

//------------------------------------------------------------------------------

// The agent creates workflow to process local files, see Script example for details
type AgentB struct {
	*worker.Reflex[string]
}

func NewAgentB(llm chatter.Chatter) *AgentB {
	registry := softcmd.NewRegistry()
	registry.Register(softcmd.Bash("MacOS", "/tmp/script"))

	agt := &AgentB{}
	agt.Reflex = worker.NewReflex(llm, 4, thinker.Encoder[string](agt), registry)

	return agt
}

func (agt AgentB) Encode(string) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask(`
		Use available tools to complete the workflow:
		(1) Use available tools to read files one by one.
		(2) Analyse file content and answer the question: Who is main character in the story? Remember the answer in your context.
		(3) Return all Remembered answers as comma separated string.`)

	return &prompt, nil
}

//------------------------------------------------------------------------------

func main() {
	// create instance of LLM API, see doc/HOWTO.md for details
	llm, err := autoconfig.New("thinker")
	if err != nil {
		panic(err)
	}

	// create instance of agents
	agtA := NewAgentA(llm)
	agtB := NewAgentB(llm)

	//
	// chaining agents using pure Go
	for _, who := range []string{"Cat", "Dog", "Cow", "Pig"} {
		// Use agent to transform animal input into story
		story, err := agtA.PromptOnce(context.Background(), who)
		if err != nil {
			panic(err)
		}

		// Write stories into file system
		err = txt2file(story)
		if err != nil {
			panic(err)
		}
	}

	// Use agent to conduct analysis of local files
	reply, err := agtB.PromptOnce(context.Background(), "")
	if err != nil {
		panic(err)
	}

	fmt.Printf("==> %s\n", reply.Output)
}

func txt2file(x *chatter.Reply) error {
	fd, err := os.CreateTemp("/tmp/script", "*.txt")
	if err != nil {
		return err
	}
	defer fd.Close()
	if _, err := fd.WriteString(x.String()); err != nil {
		return err
	}
	return nil
}
