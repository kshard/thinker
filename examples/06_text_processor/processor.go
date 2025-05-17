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

	"github.com/fogfish/stream/lfs"
	"github.com/fogfish/stream/spool"
	"github.com/kshard/chatter"
	"github.com/kshard/chatter/llm/autoconfig"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/agent/worker"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command/softcmd"
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
	llm, err := autoconfig.New("thinker")
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
	q := spool.New(r, w, spool.Mutable)

	// We need 10 files, let's use agents to get itls
	fmt.Printf("==> creating files ...\n")
	registry := softcmd.NewRegistry()
	registry.Register(softcmd.Bash("MacOS", "/tmp/script/txt"))
	init := worker.NewReflex(llm, 4, codec.FromEncoder(bootstrap), registry)
	if _, err = init.Prompt(context.Background(), 13); err != nil {
		panic(err)
	}

	// create worker to extract keywords from text files
	wrk := agent.NewPrompter(llm, processor)

	fmt.Printf("==> processing files ...\n")
	q.ForEachFile(context.Background(), "/",
		func(ctx context.Context, path string, txt []byte) ([]byte, error) {
			fmt.Printf("==> %v ...\n", path)
			kwd, err := wrk.PromptOnce(ctx, string(txt))
			if err != nil {
				return nil, err
			}
			return []byte(kwd.String()), nil
		},
	)
}
