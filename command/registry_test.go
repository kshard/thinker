//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package command_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/fogfish/it/v2"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/command"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegistryConnectCmd(t *testing.T) {
	t.Run("NonExistentBinary", func(t *testing.T) {
		registry := command.NewRegistry()

		err := registry.ConnectCmd("fs", []string{"nonexistent-binary-xyz-thinker"})

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})

	t.Run("BinaryDoesNotSpeakMCP", func(t *testing.T) {
		registry := command.NewRegistry()

		// /usr/bin/true exits immediately without speaking MCP
		err := registry.ConnectCmd("fs", []string{"/usr/bin/true"})

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})

	t.Run("EmptyID", func(t *testing.T) {
		registry := command.NewRegistry()

		// Tools are not discovered before Attach validates the id, so the
		// connection error from the non-MCP binary is hit first.  Either way,
		// a non-nil error must be returned.
		err := registry.ConnectCmd("", []string{"/usr/bin/true"})

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})
}

func TestRegistryConnectUrl(t *testing.T) {
	t.Run("ConnectionRefused", func(t *testing.T) {
		registry := command.NewRegistry()

		// Nothing is listening on this port; the MCP client must return an error.
		err := registry.ConnectUrl("remote", "http://127.0.0.1:19999/mcp")

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})

	t.Run("EmptyID", func(t *testing.T) {
		registry := command.NewRegistry()

		// Connect will fail (connection refused) before Attach validates the id.
		// Either way, a non-nil error must be returned.
		err := registry.ConnectUrl("", "http://127.0.0.1:19999/mcp")

		it.Then(t).ShouldNot(
			it.Nil(err),
		)
	})
}

func TestNewRegistry(t *testing.T) {
	registry := command.NewRegistry()

	it.Then(t).Should(
		it.True(registry != nil),
	)
}

func TestRegistryAttach(t *testing.T) {
	t.Run("AttachSingleServer", func(t *testing.T) {
		registry := command.NewRegistry()

		err := registry.Attach("fs", mockOne("tool", "A test tool"))

		ctx := registry.Context()
		it.Then(t).Should(
			it.Nil(err),
			it.Equal(len(ctx), 1),
		)
	})

	t.Run("AttachMultipleServers", func(t *testing.T) {
		registry := command.NewRegistry()

		registry.Attach("fs", mockOne("read", "Read file"))
		registry.Attach("db", mockOne("query", "Query database"))

		ctx := registry.Context()
		it.Then(t).Should(
			it.Equal(len(ctx), 2),
		)
	})

	t.Run("AttachWithoutPrefix", func(t *testing.T) {
		registry := command.NewRegistry()

		err := registry.Attach("", mockOne("tool", "A test tool"))

		ctx := registry.Context()
		it.Then(t).ShouldNot(
			it.Nil(err),
		).Should(
			it.Equal(len(ctx), 0),
		)
	})
}

func TestRegistryContext(t *testing.T) {
	t.Run("ContextWithPrefix", func(t *testing.T) {
		registry := command.NewRegistry()

		registry.Attach("fs", mockSeq(2, "read", "Read file"))

		ctx := registry.Context()
		seq := make([]string, len(ctx))
		for i, c := range ctx {
			seq[i] = c.Cmd
		}

		it.Then(t).Should(
			it.Equal(len(ctx), 2),
			it.Seq(seq).Contain("fs_read.0", "fs_read.1"),
		)
	})

	t.Run("ContextMultipleServers", func(t *testing.T) {
		registry := command.NewRegistry()

		registry.Attach("fs", mockSeq(1, "read", "Read file"))
		registry.Attach("db", mockSeq(1, "query", "Query database"))

		ctx := registry.Context()
		seq := make([]string, len(ctx))
		for i, c := range ctx {
			seq[i] = c.Cmd
		}

		it.Then(t).Should(
			it.Equal(len(ctx), 2),
			it.Seq(seq).Contain("fs_read.0", "db_query.0"),
		)
	})
}

func TestRegistryInvoke(t *testing.T) {
	t.Run("InvokePrefixedTool", func(t *testing.T) {
		registry := command.NewRegistry()

		registry.Attach("fs", mockReply("read", "Read file", "file contents"))
		registry.Context()

		reply := replyOne("fs_read", map[string]any{"path": "/test.txt"})
		phase, msg, err := registry.Invoke(&reply)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_ASK),
			it.True(msg != nil),
		)
	})

	t.Run("InvokeUnprefixedTool", func(t *testing.T) {
		registry := command.NewRegistry()

		registry.Attach("fs", mockReply("read", "Read file", "file contents"))
		registry.Context()

		reply := replyOne("tool", map[string]any{})
		phase, _, err := registry.Invoke(&reply)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_ASK),
		)
	})

	t.Run("InvokeUnknownTool", func(t *testing.T) {
		registry := command.NewRegistry()

		registry.Attach("fs", mockReply("read", "Read file", "file contents"))
		registry.Context()

		reply := replyOne("unknown:tool", map[string]any{})

		phase, _, err := registry.Invoke(&reply)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_ASK),
		)
	})

	t.Run("InvokeWithInvalidArgs", func(t *testing.T) {
		registry := command.NewRegistry()

		registry.Attach("fs", mockReply("read", "Read file", "file contents"))
		registry.Context()

		// Create a reply with invalid JSON args
		reply := chatter.Reply{
			Stage: chatter.LLM_INVOKE,
			Content: []chatter.Content{
				chatter.Invoke{
					Cmd:  "fs_read",
					Args: chatter.Json{Value: []byte("invalid json")},
				},
			},
		}

		phase, _, err := registry.Invoke(&reply)

		it.Then(t).Should(
			it.Nil(err),
			it.Equal(phase, thinker.AGENT_ASK),
		)
	})
}

//------------------------------------------------------------------------------

func mockOne(id, about string) *mock {
	return &mock{
		tools: []*mcp.Tool{
			{
				Name:        id,
				Description: about,
			},
		},
	}
}

func mockSeq(n int, id, about string) *mock {
	tools := make([]*mcp.Tool, n)
	for i := 0; i < n; i++ {
		tools[i] = &mcp.Tool{
			Name:        fmt.Sprintf("%s.%d", id, i),
			Description: fmt.Sprintf("%s %d", about, i),
		}
	}
	return &mock{
		tools: tools,
	}
}

func mockReply(id, about, reply string) *mock {
	return &mock{
		tools: []*mcp.Tool{
			{Name: id, Description: about},
		},
		returnVal: map[string]*mcp.CallToolResult{
			id: {
				Content: []mcp.Content{
					&mcp.TextContent{Text: reply},
				},
			},
		},
	}
}

// Mock MCP session for testing
type mock struct {
	tools     []*mcp.Tool
	returnVal map[string]*mcp.CallToolResult
	returnErr error
}

func (m *mock) ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{Tools: m.tools}, nil
}

func (m *mock) CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	if m.returnVal != nil {
		if result, ok := m.returnVal[params.Name]; ok {
			return result, nil
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "default result"},
		},
	}, nil
}

func (m *mock) Close() error { return nil }

// Helper to create a reply with tool calls
func replyOne(name string, args map[string]any) chatter.Reply {
	content := make([]chatter.Content, 1)

	argsJSON, _ := json.Marshal(args)
	content[0] = chatter.Invoke{
		Cmd:  name,
		Args: chatter.Json{Value: argsJSON},
	}

	return chatter.Reply{
		Stage:   chatter.LLM_INVOKE,
		Content: content,
	}
}
