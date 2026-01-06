package controllers

import (
	"context"
	"errors"
	"fmt"

	json "github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/api/response"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

// requestContext returns a baseline context for handlers. fasthttp does not provide
// a standard context, so we start from Background for downstream calls.
func requestContext(_ *fasthttp.RequestCtx) context.Context {
	return context.Background()
}

func parseBody(ctx *fasthttp.RequestCtx, target any) error {
	body := ctx.PostBody()
	if len(body) == 0 {
		return errors.New("request body is empty")
	}

	return json.Unmarshal(body, target)
}

func writeError(ctx *fasthttp.RequestCtx, stdCtx context.Context, message string, err error) {
	response.NewResponse[any](stdCtx, message, nil).WithError(err).Write(ctx)
}

func writeOK(ctx *fasthttp.RequestCtx, stdCtx context.Context, message string, data any) {
	response.NewResponse(stdCtx, message, data).Write(ctx)
}

func pathParam(ctx *fasthttp.RequestCtx, key string) (string, error) {
	val := ctx.UserValue(key)
	if val == nil {
		return "", fmt.Errorf("%s is required", key)
	}

	return fmt.Sprint(val), nil
}

func requireUUIDQuery(ctx *fasthttp.RequestCtx, key string) (uuid.UUID, error) {
	raw := ctx.QueryArgs().Peek(key)
	if len(raw) == 0 {
		return uuid.Nil, fmt.Errorf("%s parameter is required", key)
	}

	return uuid.ParseBytes(raw)
}

func requireStringQuery(ctx *fasthttp.RequestCtx, key string) (string, error) {
	raw := ctx.QueryArgs().Peek(key)
	if len(raw) == 0 {
		return "", fmt.Errorf("%s parameter is required", key)
	}

	return string(raw), nil
}

func pathParamUUID(ctx *fasthttp.RequestCtx, key string) (uuid.UUID, error) {
	val, err := pathParam(ctx, key)
	if err != nil {
		return uuid.Nil, err
	}

	return uuid.Parse(val)
}
