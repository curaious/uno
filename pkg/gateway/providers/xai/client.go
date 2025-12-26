package xai

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
	"github.com/praveen001/uno/internal/utils"
	"github.com/praveen001/uno/pkg/gateway/providers/openai/openai_responses"
	"github.com/praveen001/uno/pkg/gateway/providers/xai/xai_responses"
	"github.com/praveen001/uno/pkg/llm/responses"
)

type ClientOptions struct {
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
		opts.BaseURL = "https://api.x.ai/v1"
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
	openAiRequest := xai_responses.NativeRequestToRequest(inp)

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
		var errResp string
		err = utils.DecodeJSON(res.Body, &errResp)
		return nil, errors.New(errResp)
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
