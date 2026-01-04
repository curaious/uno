package sdk

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/agent-framework/agents"
	"github.com/praveen001/uno/pkg/agent-framework/core"
	"github.com/praveen001/uno/pkg/agent-framework/mcpclient"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

type AgentOptions struct {
	Name        string
	LLM         llm.Provider
	Tools       []core.Tool
	Output      map[string]any
	History     core.ChatHistory
	Parameters  responses.Parameters
	Instruction core.SystemPromptProvider
	McpServers  []*mcpclient.MCPClient
}

func (c *SDK) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	agentName := r.URL.Query().Get("agent")
	if agentName == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	agent, exists := c.agents[agentName]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var payload agents.AgentInput
	if err := utils.DecodeJSON(r.Body, &payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	payload.Callback = func(chunk *responses.ResponseChunk) {
		buf, err := sonic.Marshal(chunk)
		if err != nil {
			return
		}

		_, _ = fmt.Fprintf(w, "event: %s\n", chunk.ChunkType())
		_, _ = fmt.Fprintf(w, "data: %s\n\n", buf)
		flusher.Flush()
	}

	_, err := agent.Execute(r.Context(), &payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (c *SDK) NewAgent(options *AgentOptions) *agents.Agent {
	agent := agents.NewAgent(&agents.AgentOptions{
		Name:        options.Name,
		LLM:         options.LLM,
		History:     options.History,
		Parameters:  options.Parameters,
		Output:      options.Output,
		Tools:       options.Tools,
		Instruction: options.Instruction,
		McpServers:  options.McpServers,
	})

	c.agents[options.Name] = agent

	return agent
}

func (c *SDK) NewRestateAgent(options *AgentOptions) *agents.Agent {
	agent := agents.NewAgent(&agents.AgentOptions{
		Name:        options.Name,
		LLM:         options.LLM,
		History:     options.History,
		Parameters:  options.Parameters,
		Output:      options.Output,
		Tools:       options.Tools,
		Instruction: options.Instruction,
		McpServers:  options.McpServers,
		Runtime:     agents.NewRestateRuntime(c.restateConfig.Endpoint),
	})

	c.agents[options.Name] = agent
	agents.RegisterAgent(options.Name, agent)

	return agent
}

func (c *SDK) StartRestateService(host, port string) {
	go func() {
		if err := server.NewRestate().
			Bind(restate.Reflect(agents.AgentWorkflow{})).
			Start(context.Background(), fmt.Sprintf("%s:%s", host, port)); err != nil {
			log.Fatal(err)
		}
	}()
}
