//
// Copyright (C) 2025 - 2026Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SeqRegistry struct {
	regs []*Registry
	cmds chatter.Registry
}

var _ thinker.Registry = (*SeqRegistry)(nil)

func NewSeqRegistry() *SeqRegistry {
	return &SeqRegistry{
		regs: make([]*Registry, 0),
		cmds: chatter.Registry{},
	}
}

func (r *SeqRegistry) Bind(reg *Registry) {
	if reg == nil {
		return
	}

	r.regs = append(r.regs, reg)
}

func (r *SeqRegistry) Context() chatter.Registry {
	if len(r.cmds) > 0 {
		return r.cmds
	}

	seq := make([]chatter.Cmd, 0)
	for _, reg := range r.regs {
		seq = append(seq, reg.Context()...)
	}

	r.cmds = seq
	return r.cmds
}

func (r *SeqRegistry) Invoke(reply *chatter.Reply) (phase thinker.Phase, msg chatter.Message, err error) {
	ctx := context.Background()
	answer, err := reply.Invoke(func(name string, args json.RawMessage) (json.RawMessage, error) {
		seq := strings.SplitN(name, kSchemaSplit, 2)
		if len(seq) != 2 {
			return pack(
				fmt.Appendf(nil, "invalid tool name %s, missing the prefix", name),
			)
		}
		id, tool := seq[0], seq[1]

		var srv Server
		var exists bool
		for _, reg := range r.regs {
			srv, exists = reg.servers[id]
			if !exists {
				continue
			}
		}
		if !exists {
			return pack(
				fmt.Appendf(nil, "tool %s is not available in any attached MCP server", name),
			)
		}

		// Unmarshal arguments to pass to MCP
		var arguments map[string]any
		if len(args) > 0 {
			if err := json.Unmarshal(args, &arguments); err != nil {
				return pack(
					fmt.Appendf(nil, "failed to parse arguments for tool %s: %v", name, err),
				)
			}
		}

		// Call the tool via MCP using the actual tool name (without prefix)
		result, err := srv.CallTool(ctx, &mcp.CallToolParams{
			Name:      tool,
			Arguments: arguments,
		})
		if err != nil {
			return pack(
				fmt.Appendf(nil, "the tool %s execution is failed: %s", name, err),
			)
		}

		// Handle tool execution errors
		if result.IsError {
			errorMsg := "tool execution failed"
			if len(result.Content) > 0 {
				if text, ok := result.Content[0].(*mcp.TextContent); ok {
					errorMsg = text.Text
				}
			}
			return pack([]byte(errorMsg))
		}

		// Extract and pack the result
		output := extractContent(result)
		return pack(output)
	})

	if err != nil {
		return thinker.AGENT_ABORT, nil, err
	}

	return thinker.AGENT_ASK, &answer, nil
}
