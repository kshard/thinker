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
	"github.com/kshard/chatter/aio"
	"github.com/kshard/chatter/provider/autoconfig"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func encode(q string) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask(`
		Use available tools to complete the workflow:
		(1) Create 5 files one by one with few lines of random text, at least one line shall contain "%s".
		(2) Use available tools to find files containing the string: "%s".
		(3) Replace the only found string with "XXXXX".
		(4) Validate completion of task by checking "%s" in the files and fix your self if it still exists.`, q, q, q)

	return &prompt, nil
}

func main() {
	// create MCP Server with `bash` tool
	session := server()
	defer session.Close()

	// enable `shell` command for the agent, the command is sandbox to the dir.
	registry := command.NewRegistry()
	registry.Attach("os", session)

	// registry.Register(command.Bash("MacOS", "/tmp/script"))

	// create instance of LLM API, see doc/HOWTO.md for details
	llm, err := autoconfig.FromNetRC("thinker")
	if err != nil {
		panic(err)
	}

	// We create an agent that executes the workflow.
	agt := agent.NewManifold(
		// enable debug output for LLMs dialog
		aio.NewJsonLogger(os.Stdout, llm),

		// Configures the encoder to transform input of type A into a `chatter.Prompt`.
		// Here, we use an encoder that builds prompt.
		codec.FromEncoder(encode),
		codec.DecoderID,

		// Configure the decoder to transform output of LLM into type B.
		// Here, we use the tool registry to "decode" output into command call.
		registry,
	)

	// Execute agent
	_, err = agt.Prompt(context.Background(), "Hello World")
	fmt.Printf("\n\n==> Err: %v", err)
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
