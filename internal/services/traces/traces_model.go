package traces

import (
	"time"
)

// Trace represents a complete trace with all its spans
type Trace struct {
	TraceID    string    `json:"trace_id"`
	RootSpan   *Span     `json:"root_span,omitempty"`
	Spans      []Span    `json:"spans"`
	SpanCount  int       `json:"span_count"`
	Duration   float64   `json:"duration_ms"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Services   []string  `json:"services"`
	HasErrors  bool      `json:"has_errors"`
	ErrorCount int       `json:"error_count"`
}

// Span represents a single span within a trace
type Span struct {
	TraceID            string            `json:"trace_id"`
	SpanID             string            `json:"span_id"`
	ParentSpanID       string            `json:"parent_span_id,omitempty"`
	Name               string            `json:"name"`
	Kind               string            `json:"kind"`
	ServiceName        string            `json:"service_name"`
	Duration           float64           `json:"duration_ms"`
	StartTime          time.Time         `json:"start_time"`
	EndTime            time.Time         `json:"end_time"`
	StatusCode         string            `json:"status_code"`
	StatusMessage      string            `json:"status_message,omitempty"`
	Attributes         map[string]string `json:"attributes,omitempty"`
	ResourceAttributes map[string]string `json:"resource_attributes,omitempty"`
	Events             []SpanEvent       `json:"events,omitempty"`
	Children           []Span            `json:"children,omitempty"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Timestamp  time.Time         `json:"timestamp"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// TraceListItem is a summary view of a trace for listing
type TraceListItem struct {
	TraceID      string    `json:"trace_id"`
	RootSpanName string    `json:"root_span_name"`
	ServiceName  string    `json:"service_name"`
	Duration     float64   `json:"duration_ms"`
	SpanCount    uint64    `json:"span_count"`
	StartTime    time.Time `json:"start_time"`
	HasErrors    bool      `json:"has_errors"`
	ErrorCount   uint64    `json:"error_count"`
}

// ServiceStats represents aggregated statistics for a service
type ServiceStats struct {
	ServiceName string  `json:"service_name"`
	SpanCount   uint64  `json:"span_count"`
	ErrorCount  uint64  `json:"error_count"`
	ErrorRate   float64 `json:"error_rate"`
	AvgDuration float64 `json:"avg_duration_ms"`
	MaxDuration float64 `json:"max_duration_ms"`
	MinDuration float64 `json:"min_duration_ms"`
	P50Duration float64 `json:"p50_duration_ms"`
	P95Duration float64 `json:"p95_duration_ms"`
	P99Duration float64 `json:"p99_duration_ms"`
}

// EndpointStats represents aggregated statistics for an endpoint
type EndpointStats struct {
	ServiceName string  `json:"service_name"`
	SpanName    string  `json:"span_name"`
	CallCount   uint64  `json:"call_count"`
	ErrorCount  uint64  `json:"error_count"`
	ErrorRate   float64 `json:"error_rate"`
	AvgDuration float64 `json:"avg_duration_ms"`
	P50Duration float64 `json:"p50_duration_ms"`
	P95Duration float64 `json:"p95_duration_ms"`
	P99Duration float64 `json:"p99_duration_ms"`
}

// TraceQueryParams holds query parameters for trace search
type TraceQueryParams struct {
	ServiceName string    `json:"service_name,omitempty"`
	SpanName    string    `json:"span_name,omitempty"`
	TraceID     string    `json:"trace_id,omitempty"`
	MinDuration *float64  `json:"min_duration_ms,omitempty"`
	MaxDuration *float64  `json:"max_duration_ms,omitempty"`
	HasErrors   *bool     `json:"has_errors,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Limit       int       `json:"limit"`
	Offset      int       `json:"offset"`
}
