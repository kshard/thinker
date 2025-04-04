//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/kshard/chatter/llm/bedrock"
	"github.com/kshard/thinker/examples/07_aws_sfs/core"
)

func main() {
	llm, err := bedrock.New(
		bedrock.WithLLM(bedrock.LLAMA3_1_70B_INSTRUCT),
		bedrock.WithRegion("us-west-2"),
	)
	if err != nil {
		panic(err)
	}

	agt := core.NewIngestor(llm)

	lambda.Start(agt.Ingest)
}
