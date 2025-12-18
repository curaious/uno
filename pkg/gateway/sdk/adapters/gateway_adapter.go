package adapters

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/gateway"
	"github.com/praveen001/uno/pkg/llm"
	"github.com/praveen001/uno/pkg/llm/responses"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// ExternalLLMGateway calls the agent-server's gateway API via HTTP.
// Use this when you're an SDK consumer calling the agent-server remotely.
type ExternalLLMGateway struct {
	endpoint   string
	virtualKey string
	httpClient *http.Client
}

// NewExternalLLMGateway creates a provider that calls agent-server via HTTP.
func NewExternalLLMGateway(endpoint, virtualKey string) *ExternalLLMGateway {
	return &ExternalLLMGateway{
		endpoint:   strings.TrimSuffix(endpoint, "/"),
		virtualKey: virtualKey,
		httpClient: &http.Client{},
	}
}

func (p *ExternalLLMGateway) NewResponses(ctx context.Context, providerName llm.ProviderName, req *responses.Request) (*responses.Response, error) {
	// Prepend provider to model for gateway routing
	originalModel := req.Model
	req.Model = fmt.Sprintf("%s:%s", providerName, req.Model)
	defer func() { req.Model = originalModel }()

	payload, err := sonic.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint+"/api/gateway/responses", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		httpReq.Header.Add(k, v)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-virtual-key", p.virtualKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		_ = utils.DecodeJSON(resp.Body, &errResp)
		return nil, fmt.Errorf("gateway error (status %d): %v", resp.StatusCode, errResp)
	}

	var nativeResp responses.Response
	if err := utils.DecodeJSON(resp.Body, &nativeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &nativeResp, nil
}

func (p *ExternalLLMGateway) NewStreamingResponses(ctx context.Context, providerName llm.ProviderName, req *responses.Request) (chan *responses.ResponseChunk, error) {
	// Prepend provider to model for gateway routing
	originalModel := req.Model
	req.Model = fmt.Sprintf("%s:%s", providerName, req.Model)

	stream := true
	req.Stream = &stream

	payload, err := sonic.Marshal(req)
	if err != nil {
		req.Model = originalModel
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req.Model = originalModel

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint+"/api/gateway/responses", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		httpReq.Header.Add(k, v)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-virtual-key", p.virtualKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var errResp map[string]any
		_ = utils.DecodeJSON(resp.Body, &errResp)
		return nil, fmt.Errorf("gateway error (status %d): %v", resp.StatusCode, errResp)
	}

	out := make(chan *responses.ResponseChunk)
	go func() {
		defer resp.Body.Close()
		defer close(out)

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			line = strings.TrimRight(line, "\r\n")
			if !strings.HasPrefix(line, "data:") {
				continue
			}

			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)
			if data == "" || data == "[DONE]" {
				continue
			}

			var chunk responses.ResponseChunk
			if err := sonic.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			out <- &chunk
		}
	}()

	return out, nil
}

// LocalLLMGateway uses the internal LLMGateway for server-side use.
// This is used within the agent-server where we have direct access to services.
// It handles virtual key resolution and provider configuration from the database.
type LocalLLMGateway struct {
	gateway *gateway.LLMGateway
}

// NewLocalLLMGateway creates a provider using the internal gateway.
// The key can be a virtual key (sk-amg-xxx) which will be resolved to actual API keys,
// or a direct API key for the provider.
func NewLocalLLMGateway(gw *gateway.LLMGateway) *LocalLLMGateway {
	return &LocalLLMGateway{
		gateway: gw,
	}
}

func (p *LocalLLMGateway) NewResponses(ctx context.Context, providerName llm.ProviderName, req *responses.Request) (*responses.Response, error) {
	llmReq := &llm.Request{
		OfResponsesInput: req,
	}

	resp, err := p.gateway.HandleRequest(ctx, providerName, p.getKey(providerName), llmReq)
	if err != nil {
		return nil, err
	}

	return resp.OfResponsesOutput, nil
}

func (p *LocalLLMGateway) NewStreamingResponses(ctx context.Context, providerName llm.ProviderName, req *responses.Request) (chan *responses.ResponseChunk, error) {
	llmReq := &llm.Request{
		OfResponsesInput: req,
	}

	streamResp, err := p.gateway.HandleStreamingRequest(ctx, providerName, p.getKey(providerName), llmReq)
	if err != nil {
		return nil, err
	}

	return streamResp.ResponsesStreamData, nil
}

func (p *LocalLLMGateway) getKey(providerName llm.ProviderName) string {
	_, keys, err := p.gateway.ConfigStore.GetProviderConfig(providerName)
	if err != nil {
		return ""
	}

	if len(keys) == 0 {
		return ""
	}

	return keys[0].APIKey
}
