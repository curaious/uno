package controllers

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/perrors"
	"github.com/curaious/uno/internal/services"
	"github.com/curaious/uno/internal/utils"
	"github.com/curaious/uno/pkg/gateway"
	"github.com/curaious/uno/pkg/gateway/providers/anthropic/anthropic_responses"
	"github.com/curaious/uno/pkg/gateway/providers/gemini/gemini_embeddings"
	"github.com/curaious/uno/pkg/gateway/providers/gemini/gemini_responses"
	"github.com/curaious/uno/pkg/gateway/providers/openai/openai_chat_completion"
	"github.com/curaious/uno/pkg/gateway/providers/openai/openai_embeddings"
	"github.com/curaious/uno/pkg/gateway/providers/openai/openai_responses"
	"github.com/curaious/uno/pkg/llm"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel/codes"
)

func RegisterGatewayRoutes(r *router.Group, svc *services.Services, llmGateway *gateway.LLMGateway) {
	r.Handle(http.MethodPost, "/responses", func(reqCtx *fasthttp.RequestCtx) {
		stdCtx := requestContext(reqCtx)

		// Create trace
		ctx, span := tracer.Start(reqCtx.UserValue("traceCtx").(context.Context), "Controller.Gateway.Responses")

		vk := extractKey(reqCtx)

		// Parse request body into openai's responses input format
		var openAiRequest *openai_responses.Request
		if err := sonic.Unmarshal(reqCtx.PostBody(), &openAiRequest); err != nil {
			writeError(reqCtx, stdCtx, "Error unmarshalling the request body", perrors.NewErrInvalidRequest("Error unmarshalling the request body", err))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		}

		// Convert it into generic responses input
		nativeRequest := openAiRequest.ToNativeRequest()

		// Create a gateway request
		req := &llm.Request{
			OfResponsesInput: nativeRequest,
		}

		frags := strings.Split(nativeRequest.Model, ":")
		providerName := llm.ProviderName(frags[0])
		model := frags[1]
		nativeRequest.Model = model

		// Handle non-streaming request
		if !nativeRequest.IsStreamingRequest() {
			// Call gateway to handle the gateway request
			out, err := llmGateway.HandleRequest(ctx, providerName, vk, req)
			if err != nil {
				writeError(reqCtx, stdCtx, "Error handling request", perrors.NewErrInternalServerError("Error handling request", err))
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				span.End()
				return
			}

			buf, err := sonic.Marshal(out.OfResponsesOutput)
			if err != nil {
				writeError(reqCtx, stdCtx, "Error marshalling response", perrors.NewErrInternalServerError("Error marshalling response", err))
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				span.End()
				return
			}

			if _, err = reqCtx.Write(buf); err != nil {
				writeError(reqCtx, stdCtx, "Error encoding response", perrors.NewErrInternalServerError("Error encoding response", err))
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				span.End()
				return
			}

			span.End()
			return
		}

		// Handling streaming request
		out, err := llmGateway.HandleStreamingRequest(ctx, providerName, vk, req)
		if err != nil {
			writeError(reqCtx, stdCtx, "Error handling LLM Gateway streaming request", perrors.NewErrInternalServerError("Error handling LLM Gateway streaming request", err))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		}

		reqCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
			defer span.End()
		loop:
			for {
				select {
				case data, ok := <-out.ResponsesStreamData:
					if !ok {
						break loop
					}

					buf, err := sonic.Marshal(data)
					if err != nil {
						slog.WarnContext(reqCtx, "Error encoding response: %v\n", err)
						continue
					}
					fmt.Println(string(buf))

					_, _ = fmt.Fprintf(w, "event: %s\n", data.ChunkType())
					_, _ = fmt.Fprintf(w, "data: %s\n\n", buf)

					err = w.Flush()
					if err != nil {
						slog.WarnContext(reqCtx, "Error flushing buffer: %v\n", err)
					}
				}
			}
		})
	})
	r.Handle(http.MethodPost, "/embeddings", func(reqCtx *fasthttp.RequestCtx) {
		stdCtx := requestContext(reqCtx)

		// Create trace
		ctx, span := tracer.Start(reqCtx.UserValue("traceCtx").(context.Context), "Controller.Gateway.Responses")
		defer span.End()

		vk := extractKey(reqCtx)

		// Parse request body into openai's embedding input format
		var openAiRequest *openai_embeddings.Request
		if err := sonic.Unmarshal(reqCtx.PostBody(), &openAiRequest); err != nil {
			writeError(reqCtx, stdCtx, "Error unmarshalling the request body", perrors.NewErrInvalidRequest("Error unmarshalling the request body", err))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		}

		// Convert it into generic embeddings input
		nativeRequest := openAiRequest.ToNativeRequest()

		// Create a gateway request
		req := &llm.Request{
			OfEmbeddingsInput: nativeRequest,
		}

		frags := strings.Split(nativeRequest.Model, ":")
		providerName := llm.ProviderName(frags[0])
		model := frags[1]
		nativeRequest.Model = model

		out, err := llmGateway.HandleRequest(ctx, providerName, vk, req)
		if err != nil {
			writeError(reqCtx, stdCtx, "Error handling request", perrors.NewErrInternalServerError("Error handling request", err))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		}

		buf, err := sonic.Marshal(out.OfEmbeddingsOutput)
		if err != nil {
			writeError(reqCtx, stdCtx, "Error marshalling response", perrors.NewErrInternalServerError("Error marshalling response", err))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		}

		if _, err = reqCtx.Write(buf); err != nil {
			writeError(reqCtx, stdCtx, "Error encoding response", perrors.NewErrInternalServerError("Error encoding response", err))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		}
	})

	r.Handle(http.MethodPost, "/anthropic/v1/messages", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)

		// Extract virtual key from headers
		vkBuf := ctx.Request.Header.Peek("x-api-key")
		vk := string(vkBuf)

		// Parse request body into openai's responses input format
		var anthropicRequest *anthropic_responses.Request
		if err := sonic.Unmarshal(ctx.PostBody(), &anthropicRequest); err != nil {
			writeError(ctx, stdCtx, "Error unmarshalling the request body", perrors.NewErrInvalidRequest("Error unmarshalling the request body", err))
			return
		}

		// Convert it into generic responses input
		nativeRequest := anthropicRequest.ToNativeRequest()

		// Create a gateway request
		req := &llm.Request{
			OfResponsesInput: nativeRequest,
		}

		// Handle non-streaming request
		if !nativeRequest.IsStreamingRequest() {
			// Call gateway to handle the gateway request
			out, err := llmGateway.HandleRequest(ctx, llm.ProviderNameAnthropic, vk, req)
			if err != nil {
				writeError(ctx, stdCtx, "Error handling request", perrors.NewErrInternalServerError("Error handling request", err))
				return
			}

			// Convert generic output into openai specific output
			anthropicResponse := anthropic_responses.NativeResponseToResponse(out.OfResponsesOutput)

			buf, err := sonic.Marshal(anthropicResponse)
			if err != nil {
				writeError(ctx, stdCtx, "Error marshalling response", perrors.NewErrInternalServerError("Error marshalling response", err))
				return
			}

			if _, err = ctx.Write(buf); err != nil {
				writeError(ctx, stdCtx, "Error encoding response", perrors.NewErrInternalServerError("Error encoding response", err))
				return
			}

			return
		}

		// Handling streaming request
		out, err := llmGateway.HandleStreamingRequest(ctx, llm.ProviderNameAnthropic, vk, req)
		if err != nil {
			writeError(ctx, stdCtx, "Error handling LLM Gateway streaming request", perrors.NewErrInternalServerError("Error handling LLM Gateway streaming request", err))
			return
		}

		converter := anthropic_responses.NativeResponseChunkToResponseChunkConverter{}

		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		loop:
			for {
				select {
				case nativeChunk, ok := <-out.ResponsesStreamData:
					if !ok {
						break loop
					}

					//b, _ := sonic.Marshal(nativeChunk)
					//fmt.Println("---\nnative chunk -> " + string(b))
					anthropicChunks := converter.NativeResponseChunkToResponseChunk(nativeChunk)
					for _, anthropicChunk := range anthropicChunks {
						buf, err := sonic.Marshal(&anthropicChunk)
						if err != nil {
							slog.WarnContext(ctx, "Error encoding response: %v\n", err)
							continue
						}
						//fmt.Println("\t\t <- Anthropic Chunk: " + string(buf))

						_, _ = fmt.Fprintf(w, "event: %s\n", anthropicChunk.ChunkType())
						_, _ = fmt.Fprintf(w, "data: %s\n\n", buf)

						err = w.Flush()
						if err != nil {
							slog.WarnContext(ctx, "Error flushing buffer: %v\n", err)
						}
					}
				}
			}
		})
	})
	r.Handle(http.MethodPost, "/gemini/v1beta/models/{model}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)

		vk := string(ctx.Request.Header.Peek("x-goog-api-key"))

		// Extract virtual key from headers
		var err error
		if vk == "" {
			vk, err = requireStringQuery(ctx, "key")
			if err != nil {
				writeError(ctx, stdCtx, "Key is required", perrors.NewErrInvalidRequest("Key is required", err))
				return
			}
		}

		// Parse URL to get model and stream
		modelParam, err := pathParam(ctx, "model")
		if err != nil {
			writeError(ctx, stdCtx, "Invalid model format", perrors.NewErrInvalidRequest("Invalid model format", err))
			return
		}
		frag := strings.Split(modelParam, ":")

		switch frag[1] {
		case "embedContent", "batchEmbedContents":
			GeminiEmbedding(ctx, stdCtx, frag[0], frag[1], vk, llmGateway)
		default:
			GeminiResponse(ctx, stdCtx, frag[0], frag[1], vk, llmGateway)
		}

	})
	r.Handle(http.MethodPost, "/openai/responses", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)

		// Extract virtual key from headers
		vkBuf := ctx.Request.Header.Peek("Authorization")
		vk := strings.TrimPrefix(string(vkBuf), "Bearer ")

		// Parse request body into openai's responses input format
		var openAiRequest *openai_responses.Request
		if err := sonic.Unmarshal(ctx.PostBody(), &openAiRequest); err != nil {
			writeError(ctx, stdCtx, "Error unmarshalling the request body", perrors.NewErrInvalidRequest("Error unmarshalling the request body", err))
			return
		}

		provider := llm.ProviderNameOpenAI
		if strings.Contains(openAiRequest.Model, "/") {
			frag := strings.SplitN(openAiRequest.Model, "/", 2)
			provider = llm.ProviderName(frag[0])
			openAiRequest.Model = frag[1]
		}

		// Convert it into generic responses input
		nativeRequest := openAiRequest.ToNativeRequest()

		// Create a gateway request
		req := &llm.Request{
			OfResponsesInput: nativeRequest,
		}

		// Handle non-streaming request
		if !nativeRequest.IsStreamingRequest() {
			// Call gateway to handle the gateway request
			out, err := llmGateway.HandleRequest(ctx, provider, vk, req)
			if err != nil {
				writeError(ctx, stdCtx, "Error handling request", perrors.NewErrInternalServerError("Error handling request", err))
				return
			}

			// Convert generic output into openai specific output
			openAiOut := openai_responses.NativeResponseToResponse(out.OfResponsesOutput)

			buf, err := sonic.Marshal(openAiOut)
			if err != nil {
				writeError(ctx, stdCtx, "Error marshalling response", perrors.NewErrInternalServerError("Error marshalling response", err))
				return
			}

			if _, err = ctx.Write(buf); err != nil {
				writeError(ctx, stdCtx, "Error encoding response", perrors.NewErrInternalServerError("Error encoding response", err))
				return
			}

			return
		}

		// Handling streaming request
		out, err := llmGateway.HandleStreamingRequest(ctx, llm.ProviderNameOpenAI, vk, req)
		if err != nil {
			writeError(ctx, stdCtx, "Error handling LLM Gateway streaming request", perrors.NewErrInternalServerError("Error handling LLM Gateway streaming request", err))
			return
		}

		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		loop:
			for {
				select {
				case data, ok := <-out.ResponsesStreamData:
					if !ok {
						break loop
					}

					buf, err := sonic.Marshal(data)
					if err != nil {
						slog.WarnContext(ctx, "Error encoding response: %v\n", err)
						continue
					}
					fmt.Println(string(buf))

					_, _ = fmt.Fprintf(w, "event: %s\n", data.ChunkType())
					_, _ = fmt.Fprintf(w, "data: %s\n\n", buf)

					err = w.Flush()
					if err != nil {
						slog.WarnContext(ctx, "Error flushing buffer: %v\n", err)
					}
				}
			}
		})
	})
	r.Handle(http.MethodPost, "/openai/embeddings", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)

		// Extract virtual key from headers
		vkBuf := ctx.Request.Header.Peek("Authorization")
		vk := strings.TrimPrefix(string(vkBuf), "Bearer ")

		// Parse request body into openai's responses input format
		var openAiRequest *openai_embeddings.Request
		if err := sonic.Unmarshal(ctx.PostBody(), &openAiRequest); err != nil {
			writeError(ctx, stdCtx, "Error unmarshalling the request body", perrors.NewErrInvalidRequest("Error unmarshalling the request body", err))
			return
		}

		// Convert it into generic responses input
		nativeRequest := openAiRequest.ToNativeRequest()

		// Create a gateway request
		req := &llm.Request{
			OfEmbeddingsInput: nativeRequest,
		}

		// Call gateway to handle the gateway request
		out, err := llmGateway.HandleRequest(ctx, llm.ProviderNameOpenAI, vk, req)
		if err != nil {
			writeError(ctx, stdCtx, "Error handling request", perrors.NewErrInternalServerError("Error handling request", err))
			return
		}

		// Convert generic output into openai specific output
		openAiOut := openai_embeddings.NativeResponseToResponse(out.OfEmbeddingsOutput)

		buf, err := sonic.Marshal(openAiOut)
		if err != nil {
			writeError(ctx, stdCtx, "Error marshalling response", perrors.NewErrInternalServerError("Error marshalling response", err))
			return
		}

		if _, err = ctx.Write(buf); err != nil {
			writeError(ctx, stdCtx, "Error encoding response", perrors.NewErrInternalServerError("Error encoding response", err))
			return
		}
	})
	r.Handle(http.MethodPost, "/openai/chat/completions", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)

		// Extract virtual key from headers
		vkBuf := ctx.Request.Header.Peek("Authorization")
		vk := strings.TrimPrefix(string(vkBuf), "Bearer ")

		// Parse request body into openai's chat completion input format
		var openAiRequest *openai_chat_completion.Request
		if err := sonic.Unmarshal(ctx.PostBody(), &openAiRequest); err != nil {
			writeError(ctx, stdCtx, "Error unmarshalling the request body", perrors.NewErrInvalidRequest("Error unmarshalling the request body", err))
			return
		}

		// Convert it into generic chat completion input
		nativeRequest := openAiRequest.ToNativeRequest()

		// Create a gateway request
		req := &llm.Request{
			OfChatCompletionInput: nativeRequest,
		}

		if !nativeRequest.IsStreamingRequest() {
			// Call gateway to handle the gateway request
			out, err := llmGateway.HandleRequest(stdCtx, llm.ProviderNameOpenAI, vk, req)
			if err != nil {
				writeError(ctx, stdCtx, "Error handling request", perrors.NewErrInternalServerError("Error handling request", err))
				return
			}

			// Convert generic output into openai specific output
			openAiOut := openai_chat_completion.NativeResponseToResponse(out.OfChatCompletionOutput)

			buf, err := sonic.Marshal(openAiOut)
			if err != nil {
				writeError(ctx, stdCtx, "Error marshalling response", perrors.NewErrInternalServerError("Error marshalling response", err))
				return
			}

			if _, err = ctx.Write(buf); err != nil {
				writeError(ctx, stdCtx, "Error encoding response", perrors.NewErrInternalServerError("Error encoding response", err))
				return
			}

			return
		}

		// Handling streaming request
		out, err := llmGateway.HandleStreamingRequest(ctx, llm.ProviderNameOpenAI, vk, req)
		if err != nil {
			writeError(ctx, stdCtx, "Error handling LLM Gateway streaming request", perrors.NewErrInternalServerError("Error handling LLM Gateway streaming request", err))
			return
		}

		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		loop:
			for {
				select {
				case data, ok := <-out.ChatCompletionStreamData:
					if !ok {
						_, _ = fmt.Fprintf(w, "data: [DONE]")
						break loop
					}

					buf, err := sonic.Marshal(data)
					if err != nil {
						slog.WarnContext(ctx, "Error encoding response: %v\n", err)
						continue
					}
					fmt.Println(string(buf))

					_, _ = fmt.Fprintf(w, "data: %s\n\n", buf)

					err = w.Flush()
					if err != nil {
						slog.WarnContext(ctx, "Error flushing buffer: %v\n", err)
					}
				}
			}
		})
	})
}

func GeminiResponse(ctx *fasthttp.RequestCtx, stdCtx context.Context, model, action, vk string, llmGateway *gateway.LLMGateway) {
	// Parse request body into openai's responses input format
	var geminiRequest *gemini_responses.Request
	if err := sonic.Unmarshal(ctx.PostBody(), &geminiRequest); err != nil {
		writeError(ctx, stdCtx, "Error unmarshalling the request body", perrors.NewErrInvalidRequest("Error unmarshalling the request body", err))
		return
	}

	geminiRequest.Model = model
	geminiRequest.Stream = utils.Ptr(action == "streamGenerateContent")

	// Convert it into generic responses input
	nativeRequest := geminiRequest.ToNativeRequest()

	// Create a gateway request
	req := &llm.Request{
		OfResponsesInput: nativeRequest,
	}

	// Handle non-streaming request
	if !nativeRequest.IsStreamingRequest() {
		// Call gateway to handle the gateway request
		out, err := llmGateway.HandleRequest(ctx, llm.ProviderNameGemini, vk, req)
		if err != nil {
			writeError(ctx, stdCtx, "Error handling request", perrors.NewErrInternalServerError("Error handling request", err))
			return
		}

		// Convert generic output into openai specific output
		anthropicResponse := gemini_responses.NativeResponseToResponse(out.OfResponsesOutput)

		buf, err := sonic.Marshal(anthropicResponse)
		if err != nil {
			writeError(ctx, stdCtx, "Error marshalling response", perrors.NewErrInternalServerError("Error marshalling response", err))
			return
		}

		if _, err = ctx.Write(buf); err != nil {
			writeError(ctx, stdCtx, "Error encoding response", perrors.NewErrInternalServerError("Error encoding response", err))
			return
		}

		return
	}

	// Handling streaming request
	out, err := llmGateway.HandleStreamingRequest(ctx, llm.ProviderNameGemini, vk, req)
	if err != nil {
		writeError(ctx, stdCtx, "Error handling LLM Gateway streaming request", perrors.NewErrInternalServerError("Error handling LLM Gateway streaming request", err))
		return
	}

	converter := gemini_responses.NativeResponseChunkToResponseChunkConverter{}

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
	loop:
		for {
			select {
			case nativeChunk, ok := <-out.ResponsesStreamData:
				if !ok {
					break loop
				}

				geminiChunks := converter.NativeResponseChunkToResponseChunk(nativeChunk)
				for _, geminiChunk := range geminiChunks {
					buf, err := sonic.Marshal(&geminiChunk)
					if err != nil {
						slog.WarnContext(ctx, "Error encoding response: %v\n", err)
						continue
					}
					fmt.Println(string(buf))

					_, _ = fmt.Fprintf(w, "data: %s\n\n", buf)

					err = w.Flush()
					if err != nil {
						slog.WarnContext(ctx, "Error flushing buffer: %v\n", err)
					}
				}
			}
		}
	})
}

func GeminiEmbedding(ctx *fasthttp.RequestCtx, stdCtx context.Context, model, action, vk string, llmGateway *gateway.LLMGateway) {
	// Parse request body into openai's responses input format
	var geminiRequest *gemini_embeddings.Request
	if err := sonic.Unmarshal(ctx.PostBody(), &geminiRequest); err != nil {
		writeError(ctx, stdCtx, "Error unmarshalling the request body", perrors.NewErrInvalidRequest("Error unmarshalling the request body", err))
		return
	}

	// Convert it into generic responses input
	nativeRequest := geminiRequest.ToNativeRequest()

	// Create a gateway request
	req := &llm.Request{
		OfEmbeddingsInput: nativeRequest,
	}

	// Call gateway to handle the gateway request
	out, err := llmGateway.HandleRequest(ctx, llm.ProviderNameGemini, vk, req)
	if err != nil {
		writeError(ctx, stdCtx, "Error handling request", perrors.NewErrInternalServerError("Error handling request", err))
		return
	}

	// Convert generic output into openai specific output
	geminiResponse := gemini_embeddings.NativeResponseToResponse(out.OfEmbeddingsOutput)

	buf, err := sonic.Marshal(geminiResponse)
	if err != nil {
		writeError(ctx, stdCtx, "Error marshalling response", perrors.NewErrInternalServerError("Error marshalling response", err))
		return
	}

	if _, err = ctx.Write(buf); err != nil {
		writeError(ctx, stdCtx, "Error encoding response", perrors.NewErrInternalServerError("Error encoding response", err))
		return
	}

	return
}

func extractKey(ctx *fasthttp.RequestCtx) string {
	// Extract virtual key from headers
	vkBuf := ctx.Request.Header.Peek("x-virtual-key")
	vk := string(vkBuf)

	if vk == "" {
		vkBuf = ctx.Request.Header.Peek("Authorization")
		vk = strings.TrimPrefix(string(vkBuf), "Bearer ")
	}

	if vk == "" {
		vkBuf = ctx.Request.Header.Peek("Authorization")
		vk = string(vkBuf)
	}

	if vk == "" {
		vkBuf = ctx.Request.Header.Peek("x-goog-api-key")
		vk = string(vkBuf)
	}

	return vk
}
