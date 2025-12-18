package controllers

import (
	"errors"
	"fmt"

	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/praveen001/uno/internal/services"
	"github.com/praveen001/uno/internal/services/virtual_key"
	"github.com/valyala/fasthttp"

	"github.com/praveen001/uno/internal/perrors"
)

func RegisterVirtualKeyRoutes(r *router.Router, svc *services.Services) {
	// Create virtual key
	r.POST("/api/agent-server/virtual-keys", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		var body virtual_key.CreateVirtualKeyRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name == "" {
			writeError(ctx, stdCtx, "Name is required", perrors.NewErrInvalidRequest("Name is required", errors.New("name is required")))
			return
		}

		if len(body.Providers) == 0 {
			writeError(ctx, stdCtx, "At least one provider is required", perrors.NewErrInvalidRequest("At least one provider is required", errors.New("providers cannot be empty")))
			return
		}

		vk, err := svc.VirtualKey.Create(stdCtx, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create virtual key", perrors.NewErrInternalServerError("Failed to create virtual key", err))
			return
		}

		writeOK(ctx, stdCtx, "Virtual key created successfully", vk)
	})

	// List virtual keys
	r.GET("/api/agent-server/virtual-keys", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		virtualKeys, err := svc.VirtualKey.List(stdCtx)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to list virtual keys", perrors.NewErrInternalServerError("Failed to list virtual keys", err))
			return
		}

		writeOK(ctx, stdCtx, "Virtual keys retrieved successfully", virtualKeys)
	})

	// Update virtual key
	r.PUT("/api/agent-server/virtual-keys/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		idRaw := ctx.UserValue("id")
		id, err := uuid.Parse(fmt.Sprint(idRaw))
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		var body virtual_key.UpdateVirtualKeyRequest
		if err := parseBody(ctx, &body); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", perrors.NewErrInvalidRequest("Invalid request body", err))
			return
		}

		if body.Name != nil && *body.Name == "" {
			writeError(ctx, stdCtx, "Name cannot be empty", perrors.NewErrInvalidRequest("Name cannot be empty", errors.New("name cannot be empty")))
			return
		}

		if body.Providers != nil && len(*body.Providers) == 0 {
			writeError(ctx, stdCtx, "At least one provider is required", perrors.NewErrInvalidRequest("At least one provider is required", errors.New("providers cannot be empty")))
			return
		}

		vk, err := svc.VirtualKey.Update(stdCtx, id, &body)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to update virtual key", perrors.NewErrInternalServerError("Failed to update virtual key", err))
			return
		}

		writeOK(ctx, stdCtx, "Virtual key updated successfully", vk)
	})

	// Delete virtual key
	r.DELETE("/api/agent-server/virtual-keys/{id}", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		idRaw := ctx.UserValue("id")
		id, err := uuid.Parse(fmt.Sprint(idRaw))
		if err != nil {
			writeError(ctx, stdCtx, "Invalid ID format", perrors.NewErrInvalidRequest("Invalid ID format", err))
			return
		}

		if err := svc.VirtualKey.Delete(stdCtx, id); err != nil {
			writeError(ctx, stdCtx, "Failed to delete virtual key", perrors.NewErrInternalServerError("Failed to delete virtual key", err))
			return
		}

		writeOK(ctx, stdCtx, "Virtual key deleted successfully", nil)
	})
}
