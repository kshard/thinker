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

	"github.com/fogfish/golem/pipe"
	"github.com/fogfish/golem/pure/monoid"
	"github.com/kshard/chatter"
	"github.com/kshard/chatter/bedrock"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/kshard/thinker/memory"
	"github.com/kshard/thinker/reasoner"
)

//------------------------------------------------------------------------------

// The agent creates short-stoty, see Hello World example for details
type AgentA struct {
	*agent.Automata[string, string]
}

func NewAgentA(llm chatter.Chatter) *AgentA {
	agt := &AgentA{}
	agt.Automata = agent.NewAutomata(llm,
		memory.NewVoid(),
		reasoner.NewVoid[string, string](),
		codec.FromEncoder(agt.story),
		codec.DecoderID,
	)

	return agt
}

func (AgentA) story(subj string) (prompt chatter.Prompt, err error) {
	prompt.WithTask("Create a short story about 140 characters about %s.", subj)
	return
}

//------------------------------------------------------------------------------

// The agent creates workflow to process local files, see Script example for details
type AgentB struct {
	*agent.Automata[string, thinker.CmdOut]
	registry *command.Registry
}

func NewAgentB(llm chatter.Chatter) *AgentB {
	agt := &AgentB{}

	agt.registry = command.NewRegistry()
	agt.registry.Register(command.Bash("MacOS", "/tmp/script"))
	agt.registry.Register(command.Return())

	agt.Automata = agent.NewAutomata(llm,
		memory.NewStream(`
			You are automomous agent who uses tools to perform required tasks.
			You are using and remember context from earlier chat history to execute the task.
		`),
		reasoner.NewEpoch(4, reasoner.From(agt.deduct)),
		codec.FromEncoder(agt.encode),
		agt.registry,
	)

	return agt
}

func (agt AgentB) encode(string) (prompt chatter.Prompt, err error) {
	prompt.WithTask(`
		Use available tools to complete the workflow:
		(1) Use available tools to read files one by one.
		(2) Analyse file content and answer the question: Who is main character in the story? Remember the answer in your context.
		(3) Return all Remembered answers as comma separated string.`)

	// Inject tools
	agt.registry.Harden(&prompt)
	return
}

func (AgentB) deduct(state thinker.State[string, thinker.CmdOut]) (thinker.Phase, chatter.Prompt, error) {
	// the registry has failed to execute command, we have to supply the feedback to LLM
	if state.Feedback != nil && state.Confidence < 1.0 {
		var prompt chatter.Prompt
		prompt.WithTask("Refine the previous workstop step using the feedback below.")
		prompt.With(state.Feedback)

		return thinker.AGENT_REFINE, prompt, nil
	}

	// the workflow has successfully completed
	// Note: pseudo-command return is executed
	if state.Reply.Cmd == command.RETURN {
		return thinker.AGENT_RETURN, chatter.Prompt{}, nil
	}

	// the workflow step is completed
	if state.Reply.Cmd == command.BASH {
		var prompt chatter.Prompt
		prompt.WithTask("Continue the workflow execution.")
		prompt.With(
			chatter.Input("The command has returned:\n", state.Reply.Output),
		)

		return thinker.AGENT_ASK, prompt, nil
	}

	return thinker.AGENT_ABORT, chatter.Prompt{}, fmt.Errorf("unknown state")
}

//------------------------------------------------------------------------------

func main() {
	// create instance of LLM client
	llm, err := bedrock.New(
		bedrock.WithLLM(bedrock.LLAMA3_1_70B_INSTRUCT),
		bedrock.WithRegion("us-west-2"),
	)
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
	story := pipe.StdErr(pipe.Map(ctx, who,
		func(x string) (string, error) {
			return agtA.Prompt(context.Background(), x)
		},
	))

	// Write stories into file system
	file := pipe.StdErr(pipe.Map(ctx, story, txt2file))

	// Wait until all files are written
	syn := pipe.Fold(ctx, file, mString)

	// Use agent to conduct analysis of local files
	act := pipe.StdErr(pipe.Map(ctx, syn,
		func(x string) (thinker.CmdOut, error) {
			return agtB.Prompt(context.Background(), x)
		},
	))

	// Output the result of the pipeline
	<-pipe.ForEach(ctx, act,
		func(x thinker.CmdOut) { fmt.Printf("==> %s\n", x.Output) },
	)

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
