# HowTos

- [HowTos](#howtos)
  - [Setup environment](#setup-environment)


## Setup environment

The library uses [`chatter`](github.com/kshard/chatter) as an adapter to access LLMs from various providers. The first required things is to choose right provider for your application and configure it:
* [AWS Bedrock](https://aws.amazon.com/bedrock/)
* [OpenAI](https://platform.openai.com/docs/api-reference/introduction)
* [LM Studio](https://lmstudio.ai)  

Once the provider and LLMs access is configures, your application can start using it. The [`autoconfig`](github.com/kshard/chatter/llm/autoconfig) is recommended approach to begin with. It does not require any specification within Golang code.

```go
import "github.com/kshard/chatter/llm/autoconfig"

llm, err := autoconfig.New("thinker")
if err != nil {
  panic(err)
}
```

The `autoconfig` reads the desired configuration from `~/.netrc` and creates appropriate instance of LLM API.

**For AWS Bedrock**, `~/.netrc` config is
```
machine thinker
  provider bedrock
  region us-west-2
  family llama3
  model meta.llama3-1-70b-instruct-v1:0
```

**For OpenAI**, `~/.netrc` config is
```
machine chatter1
  provider openai
  host https://api.openai.com
  model gpt-4o
  secret sk-...IA
```

**For LM Studio**, `~/.netrc` config is
```
machine chatter1
  provider openai
  host http://localhost:1234
  model gemma-3-27b-it
  secret sk-...IA
```
