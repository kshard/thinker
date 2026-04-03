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
	"github.com/fogfish/typestep"
	"github.com/kshard/thinker/examples/07_aws_sfs/core"
)

func main() {
	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("example-aws-thinker"), nil)

	// fm := iam.NewInferenceProfile(stack, jsii.String("InferenceProfile"),
	// 	jsii.String("us.anthropic.claude-3-7-sonnet-20250219-v1:0"),
	// )

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

	// ingest := typestep.NewFunctionTyped(stack, jsii.String("Ingestor"),
	// 	typestep.NewFunctionTypedProps(core.TODO, &scud.FunctionGoProps{
	// 		SourceCodeModule: "github.com/kshard/thinker/examples",
	// 		SourceCodeLambda: "07_aws_sfs/cmd/ingest",
	// 	}),
	// )
	// fm.GrantAccessIn(ingest, jsii.String("us-west-2"))

	// classify := typestep.NewFunctionTyped(stack, jsii.String("Classify"),
	// 	typestep.NewFunctionTypedProps(core.TODO, &scud.FunctionGoProps{
	// 		SourceCodeModule: "github.com/kshard/thinker/examples",
	// 		SourceCodeLambda: "07_aws_sfs/cmd/classify",
	// 	}),
	// )
	// fm.GrantAccessIn(classify, jsii.String("us-west-2"))

	// insight := typestep.NewFunctionTyped(stack, jsii.String("Insight"),
	// 	typestep.NewFunctionTypedProps(core.TODO, &scud.FunctionGoProps{
	// 		SourceCodeModule: "github.com/kshard/thinker/examples",
	// 		SourceCodeLambda: "07_aws_sfs/cmd/insight",
	// 	}),
	// )

	// fm.GrantAccessIn(insight, jsii.String("us-west-2"))

	a := typestep.From[core.Document](input)
	// b := typestep.Join(ingest, a)
	// c := typestep.Join(classify, b)
	// d := typestep.Join(insight, c)
	f := typestep.ToQueue(reply, a)

	ts := typestep.NewTypeStep(stack, jsii.String("Agents"),
		&typestep.TypeStepProps{
			DeadLetterQueue: reply,
		},
	)
	typestep.StateMachine(ts, f)

	app.Synth(nil)
}
