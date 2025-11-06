//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/kshard/chatter"
	"github.com/kshard/chatter/provider/autoconfig"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	*agent.Manifold[string, string]
}

func NewAgentB(llm chatter.Chatter) *AgentB {
	registry := command.NewRegistry()
	registry.Attach("os", server())

	agt := &AgentB{}
	agt.Manifold = agent.NewManifold(llm,
		thinker.Encoder[string](agt),
		codec.String,
		registry,
	)

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
	llm, err := autoconfig.FromNetRC("thinker")
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
	reply, err := agtB.Prompt(context.Background(), "")
	if err != nil {
		panic(err)
	}

	fmt.Printf("==> %s\n", reply)
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

//------------------------------------------------------------------------------

func server() *mcp.ClientSession {
	srv := mcp.NewServer(&mcp.Implementation{Name: "shell", Version: "v1.0.0"}, nil)
	mcp.AddTool(srv, &mcp.Tool{Name: "bash", Description: "execute bash commands"}, Bash)

	cli := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v1.0.0"}, nil)

	tcli, tsrv := mcp.NewInMemoryTransports()
	go srv.Run(context.Background(), tsrv)

	session, err := cli.Connect(context.Background(), tcli, nil)
	if err != nil {
		panic(err)
	}

	return session
}

type Input struct {
	Script string `json:"script" jsonschema:"bash script to executes"`
}

type Reply struct {
	Output string `json:"output" jsonschema:"output of bash command"`
}

func Bash(ctx context.Context, req *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Reply, error) {
	cmd := exec.Command("bash", "-c", input.Script)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Dir = "/tmp/script"
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, Reply{Output: "bash has failed with an error " + err.Error()}, nil
	}

	return nil, Reply{Output: stdout.String()}, nil
}
