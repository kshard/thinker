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
	"io"
	"os/exec"

	"github.com/fogfish/stream/lfs"
	"github.com/fogfish/stream/spool"
	"github.com/kshard/chatter"
	"github.com/kshard/chatter/provider/autoconfig"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func bootstrap(n int) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask(`
		Use available tools to create %d files one by one with three or four lines of random
		but meanigful text. Each file must contain unique content.`, n)

	return &prompt, nil
}

func processor(s string) (chatter.Message, error) {
	var prompt chatter.Prompt
	prompt.WithTask(`Analyze document and extract keywords.`)
	prompt.WithBlob("Document", s)

	return &prompt, nil
}

func main() {
	llm, err := autoconfig.FromNetRC("thinker")
	if err != nil {
		panic(err)
	}

	// In this example, we need to mount two file systems, containing input and
	// output data.
	r, err := lfs.New("/tmp/script/txt")
	if err != nil {
		panic(err)
	}
	w, err := lfs.New("/tmp/script/kwd")
	if err != nil {
		panic(err)
	}
	q := spool.New(r, w, spool.IsMutable)

	// We need 10 files, let's use agents to get itls
	fmt.Printf("==> creating files ...\n")
	registry := command.NewRegistry()
	registry.Attach("os", server())

	init := agent.NewManifold(llm, codec.FromEncoder(bootstrap), codec.DecoderString, registry)
	if _, err = init.Prompt(context.Background(), 13); err != nil {
		panic(err)
	}

	// create worker to extract keywords from text files
	wrk := agent.NewPrompter(llm, processor)

	fmt.Printf("==> processing files ...\n")
	q.ForEach(context.Background(), "/",
		func(ctx context.Context, path string, r io.Reader, w io.Writer) error {
			fmt.Printf("==> %v ...\n", path)

			txt, err := io.ReadAll(r)
			if err != nil {
				return err
			}

			kwd, err := wrk.PromptOnce(ctx, string(txt))
			if err != nil {
				return err
			}

			_, err = w.Write([]byte(kwd.String()))
			if err != nil {
				return err
			}

			return nil
		},
	)
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
	cmd.Dir = "/tmp/script/txt"
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, Reply{Output: "bash has failed with an error " + err.Error()}, nil
	}

	return nil, Reply{Output: stdout.String()}, nil
}
