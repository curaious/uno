package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/curaious/uno/pkg/sandbox"
)

// Client talks to a sandbox daemon inside a sandbox pod.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient constructs a client for the given sandbox handle.
func NewClient(handle *sandbox.SandboxHandle) *Client {
	base := fmt.Sprintf("http://%s:%d", handle.PodIP, 8080)
	return &Client{
		baseURL: base,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// RunBashCommand executes a bash command inside the sandbox.
func (c *Client) RunBashCommand(ctx context.Context, in *ExecRequest) (*ExecResponse, error) {
	var res ExecResponse
	if err := c.doJSON(ctx, http.MethodPost, "/exec/bash", in, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// RunPythonScript executes a Python script inside the sandbox.
func (c *Client) RunPythonScript(ctx context.Context, in *ExecRequest) (*ExecResponse, error) {
	var res ExecResponse
	if err := c.doJSON(ctx, http.MethodPost, "/exec/python", in, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ReadFile reads a file from the sandbox filesystem.
func (c *Client) ReadFile(ctx context.Context, filePath string) (*fileContent, error) {
	var out fileContent
	if err := c.doJSON(ctx, http.MethodGet, "/files/"+url.PathEscape(filePath), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// WriteFile writes content to a file in the sandbox filesystem.
func (c *Client) WriteFile(ctx context.Context, filePath, content string) (*fileContent, error) {
	in := fileContent{Path: filePath, Content: content}
	var out fileContent
	if err := c.doJSON(ctx, http.MethodPost, "/files/"+url.PathEscape(filePath), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteFile deletes a file from the sandbox filesystem.
func (c *Client) DeleteFile(ctx context.Context, filePath string) error {
	return c.doJSON(ctx, http.MethodDelete, "/files/"+url.PathEscape(filePath), nil, nil)
}

// doJSON sends a JSON request and decodes a JSON response (if out is non-nil).
func (c *Client) doJSON(ctx context.Context, method, p string, in any, out any) error {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, p)

	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sandbox error: status=%d body=%s", resp.StatusCode, string(b))
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
