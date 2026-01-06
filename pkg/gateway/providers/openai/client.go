package openai

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/gateway/providers/base"
	"github.com/curaious/uno/pkg/gateway/providers/openai/openai_chat_completion"
	"github.com/curaious/uno/pkg/gateway/providers/openai/openai_embeddings"
	"github.com/curaious/uno/pkg/gateway/providers/openai/openai_responses"
	"github.com/curaious/uno/pkg/llm/chat_completion"
	"github.com/curaious/uno/pkg/llm/embeddings"
	"github.com/curaious/uno/pkg/llm/responses"
)

type ClientOptions struct {
	// https://api.openai.com/v1
	BaseURL string
	ApiKey  string
	Headers map[string]string

	transport *http.Client
}

type Client struct {
	*base.BaseProvider
	opts *ClientOptions
}

func NewClient(opts *ClientOptions) *Client {
	if opts.transport == nil {
		opts.transport = http.DefaultClient
	}

	if opts.BaseURL == "" {
		opts.BaseURL = "https://api.openai.com/v1"
	}

	return &Client{
		opts: opts,
	}
}

func (c *Client) NewResponses(ctx context.Context, inp *responses.Request) (*responses.Response, error) {
	openAiRequest := openai_responses.NativeRequestToRequest(inp)

	payload, err := sonic.Marshal(openAiRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.opts.BaseURL+"/responses", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.opts.ApiKey)

	res, err := c.opts.transport.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var openAiResponse *openai_responses.Response
	err = utils.DecodeJSON(res.Body, &openAiResponse)
	if err != nil {
		return nil, err
	}

	if openAiResponse.Error != nil {
		return nil, errors.New(openAiResponse.Error.Message)
	}

	return openAiResponse.ToNativeResponse(), nil
}

func (c *Client) NewStreamingResponses(ctx context.Context, inp *responses.Request) (chan *responses.ResponseChunk, error) {
	openAiRequest := openai_responses.NativeRequestToRequest(inp)

	payload, err := sonic.Marshal(openAiRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.opts.BaseURL+"/responses", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.opts.ApiKey)

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

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			line = strings.TrimRight(line, "\r\n")
			fmt.Println(line)
			if strings.HasPrefix(line, "data:") {
				openAiResponseChunk := &openai_responses.ResponseChunk{}
				err = sonic.Unmarshal([]byte(strings.TrimPrefix(line, "data:")), openAiResponseChunk)
				if err != nil {
					slog.WarnContext(ctx, "unable to unmarshal openai response chunk", slog.String("data", line), slog.Any("error", err))
					continue
				}
				out <- openAiResponseChunk.ToNativeResponseChunk()
			}
		}
	}()

	return out, nil
}

func (c *Client) NewEmbedding(ctx context.Context, inp *embeddings.Request) (*embeddings.Response, error) {
	openAiRequest := openai_embeddings.NativeRequestToRequest(inp)

	payload, err := sonic.Marshal(openAiRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.opts.BaseURL+"/embeddings", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.opts.ApiKey)

	res, err := c.opts.transport.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var errResp map[string]any
		err = utils.DecodeJSON(res.Body, &errResp)
		return nil, errors.New(errResp["error"].(map[string]any)["message"].(string))
	}

	var openAiResponse *openai_embeddings.Response
	err = utils.DecodeJSON(res.Body, &openAiResponse)
	if err != nil {
		return nil, err
	}

	return openAiResponse.ToNativeResponse(), nil
}

func (c *Client) NewChatCompletion(ctx context.Context, inp *chat_completion.Request) (*chat_completion.Response, error) {
	openAiRequest := openai_chat_completion.NativeRequestToRequest(inp)

	payload, err := sonic.Marshal(openAiRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.opts.BaseURL+"/chat/completions", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.opts.ApiKey)

	res, err := c.opts.transport.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var errResp map[string]any
		err = utils.DecodeJSON(res.Body, &errResp)
		if err != nil {
			return nil, err
		}
		if errorObj, ok := errResp["error"].(map[string]any); ok {
			if message, ok := errorObj["message"].(string); ok {
				return nil, errors.New(message)
			}
		}
		return nil, errors.New("unknown error occurred")
	}

	var openAiResponse *openai_chat_completion.Response
	err = utils.DecodeJSON(res.Body, &openAiResponse)
	if err != nil {
		return nil, err
	}

	return openAiResponse.ToNativeResponse(), nil
}

func (c *Client) NewStreamingChatCompletion(ctx context.Context, inp *chat_completion.Request) (chan *chat_completion.ResponseChunk, error) {
	openAiRequest := openai_chat_completion.NativeRequestToRequest(inp)

	payload, err := sonic.Marshal(openAiRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.opts.BaseURL+"/chat/completions", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.opts.ApiKey)

	res, err := c.opts.transport.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		var errResp map[string]any
		err = utils.DecodeJSON(res.Body, &errResp)
		if err != nil {
			return nil, err
		}
		if errorObj, ok := errResp["error"].(map[string]any); ok {
			if message, ok := errorObj["message"].(string); ok {
				return nil, errors.New(message)
			}
		}
		return nil, errors.New("unknown error occurred")
	}

	out := make(chan *chat_completion.ResponseChunk)

	go func() {
		defer res.Body.Close()
		defer close(out)
		reader := bufio.NewReader(res.Body)

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			line = strings.TrimRight(line, "\r\n")
			fmt.Println(line)

			if line == "data: [DONE]" {
				return
			}

			if strings.HasPrefix(line, "data:") {
				openAiChatCompletionChunk := &openai_chat_completion.ResponseChunk{}
				err = sonic.Unmarshal([]byte(strings.TrimPrefix(line, "data:")), openAiChatCompletionChunk)
				if err != nil {
					slog.WarnContext(ctx, "unable to unmarshal chat completion response chunk", slog.String("data", line), slog.Any("error", err))
					continue
				}
				out <- openAiChatCompletionChunk.ToNativeResponseChunk()
			}
		}
	}()

	return out, nil
}
