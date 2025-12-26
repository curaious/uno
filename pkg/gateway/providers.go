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
	"github.com/praveen001/uno/pkg/gateway/providers/xai"
	"github.com/praveen001/uno/pkg/llm"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func (g *LLMGateway) getProvider(ctx context.Context, providerName llm.ProviderName, key string) (llm.Provider, error) {
	_, span := tracer.Start(ctx, "Gateway.GetProvider")
	defer span.End()

	hasConfigStore := g.ConfigStore != nil

	var directKey, baseUrl string
	var customHeaders map[string]string

	// If virtual key is provided, fetch the associated direct key
	if strings.HasPrefix(key, "sk-amg") {
		// When using virtual key, configStore is required
		if !hasConfigStore {
			err := errors.New("config store is required when virtual key is provided")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		virtualKey, err := g.ConfigStore.GetVirtualKey(key)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
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

		providerConfig, err := g.ConfigStore.GetProviderConfig(providerName)
		if err != nil {
			err = errors.New("failed to get provider config")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		if providerConfig != nil {
			baseUrl = providerConfig.BaseURL
			customHeaders = providerConfig.CustomHeaders

			// Todo: support for key rotation
			if len(providerConfig.ApiKeys) > 0 {
				directKey = providerConfig.ApiKeys[0].APIKey
			}
		}
	} else {
		directKey = key

		providerConfig, err := g.ConfigStore.GetProviderConfig(providerName)
		if err != nil {
			err = errors.New("failed to get provider config")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		if providerConfig != nil {
			baseUrl = providerConfig.BaseURL
			customHeaders = providerConfig.CustomHeaders
		}
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

	case llm.ProviderNameXAI:
		return xai.NewClient(&xai.ClientOptions{
			BaseURL: baseUrl,
			ApiKey:  directKey,
			Headers: customHeaders,
		}), nil
	case llm.ProviderNameOllama:
		return openai.NewClient(&openai.ClientOptions{
			BaseURL: baseUrl,
			ApiKey:  directKey,
			Headers: customHeaders,
		}), nil
	}

	return nil, fmt.Errorf("unknown provider: %s", providerName)
}
