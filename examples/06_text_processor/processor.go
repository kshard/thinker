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
	"github.com/kshard/chatter"
	"github.com/kshard/chatter/llm/autoconfig"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/kshard/thinker/x/xfs"
)

func bootstrap(n int) (prompt chatter.Prompt, err error) {
	prompt.WithTask(`
		Use available tools to create %d files one by one with three or four lines of random
		but meanigful text. Each file must contain unique content.`, n)

	return
}

func processor(s string) (prompt chatter.Prompt, err error) {
	prompt.WithTask(`Analyze document and extract keywords.`)

	prompt.With(
		chatter.Blob("Document", s),
	)

	return
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

	// We need 10 files, let's use agents to get itls
	fmt.Printf("==> creating files ...\n")
	registry := command.NewRegistry()
	registry.Register(command.Bash("MacOS", "/tmp/script/txt"))
	init := agent.NewWorker(llm, 4, codec.FromEncoder(bootstrap), registry)
	if _, err = init.Prompt(context.Background(), 13); err != nil {
		panic(err)
	}

	// create worker to extract keywords from text files
	wrk := agent.NewPrompter(llm, processor)

	fmt.Printf("==> processing files ...\n")
	xfs.NewWorker(r, w).
		Walk(context.Background(), "/",
			func(ctx context.Context, w *xfs.Worker, path string) error {
				fmt.Printf("==> %s\n", path)

				txt, err := w.ReadFile(path)
				if err != nil {
					return err
				}

				kwd, err := wrk.PromptOnce(ctx, string(txt))
				if err != nil {
					return err
				}

				err = w.WriteFile(path, []byte(kwd))
				if err != nil {
					return err
				}

				err = w.Reader.Remove(path)
				if err != nil {
					return err
				}

				return nil
			},
		)
}
