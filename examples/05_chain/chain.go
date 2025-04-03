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

	"github.com/fogfish/golem/pipe/v2"
	"github.com/fogfish/golem/pure/monoid"
	"github.com/kshard/chatter"
	"github.com/kshard/chatter/llm/autoconfig"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/command"
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

func (AgentA) story(subj string) (prompt chatter.Prompt, err error) {
	prompt.WithTask("Create a short story about 140 characters about %s.", subj)
	return
}

//------------------------------------------------------------------------------

// The agent creates workflow to process local files, see Script example for details
type AgentB struct {
	*agent.Worker[string]
}

func NewAgentB(llm chatter.Chatter) *AgentB {
	registry := command.NewRegistry()
	registry.Register(command.Bash("MacOS", "/tmp/script"))

	agt := &AgentB{}
	agt.Worker = agent.NewWorker(llm, 4, thinker.Encoder[string](agt), registry)

	return agt
}

func (agt AgentB) Encode(string) (prompt chatter.Prompt, err error) {
	prompt.WithTask(`
		Use available tools to complete the workflow:
		(1) Use available tools to read files one by one.
		(2) Analyse file content and answer the question: Who is main character in the story? Remember the answer in your context.
		(3) Return all Remembered answers as comma separated string.`)

	return
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
	// chaining agents using Go channels and fogfish/golem/pipe

	// create context to manage the chain
	ctx, close := context.WithCancel(context.Background())

	// Input to the chain
	who := pipe.Seq("Cat", "Dog", "Cow", "Pig")

	// Use agent to transform input into story
	story := pipe.StdErr(pipe.Map(ctx, who, pipe.Lift(agtA.Seek)))

	// Write stories into file system
	file := pipe.StdErr(pipe.Map(ctx, story, pipe.Lift(txt2file)))

	// Wait until all files are written
	syn := pipe.Fold(ctx, file, mString)

	// Use agent to conduct analysis of local files
	act := pipe.StdErr(pipe.Map(ctx, syn, pipe.Lift(agtB.Echo)))

	// Output the result of the pipeline
	<-pipe.ForEach(ctx, act, pipe.Pure(stdout))

	close()
}

func txt2file(x string) (string, error) {
	fd, err := os.CreateTemp("/tmp/script", "*.txt")
	if err != nil {
		return "", err
	}
	defer fd.Close()
	if _, err := fd.WriteString(x); err != nil {
		return "", err
	}
	return fd.Name(), nil
}

// naive string monoid
var mString = monoid.FromOp("", func(a string, b string) string { return a + " " + b })

func stdout(x thinker.CmdOut) thinker.CmdOut {
	fmt.Printf("==> %s\n", x.Output)
	return x
}
