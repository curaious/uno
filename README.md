# Uno

A high-performance LLM Gateway and Agent Framework written in Go.

Uno provides a unified interface for interacting with LLMs from OpenAI, Anthropic, Gemini, xAI, and Ollama. Use it as a standalone gateway with virtual keys and observability, or embed the SDK directly in your Go applications.

## Features

- **Unified API** — Single request/response format across all providers
- **Virtual Keys** — Protect provider API keys with Uno-generated keys
- **Observability** — Built-in tracing with OpenTelemetry and ClickHouse
- **Agent Framework** — Build agents with tool calling, MCP integration, and conversation history
- **Durable Execution** — Run agents with [Restate](https://restate.dev) for fault-tolerant workflows

## Quickstart

### Gateway Mode

Start the gateway with Docker:

```bash
npx @curaious/uno
```

Open `http://localhost:3000` to configure providers and create virtual keys.

Point your existing SDK to the gateway:

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:6060/api/gateway/openai",
    api_key="your-virtual-key",
)

response = client.responses.create(
    model="gpt-4.1-mini",
    input="Hello!",
)
```

### SDK Mode

Install:

```bash
go get -u github.com/curaious/uno
```

Use the SDK directly:

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/curaious/uno/pkg/gateway"
    "github.com/curaious/uno/pkg/llm"
    "github.com/curaious/uno/pkg/llm/responses"
    "github.com/curaious/uno/pkg/sdk"
    "github.com/curaious/uno/internal/utils"
)

func main() {
    client, _ := sdk.New(&sdk.ClientOptions{
        LLMConfigs: sdk.NewInMemoryConfigStore([]*gateway.ProviderConfig{
            {
                ProviderName: llm.ProviderNameOpenAI,
                ApiKeys: []*gateway.APIKeyConfig{
                    {Name: "default", APIKey: os.Getenv("OPENAI_API_KEY")},
                },
            },
        }),
    })

    model := client.NewLLM(sdk.LLMOptions{
        Provider: llm.ProviderNameOpenAI,
        Model:    "gpt-4.1-mini",
    })

    resp, _ := model.NewResponses(context.Background(), &responses.Request{
        Input: responses.InputUnion{
            OfString: utils.Ptr("What is the capital of France?"),
        },
    })

    fmt.Println(resp.Output[0].OfOutputMessage.Content[0].OfOutputText.Text)
}
```

## Provider Support

| Provider | Text | Image Gen | Image Input | Tool Calls | Reasoning | Streaming | Structured Output | Embeddings |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| OpenAI | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Anthropic | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| Gemini | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| xAI | ✅ | ❌ | ❌ | ✅ | ❌ | ✅ | ✅ | ❌ |
| Ollama | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ |

## SDK Capabilities

### Responses API

Generate text, images, and structured output:

```go
resp, _ := model.NewResponses(ctx, &responses.Request{
    Instructions: utils.Ptr("You are a helpful assistant."),
    Input:        responses.InputUnion{OfString: utils.Ptr("Hello!")},
})
```

### Agents

Build agents with tools and conversation memory:

```go
agent := client.NewAgent(&sdk.AgentOptions{
    Name:        "Assistant",
    Instruction: client.Prompt("You are a helpful assistant."),
    LLM:         client.NewLLM(sdk.LLMOptions{Provider: llm.ProviderNameOpenAI, Model: "gpt-4o"}),
    Tools:       []core.Tool{weatherTool, searchTool},
})

output, _ := agent.Execute(ctx, []responses.InputMessageUnion{
    responses.UserMessage("What's the weather in Tokyo?"),
}, callback)
```

### MCP Tools

Connect to MCP servers:

```go
agent := client.NewAgent(&sdk.AgentOptions{
    Name:        "MCP Agent",
    Instruction: client.Prompt("You have access to external tools."),
    LLM:         model,
    MCPServers: []*core.MCPServer{
        {Name: "filesystem", Command: "npx", Args: []string{"-y", "@anthropic/mcp-filesystem"}},
    },
})
```

### Embeddings

Generate text embeddings:

```go
resp, _ := model.NewEmbedding(ctx, &embeddings.Request{
    Input: embeddings.InputUnion{
        OfString: utils.Ptr("The food was delicious"),
    },
})
```

## Examples

See the [`examples/`](./examples) directory:

| Example | Description |
| :--- | :--- |
| `1_text_generation` | Basic text generation with streaming |
| `2_tool_calling` | Function calling with LLMs |
| `3_reasoning` | Chain-of-thought reasoning |
| `4_image_processing` | Image input processing |
| `5_image_generation` | Generate images with DALL-E/Imagen |
| `6_simple_agent` | Basic agent setup |
| `7_tool_calling_agent` | Agent with function tools |
| `8_agent_multi_turn_conversation` | Multi-turn conversations |
| `9_agent_with_mcp_tools` | MCP server integration |
| `10_agent_as_a_tool` | Compose agents as tools |
| `11_human_in_the_loop` | Human approval workflows |
| `12_embeddings` | Text embeddings |

## Documentation

Full documentation: [docs](./docs)

## License

Apache 2.0

