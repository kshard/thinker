package command

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Native interface {
	Bind(*mcp.Server)
	Spec() *mcp.Tool
}

func From[A, B any](spec *mcp.Tool, f mcp.ToolHandlerFor[A, B]) Native {
	return &native[A, B]{spec: spec, f: f}
}

type native[A, B any] struct {
	spec *mcp.Tool
	f    mcp.ToolHandlerFor[A, B]
}

func (n *native[A, B]) Bind(srv *mcp.Server) { mcp.AddTool(srv, n.spec, n.f) }
func (n *native[A, B]) Spec() *mcp.Tool      { return n.spec }

// Connect native function as a tool to the registry with the given id.
func (r *Registry) WithNative(f Native) *Registry {
	// TODO: implement connection closing

	spec := f.Spec()
	srv := mcp.NewServer(&mcp.Implementation{Name: spec.Name, Version: "v0.0.0"}, nil)
	f.Bind(srv)

	cli := mcp.NewClient(
		&mcp.Implementation{Name: "api_" + spec.Name, Version: "v0.0.0"},
		&mcp.ClientOptions{},
	)

	tcli, tsrv := mcp.NewInMemoryTransports()

	go srv.Run(context.Background(), tsrv)

	api, err := cli.Connect(context.Background(), tcli, nil)
	if err != nil {
		panic(err)
	}

	err = r.Attach(spec.Name, api)
	if err != nil {
		panic(err)
	}

	return r
}
