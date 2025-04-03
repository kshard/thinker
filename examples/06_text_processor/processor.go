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

	"github.com/fogfish/golem/pipe/v2"
	"github.com/fogfish/stream/lfs"
	"github.com/kshard/chatter"
	"github.com/kshard/chatter/llm/autoconfig"
	"github.com/kshard/thinker/agent"
	"github.com/kshard/thinker/codec"
	"github.com/kshard/thinker/command"
	"github.com/kshard/thinker/command/xfs"
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

func show(x string) string {
	fmt.Printf("==> %s\n", x)
	return x
}

func main() {
	llm, err := autoconfig.New("thinker")
	if err != nil {
		panic(err)
	}

	// In this example, we need to mount two file systems, containing input and
	// output data.
	in, err := lfs.New("/tmp/script/txt")
	if err != nil {
		panic(err)
	}
	to, err := lfs.New("/tmp/script/kwd")
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

	// Creating the FileSystem I/O utility
	rfs := xfs.New(in)
	wfs := xfs.New(to)

	// create worker to extract keywords from text files
	wrk := agent.NewPrompter(llm, processor)

	// Create processing pipeline
	fmt.Printf("==> processing files ...\n")
	ctx, cancel := context.WithCancel(context.Background())

	// 1. Walk over file system
	a, errA := rfs.Walk(ctx, "/", "")
	// 2. Print file name
	b := pipe.StdErr(pipe.Map(ctx, a, pipe.Pure(show)))
	// 3. Read the file
	c, errC := pipe.Map(ctx, b, pipe.Try(rfs.Read))
	// 4. Process the file with agent
	d, errD := pipe.Map(ctx, c, pipe.Try(xfs.Echo(wrk)))
	// 5. Write agents output to the new file, preserving the name
	e, errE := pipe.Map(ctx, d, pipe.Try(wfs.Create))
	// 6. Remove input file
	f, errF := pipe.Map(ctx, e, pipe.Try(rfs.Remove))

	<-pipe.Void(ctx, pipe.StdErr(f, pipe.Join(ctx, errA, errC, errD, errE, errF)))
	cancel()
}
