# Uno
This project provides primarily two things:
* [Golang SDK](https://github.com/curaious/uno?tab=readme-ov-file#golang-sdk) for building LLM powered apps.
    * [Make LLM calls](https://github.com/curaious/uno?tab=readme-ov-file#making-llm-calls) to various provide using an unified interface.
    * [Build agents](https://github.com/curaious/uno?tab=readme-ov-file#building-agents) with custom tools, mcp servers, memory, and durable execution.

* [AI Gateway](https://github.com/curaious/uno?tab=readme-ov-file#gateway-server) for managing access and rate limiting LLM calls with observability, and building and deploying agents without writing code with observability.

## Golang SDK

### Quickstart

**Add the SDK to your project**

```
go get -u github.com/curaious/uno
```

**Initialize the SDK**
```go
client, err := sdk.New(&sdk.ClientOptions{
	LLMConfigs: sdk.NewInMemoryConfigStore([]*gateway.ProviderConfig{
		{
			ProviderName:  llm.ProviderNameOpenAI,
			ApiKeys: []*gateway.APIKeyConfig{
				{
					Name:   "Key 1",
					APIKey: os.Getenv("OPENAI_API_KEY"),
				},
			},
		},
	}),
})
```

---

### Making LLM Calls

**Step 1: Create a model instance**
```go
// OpenAI
model := client.NewLLM(sdk.LLMOptions{
	Provider: llm.ProviderNameOpenAI,
	Model:    "gpt-4.1-mini",
})

// Anthropic
model := client.NewLLM(sdk.LLMOptions{
	Provider: llm.ProviderNameAnthropic,
	Model:    "claude-haiku-4-5",
})
```
**Step 2: Make the LLM Call**
```go
// Completions
resp, err := model.NewResponses(
	context.Background(),
	&responses.Request{
		Instructions: utils.Ptr("You are helpful assistant. You greet user with a light-joke"),
		Input: responses.InputUnion{
			OfString: utils.Ptr("Hello!"),
		},
	},
)

// Embeddings
resp, err := model.NewEmbedding(context.Background(), &embeddings.Request{
    Input: embeddings.InputUnion{
        OfString: utils.Ptr("The food was delicious and the waiter..."),
    },
})

// Text-to-spech
resp, err := model.NewSpeech(context.Background(), &speech.Request{
    Input: "Hello, this is a test of the text-to-speech system.",
    Model: "tts-1",
    Voice: "alloy",
})
```

**Refer to documentation for advanced usage:**
  * [Text Generation](https://curaious.techinscribed.com/uno-sdk/responses/text-generation)
  * [Tool Calling](https://curaious.techinscribed.com/uno-sdk/responses/tool-calling)
  * [Reasoning](https://curaious.techinscribed.com/uno-sdk/responses/reasoning)
  * [Structured Output](https://curaious.techinscribed.com/uno-sdk/responses/structured-output)
  * [Image Generation](https://curaious.techinscribed.com/uno-sdk/responses/image-generation)
  * [Web Search Tool](https://curaious.techinscribed.com/uno-sdk/responses/web-search-tool)
  * [Code Execution Tool](https://curaious.techinscribed.com/uno-sdk/responses/code-execution-tool)

---

## Building Agents
```go
agent := client.NewAgent(&sdk.AgentOptions{
    Name:        "Hello world agent",
    Instruction: client.Prompt("You are helpful assistant. You are interacting with the user named {{name}}"),
    LLM:         model,
    Parameters: responses.Parameters{
        Temperature: utils.Ptr(0.2),
    },
})

out, err := agent.Execute(context.Background(), &agents.AgentInput{
    Messages: []responses.InputMessageUnion{
        responses.UserMessage("Hello!"),
    },
})
```

**Refer to documentation for more advanced usage:**
  * [System Prompt](https://curaious.techinscribed.com/uno-sdk/agents/system-instruction)
  * [Function Tools](https://curaious.techinscribed.com/uno-sdk/agents/tools/function-tools)
  * [MCP Tools](https://curaious.techinscribed.com/uno-sdk/agents/tools/mcp-tools)
  * [Agent as a Tool](https://curaious.techinscribed.com/uno-sdk/agents/tools/agent-as-a-tool)
  * [Human in the loop](https://curaious.techinscribed.com/uno-sdk/agents/tools/human-in-the-loop)
  * [Conversation History](https://curaious.techinscribed.com/uno-sdk/agents/conversations/history)
  * [History Compaction or Summarization](https://curaious.techinscribed.com/uno-sdk/agents/conversations/summarization)
  * [Durable Execution via Restate](https://curaious.techinscribed.com/uno-sdk/agents/durable/restate)
  * [Durable Execution via Temporal](https://curaious.techinscribed.com/uno-sdk/agents/durable/temporal)
  * [Serving Agent through HTTP](https://curaious.techinscribed.com/uno-sdk/agents/serving-agents/serving-agents-http)

## AI Gateway

### Quickstart

**Prerequisite:** 
_Docker and Docker compose installed and running._
 
**Install AI Gateway and run locally**
```
npx -y @curaious/uno
```
then visit http://localhost:3000

**Refer to documentation for advanced usage:**
* LLM Gateway
  * [Configuring Providers](https://curaious.techinscribed.com/gateway/llm/providers)
  * [Managing Virtual Keys & Rate Limits](https://curaious.techinscribed.com/gateway/llm/virtual-keys)
  * [Using with OpenAI SDK](https://curaious.techinscribed.com/gateway/llm/sdk-integrations/openai)
  * [Observability](https://curaious.techinscribed.com/gateway/llm/tracing)
* No-code Agent Builder
  * [Choosing agent runtime](https://curaious.techinscribed.com/gateway/agent-builder/agent-runtime)
  * [Setting model parameters](https://curaious.techinscribed.com/gateway/agent-builder/model-configuration)
  * [Configure system prompt](https://curaious.techinscribed.com/gateway/agent-builder/system-prompt)
  * [Using built-in tools](https://curaious.techinscribed.com/gateway/agent-builder/built-in-tools)
  * [Connecting MCP servers](https://curaious.techinscribed.com/gateway/agent-builder/mcp-server)
  * [Structured Output](https://curaious.techinscribed.com/gateway/agent-builder/structured-output)
  * [Setup history & summarization](https://curaious.techinscribed.com/gateway/agent-builder/conversation-history)
  * [Versioning agents](https://curaious.techinscribed.com/gateway/agent-builder/versioning)
  * [Chat with the agent](https://curaious.techinscribed.com/gateway/agent-builder/conversing-with-the-agent)

## License

Apache 2.0
