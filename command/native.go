//
// Copyright (C) 2025 - 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

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
func (r *Registry) WithNative(id string, fs ...Native) *Registry {
	// TODO: implement connection closing

	srv := mcp.NewServer(&mcp.Implementation{Name: id, Version: "v0.0.0"}, nil)
	for _, f := range fs {
		f.Bind(srv)
	}

	cli := mcp.NewClient(
		&mcp.Implementation{Name: "api_" + id, Version: "v0.0.0"},
		&mcp.ClientOptions{},
	)

	tcli, tsrv := mcp.NewInMemoryTransports()

	go srv.Run(context.Background(), tsrv)

	api, err := cli.Connect(context.Background(), tcli, nil)
	if err != nil {
		panic(err)
	}

	err = r.Attach(id, api)
	if err != nil {
		panic(err)
	}

	return r
}
