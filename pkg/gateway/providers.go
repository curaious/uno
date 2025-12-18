package gateway

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/praveen001/uno/pkg/gateway/providers/anthropic"
	"github.com/praveen001/uno/pkg/gateway/providers/gemini"
	"github.com/praveen001/uno/pkg/gateway/providers/openai"
	"github.com/praveen001/uno/pkg/llm"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func (g *LLMGateway) getProvider(ctx context.Context, providerName llm.ProviderName, key string) (llm.Provider, error) {
	_, span := tracer.Start(ctx, "Gateway.GetProvider")
	defer span.End()

	isVirtualKey := strings.HasPrefix(key, "sk-amg")

	baseUrl := ""
	customHeaders := map[string]string{}
	directKey := key
	if isVirtualKey {
		// Get the virtual key and its configs
		virtualKey, err := g.ConfigStore.GetVirtualKey(key)
		if err != nil {
			return nil, err
		}
		span.SetAttributes(
			attribute.Int("virtual_key.allowed_providers", len(virtualKey.AllowedProviders)),
			attribute.Int("virtual_key.allowed_models", len(virtualKey.AllowedModels)),
		)

		// Check whether the given provider is allowed for the virtual key
		if len(virtualKey.AllowedProviders) > 0 && !slices.Contains(virtualKey.AllowedProviders, providerName) {
			err := errors.New("provider access denied by virtual key")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		// Check whether the model is allowed for the virtual key
		//if len(virtualKey.AllowedModels) > 0 && {
		//
		//}

		// Check rate limits
		// Check budget limits

		// Convert to direct key
		pc, keys, err := g.ConfigStore.GetProviderConfig(providerName)
		if err != nil {
			return nil, err
		}

		if pc != nil {
			baseUrl = pc.BaseURL
			customHeaders = pc.CustomHeaders
		}

		directKey = keys[0].APIKey
	}

	span.SetAttributes(attribute.String("base_url", baseUrl))

	switch providerName {
	case llm.ProviderNameOpenAI:
		return openai.NewClient(&openai.ClientOptions{
			BaseURL: baseUrl,
			ApiKey:  directKey,
			Headers: customHeaders,
		}), nil

	case llm.ProviderNameAnthropic:
		return anthropic.NewClient(&anthropic.ClientOptions{
			BaseURL: baseUrl,
			ApiKey:  directKey,
			Headers: customHeaders,
		}), nil

	case llm.ProviderNameGemini:
		return gemini.NewClient(&gemini.ClientOptions{
			BaseURL: baseUrl,
			ApiKey:  directKey,
			Headers: customHeaders,
		}), nil
	}

	return nil, fmt.Errorf("unknown provider: %s", providerName)
}
