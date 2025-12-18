package controllers

import (
	"context"
	"errors"
	"strconv"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/fasthttp/router"
	"github.com/praveen001/uno/internal/api/response"
	"github.com/praveen001/uno/internal/services"
	"github.com/praveen001/uno/internal/services/traces"
	"github.com/valyala/fasthttp"

	"github.com/praveen001/uno/internal/perrors"
)

// RegisterTracesRoutes registers trace-related API routes
func RegisterTracesRoutes(r *router.Group, svc *services.Services) {
	tracesSvc := svc.Traces
	if tracesSvc == nil {
		return
	}

	// List traces with optional filters
	r.GET("/traces", func(reqCtx *fasthttp.RequestCtx) {
		ctx := requestContext(reqCtx)

		params := parseTraceQueryParams(reqCtx)

		traceList, totalCount, err := tracesSvc.ListTraces(ctx, params)
		if err != nil {
			writeError(reqCtx, ctx, "Failed to list traces", perrors.NewErrInternalServerError("Failed to list traces", err))
			return
		}

		writeResponse(reqCtx, ctx, "Traces retrieved successfully", map[string]any{
			"traces": traceList,
			"total":  totalCount,
			"limit":  params.Limit,
			"offset": params.Offset,
		})
	})

	// Get a specific trace with all spans
	r.GET("/traces/{trace_id}", func(reqCtx *fasthttp.RequestCtx) {
		ctx := requestContext(reqCtx)

		traceID := reqCtx.UserValue("trace_id").(string)
		if traceID == "" {
			writeError(reqCtx, ctx, "Trace ID is required", perrors.NewErrInvalidRequest("trace_id is required", errors.New("trace_id is required")))
			return
		}

		trace, err := tracesSvc.GetTrace(ctx, traceID)
		if err != nil {
			writeError(reqCtx, ctx, "Failed to get trace", perrors.NewErrInternalServerError("Failed to get trace", err))
			return
		}

		writeResponse(reqCtx, ctx, "Trace retrieved successfully", trace)
	})

	// Get service statistics
	r.GET("/traces/stats/services", func(reqCtx *fasthttp.RequestCtx) {
		ctx := requestContext(reqCtx)

		startTime, endTime := parseTimeRange(reqCtx)

		stats, err := tracesSvc.GetServiceStats(ctx, startTime, endTime)
		if err != nil {
			writeError(reqCtx, ctx, "Failed to get service stats", perrors.NewErrInternalServerError("Failed to get service stats", err))
			return
		}

		writeResponse(reqCtx, ctx, "Service stats retrieved successfully", stats)
	})

	// Get endpoint statistics
	r.GET("/traces/stats/endpoints", func(reqCtx *fasthttp.RequestCtx) {
		ctx := requestContext(reqCtx)

		serviceName := string(reqCtx.QueryArgs().Peek("service_name"))
		startTime, endTime := parseTimeRange(reqCtx)

		stats, err := tracesSvc.GetEndpointStats(ctx, serviceName, startTime, endTime)
		if err != nil {
			writeError(reqCtx, ctx, "Failed to get endpoint stats", perrors.NewErrInternalServerError("Failed to get endpoint stats", err))
			return
		}

		writeResponse(reqCtx, ctx, "Endpoint stats retrieved successfully", stats)
	})

	// Get list of services
	r.GET("/traces/services", func(reqCtx *fasthttp.RequestCtx) {
		ctx := requestContext(reqCtx)

		services, err := tracesSvc.GetServices(ctx)
		if err != nil {
			writeError(reqCtx, ctx, "Failed to get services", perrors.NewErrInternalServerError("Failed to get services", err))
			return
		}

		writeResponse(reqCtx, ctx, "Services retrieved successfully", services)
	})
}

func parseTraceQueryParams(reqCtx *fasthttp.RequestCtx) *traces.TraceQueryParams {
	params := &traces.TraceQueryParams{
		ServiceName: string(reqCtx.QueryArgs().Peek("service_name")),
		SpanName:    string(reqCtx.QueryArgs().Peek("span_name")),
		TraceID:     string(reqCtx.QueryArgs().Peek("trace_id")),
		Limit:       50,
		Offset:      0,
	}

	if limitStr := string(reqCtx.QueryArgs().Peek("limit")); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = limit
		}
	}

	if offsetStr := string(reqCtx.QueryArgs().Peek("offset")); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Offset = offset
		}
	}

	if minDurStr := string(reqCtx.QueryArgs().Peek("min_duration")); minDurStr != "" {
		if minDur, err := strconv.ParseFloat(minDurStr, 64); err == nil {
			params.MinDuration = &minDur
		}
	}

	if maxDurStr := string(reqCtx.QueryArgs().Peek("max_duration")); maxDurStr != "" {
		if maxDur, err := strconv.ParseFloat(maxDurStr, 64); err == nil {
			params.MaxDuration = &maxDur
		}
	}

	if hasErrorsStr := string(reqCtx.QueryArgs().Peek("has_errors")); hasErrorsStr != "" {
		hasErrors := hasErrorsStr == "true" || hasErrorsStr == "1"
		params.HasErrors = &hasErrors
	}

	params.StartTime, params.EndTime = parseTimeRange(reqCtx)

	return params
}

func parseTimeRange(reqCtx *fasthttp.RequestCtx) (time.Time, time.Time) {
	var startTime, endTime time.Time

	if startStr := string(reqCtx.QueryArgs().Peek("start_time")); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = t
		}
	}
	if startTime.IsZero() {
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if endStr := string(reqCtx.QueryArgs().Peek("end_time")); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = t
		}
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	return startTime, endTime
}

func writeResponse(reqCtx *fasthttp.RequestCtx, ctx context.Context, message string, data any) {
	resp := response.NewResponse(ctx, message, data)
	reqCtx.SetContentType("application/json")
	reqCtx.SetStatusCode(fasthttp.StatusOK)
	buf, _ := json.Marshal(resp)
	reqCtx.SetBody(buf)
}
