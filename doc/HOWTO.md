# HowTos

- [HowTos](#howtos)
  - [Setup environment](#setup-environment)
    - [Example configurations](#example-configurations)


## Setup environment

The library uses [`chatter`](github.com/kshard/chatter) as an adapter to access LLMs from various providers. The first required things is to choose right provider for your application and configure it:
* [AWS Bedrock](https://aws.amazon.com/bedrock/)
* [OpenAI](https://platform.openai.com/docs/api-reference/introduction)
* [LM Studio](https://lmstudio.ai)  

Once the provider and LLMs access is configures, your application can start using it. The [`autoconfig`](github.com/kshard/chatter/provider/autoconfig) is recommended approach to begin with. It does not require any specification within Golang code.

```go
import "github.com/kshard/chatter/provider/autoconfig"

llm, err := autoconfig.FromNetRC("thinker")
if err != nil {
  panic(err)
}
```

The `autoconfig` reads the desired configuration from `~/.netrc` and creates appropriate instance of LLM API. Your `~/.netrc` file must include at least the `provider` and `model` fields under a named service entry. For example:

```
machine thinker
  provider provider:bedrock/foundation/converse
  model us.anthropic.claude-3-7-sonnet-20250219-v1:0
```

* `provider` specifies the full path to the provider's capability (e.g., `provider:bedrock/foundation/converse`). The path ressembles import path of providers implemented by this library
* `model` specifies the exact model name as recognized by the provider

Each provider and model family may support additional options. These can also be added under the same `machine` entry and will be passed into the corresponding provider implementation.

```
region     // used by Bedrock providers
host       // used by OpenAI providers
secret     // used by OpenAI providers
timeout    // used by OpenAI providers
dimensions // used by embedding families
```

### Example configurations 


**For AWS Bedrock**, `~/.netrc` config is
```
machine thinker
  provider provider:bedrock/foundation/converse
  model us.anthropic.claude-3-7-sonnet-20250219-v1:0
  region us-west-2
```

**For OpenAI**, `~/.netrc` config is
```
machine chatter1
  provider provider:openai/foundation/gpt
  model gpt-4o
  host https://api.openai.com
  secret sk-...IA
```

**For LM Studio**, `~/.netrc` config is
```
machine chatter1
  provider provider:openai/foundation/gpt
  model gemma-3-27b-it
  host http://localhost:1234
  timeout 300
```
