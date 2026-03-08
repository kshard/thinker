//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

// Package command implements a registry for Model Context Protocol (MCP) servers.
// It provides integration with MCP tools, allowing agents to dynamically discover
// and invoke tools exposed by MCP servers.
package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP Server interface
type Server interface {
	ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error)
	Close() error
}

const kSchemaSplit = "_"

type Registry struct {
	servers map[string]Server
	cmds    chatter.Registry
}

var _ thinker.Registry = (*Registry)(nil)

// NewRegistry creates a new registry of MCP servers.
func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]Server),
		cmds:    chatter.Registry{},
	}
}

func (r *Registry) ConnectUrl(id string, url string) error {
	// TODO: implement connection closing
	rpc, err := NewAuthTransport(AuthConfig{Endpoint: url})
	if err != nil {
		return err
	}

	cli := mcp.NewClient(&mcp.Implementation{Name: id}, nil)
	api, err := cli.Connect(context.Background(), rpc, nil)
	if err != nil {
		return err
	}

	err = r.Attach(id, api)
	if err != nil {
		return err
	}

	return nil
}

func (r *Registry) ConnectCmd(id string, cmd []string) error {
	// TODO: implement connection closing

	run := exec.Command(cmd[0], cmd[1:]...)
	rpc := &mcp.CommandTransport{Command: run}
	cli := mcp.NewClient(&mcp.Implementation{Name: id}, nil)
	api, err := cli.Connect(context.Background(), rpc, nil)
	if err != nil {
		return err
	}

	err = r.Attach(id, api)
	if err != nil {
		return err
	}

	return nil
}

// Attach MCP server to the registry, making its tools available to the agent.
// The server is identified by a unique prefix, which is used to namespace
// tool names (e.g., fs_read). Tool names use underscore separator (prefix_toolname)
// due to AWS Bedrock constraints which only allow [a-zA-Z0-9_-] characters.
// The first token before underscore is always the server prefix.
// Each server runs independently, and its tools are registered with the prefix
// to avoid naming conflicts.
func (r *Registry) Attach(id string, server Server) error {
	if id == "" {
		return fmt.Errorf("server ID cannot be empty")
	}

	r.servers[id] = server
	r.cmds = chatter.Registry{}

	return nil
}

// Context returns the registry as LLM embeddable schema.
// It fetches the list of available tools from all attached MCP servers.
func (r *Registry) Context() chatter.Registry {
	// Return cached if available
	if len(r.cmds) > 0 {
		return r.cmds
	}

	ctx := context.Background()
	seq := make([]chatter.Cmd, 0)

	// Collect tools from all attached servers
	for id, srv := range r.servers {
		tools, err := srv.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			continue
		}

		for _, tool := range tools.Tools {
			cmd := convertTool(*tool, id)
			seq = append(seq, cmd)
		}
	}

	r.cmds = seq
	return r.cmds
}

// Invoke executes the tools requested by the LLM via the appropriate MCP server.
func (r *Registry) Invoke(reply *chatter.Reply) (thinker.Phase, chatter.Message, error) {
	ctx := context.Background()
	answer, err := reply.Invoke(func(name string, args json.RawMessage) (json.RawMessage, error) {
		seq := strings.SplitN(name, kSchemaSplit, 2)
		if len(seq) != 2 {
			return pack(
				fmt.Appendf(nil, "invalid tool name %s, missing the prefix", name),
			)
		}
		id, tool := seq[0], seq[1]

		// Find which server handles this tool
		srv, exists := r.servers[id]
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

// convertTool converts an MCP Tool to a chatter.Cmd format.
func convertTool(tool mcp.Tool, prefix string) chatter.Cmd {
	about := tool.Description
	if about == "" && tool.Title != "" {
		about = tool.Title
	}

	var schema json.RawMessage
	if tool.InputSchema != nil {
		if raw, ok := tool.InputSchema.(json.RawMessage); ok {
			schema = raw
		} else {
			// Convert to JSON if not already RawMessage
			if b, err := json.Marshal(tool.InputSchema); err == nil {
				schema = json.RawMessage(b)
			}
		}
	}

	// Apply prefix if set (compact IRI notation: prefix:tool_name)
	name := tool.Name
	if prefix != "" {
		name = prefix + kSchemaSplit + tool.Name
	}

	return chatter.Cmd{
		Cmd:    name,
		About:  about,
		Schema: schema,
	}
}

// extractContent extracts the text content from a CallToolResult.
func extractContent(result *mcp.CallToolResult) []byte {
	if len(result.Content) == 0 {
		return []byte{}
	}

	// Try to extract text content
	for _, content := range result.Content {
		switch c := content.(type) {
		case *mcp.TextContent:
			return []byte(c.Text)
		case *mcp.ImageContent:
			// For image content, return a description or empty
			return []byte("[Image content]")
		}
	}

	// Fallback: try to marshal the content as JSON
	if b, err := json.Marshal(result.Content); err == nil {
		return b
	}

	return []byte{}
}

// pack wraps the tool output in the expected format.
func pack(b []byte) (json.RawMessage, error) {
	pckt := map[string]any{
		"toolOutput": string(b),
	}

	bin, err := json.Marshal(pckt)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(bin), nil
}
