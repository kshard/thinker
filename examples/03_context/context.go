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
	"github.com/kshard/chatter/aio"
	"github.com/kshard/chatter/provider/autoconfig"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//------------------------------------------------------------------------------

type Input struct {
	A int `json:"a" jsonschema:"number a"`
	B int `json:"b" jsonschema:"number b"`
}

type Reply struct {
	C int `json:"c" jsonschema:"the result of the operation"`
}

func Add(ctx context.Context, req *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Reply, error) {
	return nil, Reply{C: input.A + input.B}, nil
}

func Multiply(ctx context.Context, req *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Reply, error) {
	return nil, Reply{C: input.A * input.B}, nil
}

func Subtract(ctx context.Context, req *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Reply, error) {
	return nil, Reply{C: input.A - input.B}, nil
}

//------------------------------------------------------------------------------

func encode(q int) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask(`
		Use available tools to complete the following tasks:
		1. Add the numbers 10 and 20 using the calculator
		2. Multiply the result by 2
		3. Subtract 15 from the final result
		4. Return the final answer

		Work step by step and show your calculations.
`)

	return &prompt, nil
}

func main() {
	srv := mcp.NewServer(&mcp.Implementation{Name: "calculator", Version: "v1.0.0"}, nil)
	mcp.AddTool(srv, &mcp.Tool{Name: "add", Description: "add two numbers"}, Add)
	mcp.AddTool(srv, &mcp.Tool{Name: "multiply", Description: "multiply two numbers"}, Multiply)
	mcp.AddTool(srv, &mcp.Tool{Name: "subtract", Description: "subtract b from a"}, Subtract)

	cli := mcp.NewClient(
		&mcp.Implementation{Name: "client1", Version: "v1.0.0"},
		&mcp.ClientOptions{},
	)

	tcli, tsrv := mcp.NewInMemoryTransports()

	go srv.Run(context.Background(), tsrv)

	session, err := cli.Connect(context.Background(), tcli, nil)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	registry := command.NewRegistry()
	registry.Attach("calc", session)

	// create instance of LLM API
	llm, err := autoconfig.FromNetRC("thinker")
	if err != nil {
		panic(err)
	}

	// Create agent with the provided registry
	agt := agent.NewManifold(
		aio.NewJsonLogger(os.Stdout, llm),
		codec.FromEncoder(encode),
		codec.String,
		registry,
	)

	// Execute agent
	out, err := agt.Prompt(context.Background(), 5)
	if err != nil {
		fmt.Printf("\n==> failure %v\n", err)
	}

	fmt.Printf("\n==> %s\n", out)
}
