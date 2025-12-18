package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/gateway/providers/anthropic/anthropic_responses"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type ClientOptions struct {
	// https://api.openai.com/v1
	BaseURL string
	ApiKey  string
	Headers map[string]string

	transport *http.Client
}

type Client struct {
	opts *ClientOptions
}

func NewClient(opts *ClientOptions) *Client {
	if opts.transport == nil {
		opts.transport = http.DefaultClient
	}

	if opts.BaseURL == "" {
		opts.BaseURL = "https://api.anthropic.com/v1"
	}

	return &Client{
		opts: opts,
	}
}

func (c *Client) NewResponses(ctx context.Context, inp *responses.Request) (*responses.Response, error) {
	anthropicRequest := anthropic_responses.NativeRequestToRequest(inp)

	payload, err := sonic.Marshal(anthropicRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.opts.BaseURL+"/messages", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", c.opts.ApiKey)
	req.Header.Set("Anthropic-Version", "2023-06-01")
	if anthropicRequest.OutputFormat != nil {
		req.Header.Set("Anthropic-Beta", "structured-outputs-2025-11-13")
	}
	for k, v := range c.opts.Headers {
		req.Header.Set(k, v)
	}

	res, err := c.opts.transport.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var anthropicResponse *anthropic_responses.Response
	err = utils.DecodeJSON(res.Body, &anthropicResponse)
	if err != nil {
		return nil, err
	}

	if anthropicResponse.Error != nil {
		return nil, errors.New(anthropicResponse.Error.Message)
	}

	return anthropicResponse.ToNativeResponse(), nil
}

func (c *Client) NewStreamingResponses(ctx context.Context, inp *responses.Request) (chan *responses.ResponseChunk, error) {
	anthropicRequest := anthropic_responses.NativeRequestToRequest(inp)

	payload, err := sonic.Marshal(anthropicRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.opts.BaseURL+"/messages", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.opts.ApiKey)
	req.Header.Set("Anthropic-Version", "2023-06-01")
	if anthropicRequest.OutputFormat != nil {
		req.Header.Set("Anthropic-Beta", "structured-outputs-2025-11-13")
	}
	for k, v := range c.opts.Headers {
		req.Header.Set(k, v)
	}

	res, err := c.opts.transport.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		var errResp map[string]any
		err = utils.DecodeJSON(res.Body, &errResp)
		return nil, errors.New(errResp["error"].(map[string]any)["message"].(string))
	}

	out := make(chan *responses.ResponseChunk)

	go func() {
		defer res.Body.Close()
		defer close(out)

		reader := bufio.NewReader(res.Body)
		converter := anthropic_responses.ResponseChunkToNativeResponseChunkConverter{}

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			line = strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(line, "data:") {
				anthropicResponseChunk := &anthropic_responses.ResponseChunk{}
				if err = sonic.Unmarshal([]byte(strings.TrimPrefix(line, "data:")), anthropicResponseChunk); err != nil {
					slog.WarnContext(ctx, "unable to unmarshal anthropic response chunk", slog.String("data", line), slog.Any("error", err))
					continue
				}

				for _, nativeChunk := range converter.ResponseChunkToNativeResponseChunk(anthropicResponseChunk) {
					out <- nativeChunk
				}
			}
		}
	}()

	return out, nil
}
