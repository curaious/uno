package sdk

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/agent-framework/agents"
	"github.com/curaious/uno/pkg/agent-framework/core"
	"github.com/curaious/uno/pkg/agent-framework/history"
	"github.com/curaious/uno/pkg/agent-framework/mcpclient"
	"github.com/curaious/uno/pkg/agent-framework/runtime/restate_runtime"
	"github.com/curaious/uno/pkg/agent-framework/runtime/temporal_runtime"
	"github.com/curaious/uno/pkg/agent-framework/streaming"
	"github.com/curaious/uno/pkg/llm"
	"github.com/curaious/uno/pkg/llm/responses"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type AgentOptions struct {
	Name        string
	LLM         llm.Provider
	Tools       []core.Tool
	Output      map[string]any
	History     *history.CommonConversationManager
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
		Runtime:     restate_runtime.NewRestateRuntime(c.restateConfig.Endpoint),
	})

	streamHandler, err := streaming.NewRedisStreamBroker(streaming.RedisStreamBrokerOptions{
		Addr: "localhost:6379",
	})
	if err != nil {
		log.Fatal(err)
	}

	agent.SetStreamBroker(streamHandler)

	c.agents[options.Name] = agent

	return agent
}

func (c *SDK) StartRestateService(host, port string) {
	for _, agent := range c.agents {
		agents.RegisterAgent(agent.Name(), agent)
	}

	go func() {
		if err := server.NewRestate().
			Bind(restate.Reflect(restate_runtime.AgentWorkflow{})).
			Start(context.Background(), fmt.Sprintf("%s:%s", host, port)); err != nil {
			log.Fatal(err)
		}
	}()
}

func (c *SDK) NewTemporalAgent(options *AgentOptions) *agents.Agent {
	agent := agents.NewAgent(&agents.AgentOptions{
		Name:        options.Name,
		LLM:         options.LLM,
		History:     options.History,
		Parameters:  options.Parameters,
		Output:      options.Output,
		Tools:       options.Tools,
		Instruction: options.Instruction,
		McpServers:  options.McpServers,
		Runtime:     temporal_runtime.NewTemporalRuntime(c.temporalConfig.Endpoint),
	})

	streamHandler, err := streaming.NewRedisStreamBroker(streaming.RedisStreamBrokerOptions{
		Addr: "localhost:6379",
	})
	if err != nil {
		log.Fatal(err)
	}

	agent.SetStreamBroker(streamHandler)

	c.agents[options.Name] = agent

	// Wrap the agent IO to call temporal activities
	c.temporalAgentConfigs[options.Name] = options

	return agent
}

func (c *SDK) StartTemporalService() {
	cli, err := client.Dial(client.Options{
		HostPort: c.temporalConfig.Endpoint,
	})
	if err != nil {
		panic("unable to create temporal client")
	}

	go func() {
		w := worker.New(cli, "AgentWorkflowTaskQueue", worker.Options{})

		// Register workflows and activities based on the agents available in the SDK
		for agentName := range c.temporalAgentConfigs {
			agent := c.agents[agentName]
			temporalAgent := temporal_runtime.NewTemporalAgent(agent)

			w.RegisterActivityWithOptions(temporalAgent.LoadMessages, activity.RegisterOptions{Name: agentName + "_LoadMessagesActivity"})
			w.RegisterActivityWithOptions(temporalAgent.SaveMessages, activity.RegisterOptions{Name: agentName + "_SaveMessagesActivity"})
			w.RegisterActivityWithOptions(temporalAgent.SaveSummary, activity.RegisterOptions{Name: agentName + "_SaveSummaryActivity"})
			w.RegisterActivityWithOptions(temporalAgent.GetPrompt, activity.RegisterOptions{Name: agentName + "_GetPromptActivity"})
			w.RegisterActivityWithOptions(temporalAgent.NewStreamingResponses, activity.RegisterOptions{Name: agentName + "_NewStreamingResponsesActivity"})
			w.RegisterActivityWithOptions(temporalAgent.CallTool, activity.RegisterOptions{Name: agentName + "_CallToolActivity"})
			w.RegisterActivityWithOptions(temporalAgent.RunCreated, activity.RegisterOptions{Name: agentName + "_RunCreatedActivity"})
			w.RegisterActivityWithOptions(temporalAgent.RunPaused, activity.RegisterOptions{Name: agentName + "_RunPausedActivity"})
			w.RegisterActivityWithOptions(temporalAgent.RunCompleted, activity.RegisterOptions{Name: agentName + "_RunCompletedActivity"})

			w.RegisterWorkflowWithOptions(temporalAgent.Execute, workflow.RegisterOptions{
				Name: agentName + "_AgentWorkflow",
			})
		}

		err = w.Run(worker.InterruptCh())
		if err != nil {
			log.Fatal(err)
		}
	}()
}
