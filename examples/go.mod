module github.com/kshard/thinker/examples

go 1.25.0

replace github.com/kshard/thinker => ../

require (
	github.com/aws/aws-cdk-go/awscdk/v2 v2.248.0
	github.com/aws/aws-lambda-go v1.54.0
	github.com/aws/jsii-runtime-go v1.127.0
	github.com/fogfish/scud v0.13.1
	github.com/fogfish/stream v1.3.6
	github.com/fogfish/typestep v0.0.6
	github.com/kshard/chatter v0.11.2
	github.com/kshard/chatter/provider/autoconfig v0.13.0
	github.com/kshard/chatter/provider/bedrock v0.10.1
	github.com/kshard/thinker v0.0.0-00010101000000-000000000000
	github.com/modelcontextprotocol/go-sdk v1.4.1
)

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.18.2 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/ajg/form v1.7.1 // indirect
	github.com/aws/aws-sdk-go-v2 v1.41.5 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.8 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.14 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.14 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.21 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.22.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.6 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/bedrock v1.58.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.50.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.98.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.10 // indirect
	github.com/aws/constructs-go/constructs/v10 v10.6.0 // indirect
	github.com/aws/smithy-go v1.24.3 // indirect
	github.com/cdklabs/awscdk-asset-awscli-go/awscliv1/v2 v2.2.273 // indirect
	github.com/cdklabs/awscdk-asset-node-proxy-agent-go/nodeproxyagentv6/v2 v2.1.1 // indirect
	github.com/cdklabs/cloud-assembly-schema-go/awscdkcloudassemblyschema/v53 v53.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fogfish/faults v0.3.2 // indirect
	github.com/fogfish/golem/duct v0.0.1 // indirect
	github.com/fogfish/golem/hseq v1.3.0 // indirect
	github.com/fogfish/golem/optics v0.14.0 // indirect
	github.com/fogfish/guid/v2 v2.1.0 // indirect
	github.com/fogfish/gurl/v2 v2.10.0 // indirect
	github.com/fogfish/logger/v3 v3.2.1 // indirect
	github.com/fogfish/logger/x/xlog v0.0.1 // indirect
	github.com/fogfish/opts v0.0.5 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.14 // indirect
	github.com/googleapis/gax-go/v2 v2.19.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jdxcode/netrc v1.0.0 // indirect
	github.com/kshard/chatter/provider/google v0.1.2 // indirect
	github.com/kshard/chatter/provider/openai v0.10.1 // indirect
	github.com/kshard/float8 v0.0.3 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.67.0 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/metric v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/lint v0.0.0-20241112194109-818c5a804067 // indirect
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/telemetry v0.0.0-20260311193753-579e4da9a98c // indirect
	golang.org/x/text v0.35.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.43.0 // indirect
	golang.org/x/tools/cmd/godoc v0.1.0-deprecated // indirect
	golang.org/x/tools/godoc v0.1.0-deprecated // indirect
	google.golang.org/genai v1.51.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260316180232-0b37fe3546d5 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
