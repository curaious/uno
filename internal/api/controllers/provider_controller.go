package controllers

import (
	"errors"

	"github.com/curaious/uno/internal/services"
	provider2 "github.com/curaious/uno/internal/services/provider"
	"github.com/curaious/uno/pkg/llm"
	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"

	"github.com/curaious/uno/internal/perrors"
)

func RegisterProviderRoutes(r *router.Router, svc *services.Services) {
	// Get provider models
	r.GET("/api/agent-server/providers/models", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		modelsResponse := provider2.GetProviderModelsResponse()
		writeOK(ctx, stdCtx, "Provider models retrieved successfully", modelsResponse)
	})

	// Create API key
	r.POST("/api/agent-server/api-keys", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		var body provider2.CreateAPIKeyRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.ProviderType == "" {
			writeError(ctx, stdCtx, "Provider type is required", perrors.NewErrInvalidRequest("Provider type is required", errors.New("provider_type is required")))
			return
		}

		if !body.ProviderType.IsValid() {
			writeError(ctx, stdCtx, "Invalid provider type", perrors.NewErrInvalidRequest("Invalid provider type", errors.New("provider_type must be one of: OpenAI, Anthropic, Gemini, xAI")))
			return
		}

		if body.Name == "" {
			writeError(ctx, stdCtx, "Name is required", perrors.NewErrInvalidRequest("Name is required", errors.New("name is required")))
			return
		}

		if body.APIKey == "" {
			writeError(ctx, stdCtx, "API key is required", perrors.NewErrInvalidRequest("API key is required", errors.New("api_key is required")))
			return
		}

		apiKey, err := svc.Provider.Create(stdCtx, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create API key", perrors.NewErrInternalServerError("Failed to create API key", err))
			return
		}

		writeOK(ctx, stdCtx, "API key created successfully", apiKey)
	})

	// List API keys
	r.GET("/api/agent-server/api-keys", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		providerParam := string(ctx.QueryArgs().Peek("provider"))
		enabledOnly := ctx.QueryArgs().GetBool("enabled_only")

		var providerType *llm.ProviderName
		if providerParam != "" {
			p := llm.ProviderName(providerParam)
			if !p.IsValid() {
				writeError(ctx, stdCtx, "Invalid provider type", perrors.NewErrInvalidRequest("Invalid provider type", errors.New("provider must be one of: OpenAI, Anthropic, Gemini, xAI")))
				return
			}
			providerType = &p
		}

		apiKeys, err := svc.Provider.List(stdCtx, providerType, enabledOnly)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list API keys", perrors.NewErrInternalServerError("Failed to list API keys", err))
			return
		}

		writeOK(ctx, stdCtx, "API keys retrieved successfully", apiKeys)
	})

	// Update API key
	r.PUT("/api/agent-server/api-keys/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		idRaw, err := pathParam(ctx, "id")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		id, err := uuid.Parse(idRaw)
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		var body provider2.UpdateAPIKeyRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.APIKey != nil && *body.APIKey == "" {
			writeError(ctx, stdCtx, "API key cannot be empty", perrors.NewErrInvalidRequest("API key cannot be empty", errors.New("api_key cannot be empty")))
			return
		}

		if body.Name != nil && *body.Name == "" {
			writeError(ctx, stdCtx, "Name cannot be empty", perrors.NewErrInvalidRequest("Name cannot be empty", errors.New("name cannot be empty")))
			return
		}

		apiKey, err := svc.Provider.Update(stdCtx, id, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to update API key", perrors.NewErrInternalServerError("Failed to update API key", err))
			return
		}

		writeOK(ctx, stdCtx, "API key updated successfully", apiKey)
	})

	// Delete API key
	r.DELETE("/api/agent-server/api-keys/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		idRaw, err := pathParam(ctx, "id")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		id, err := uuid.Parse(idRaw)
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		if err := svc.Provider.Delete(stdCtx, id); err != nil {
			writeError(ctx, stdCtx, "Failed to delete API key", perrors.NewErrInternalServerError("Failed to delete API key", err))
			return
		}

		writeOK(ctx, stdCtx, "API key deleted successfully", nil)
	})

	// List all provider configs
	r.GET("/api/agent-server/provider-configs", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		configs, err := svc.Provider.ListProviderConfigs(stdCtx)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list provider configs", perrors.NewErrInternalServerError("Failed to list provider configs", err))
			return
		}

		writeOK(ctx, stdCtx, "Provider configs retrieved successfully", configs)
	})

	// Create or update provider config
	r.PUT("/api/agent-server/provider-configs/{provider_type}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		providerTypeRaw, err := pathParam(ctx, "provider_type")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid provider type", perrors.NewErrInvalidRequest("Invalid provider type", err))
			return
		}

		pt := llm.ProviderName(providerTypeRaw)
		if !pt.IsValid() {
			writeError(ctx, stdCtx, "Invalid provider type", perrors.NewErrInvalidRequest("Invalid provider type", errors.New("provider_type must be one of: OpenAI, Anthropic, Gemini, xAI")))
			return
		}

		var body provider2.CreateProviderConfigRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		body.ProviderType = pt
		config, err := svc.Provider.CreateOrUpdateProviderConfig(stdCtx, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create or update provider config", perrors.NewErrInternalServerError("Failed to create or update provider config", err))
			return
		}

		writeOK(ctx, stdCtx, "Provider config saved successfully", config)
	})
}
