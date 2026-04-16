//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package main

import (
	"context"
	"fmt"

	"github.com/kshard/chatter/provider/autoconfig"
	"github.com/kshard/thinker/agent/nanobot"
	"github.com/kshard/thinker/command"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	// Configure access to LLMs
	llm = autoconfig.MustFrom(autoconfig.Instance{
		Name:     "base",
		Provider: "provider:bedrock/foundation/converse",
		Model:    "global.anthropic.claude-sonnet-4-5-20250929-v1:0",
	})

	// Create nanobot runtime.
	env = nanobot.NewRuntime(nil, llm).
		WithRegistry(
			// This example uses native MCP server, but in the real app external
			// one would be used, which is automatically discovered by bots itself.
			command.NewRegistry().WithNative("calc", fMul),
		)

	// Create the ReAct agent using the prompt file.
	bot = nanobot.ReAct[float32, string](env,
		"data:text/markdown,What is a 15%% tip on a ${{ . }} bill?",
	)
)

func main() {
	val, err := bot.Prompt(context.Background(), 120)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", val)
}

//
// Native MCP Server for number multiplication.
// In a real application, this could be an external service running in
// a separate process or container, and the agent would connect to
// it over HTTP or another protocol.
//
// For simplicity, we define it inline here.
//

var fMul = command.From(&mcp.Tool{Name: "mul", Description: "multiply two numbers"}, mul)

type input struct {
	A float32 `json:"a" jsonschema:"number a"`
	B float32 `json:"b" jsonschema:"number b"`
}

type reply struct {
	C float32 `json:"c" jsonschema:"the result of the operation"`
}

func mul(ctx context.Context, req *mcp.CallToolRequest, input input) (*mcp.CallToolResult, reply, error) {
	return nil, reply{C: input.A * input.B}, nil
}
