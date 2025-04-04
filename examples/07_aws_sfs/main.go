//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/jsii-runtime-go"
	"github.com/fogfish/scud"
	"github.com/fogfish/typestep"
	"github.com/kshard/chatter/llm/bedrock"
	"github.com/kshard/thinker/examples/07_aws_sfs/core"
)

func main() {
	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("example-aws-thinker"), nil)

	llm := bedrock.NewMetaLlama31B70V1(stack)

	input := awsevents.NewEventBus(stack, jsii.String("Input"),
		&awsevents.EventBusProps{
			EventBusName: awscdk.Aws_STACK_NAME(),
		},
	)

	reply := awssqs.NewQueue(stack, jsii.String("Reply"),
		&awssqs.QueueProps{
			QueueName: awscdk.Aws_STACK_NAME(),
		},
	)

	ingest := scud.NewFunctionGo(stack, jsii.String("Ingestor"),
		&scud.FunctionGoProps{
			SourceCodeModule: "github.com/kshard/thinker/examples",
			SourceCodeLambda: "07_aws_sfs/cmd/ingest",
		},
	)
	llm.GrantAccessIn(ingest, jsii.String("us-west-2"))

	classify := scud.NewFunctionGo(stack, jsii.String("Classify"),
		&scud.FunctionGoProps{
			SourceCodeModule: "github.com/kshard/thinker/examples",
			SourceCodeLambda: "07_aws_sfs/cmd/classify",
		},
	)
	llm.GrantAccessIn(classify, jsii.String("us-west-2"))

	insight := scud.NewFunctionGo(stack, jsii.String("Insight"),
		&scud.FunctionGoProps{
			SourceCodeModule: "github.com/kshard/thinker/examples",
			SourceCodeLambda: "07_aws_sfs/cmd/insight",
		},
	)
	llm.GrantAccessIn(insight, jsii.String("us-west-2"))

	a := typestep.From[core.Document](input)
	b := typestep.Join(new(core.Ingestor).Ingest, ingest, a)
	c := typestep.Join(new(core.Classifier).Classify, classify, b)
	d := typestep.Join(new(core.Insighter).Insight, insight, c)
	f := typestep.ToQueue(reply, d)

	ts := typestep.NewTypeStep(stack, jsii.String("Agents"),
		&typestep.TypeStepProps{
			DeadLetterQueue: reply,
		},
	)
	typestep.StateMachine(ts, f)

	app.Synth(nil)
}
