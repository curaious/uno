package integrations

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/praveen001/uno/internal/utils"
)

type LangfuseClient struct {
	Endpoint string
	ApiKey   string
}

func NewLangfuseClient(endpoint, username, password string) *LangfuseClient {
	authKey := fmt.Sprintf("%s:%s", username, password)

	return &LangfuseClient{
		Endpoint: endpoint,
		ApiKey:   base64.StdEncoding.EncodeToString([]byte(authKey)),
	}
}

type LangfusePromptResponse struct {
	Id        string      `json:"id"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
	ProjectId string      `json:"projectId"`
	CreatedBy string      `json:"createdBy"`
	Prompt    string      `json:"prompt"`
	Name      string      `json:"name"`
	Version   int         `json:"version"`
	Type      string      `json:"type"`
	IsActive  interface{} `json:"isActive"`
	Config    struct {
	} `json:"config"`
	Tags            []interface{} `json:"tags"`
	Labels          []string      `json:"labels"`
	CommitMessage   interface{}   `json:"commitMessage"`
	ResolutionGraph interface{}   `json:"resolutionGraph"`
}

func (c *LangfuseClient) GetPrompt(name string, label string) (LangfusePromptResponse, error) {
	apiEndpoint := c.Endpoint + "/api/public/v2/prompts/" + name
	if label != "" {
		apiEndpoint += "?label=" + label
	}

	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return LangfusePromptResponse{}, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Basic "+c.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return LangfusePromptResponse{}, err
	}
	defer res.Body.Close()

	out := LangfusePromptResponse{}
	if err := utils.DecodeJSON(res.Body, &out); err != nil {
		return LangfusePromptResponse{}, err
	}

	return out, nil
}
