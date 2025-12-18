package traces

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// TracesService provides methods to query trace data from ClickHouse
type TracesService struct {
	conn driver.Conn
}

// NewTracesService creates a new TracesService
func NewTracesService(conn driver.Conn) *TracesService {
	return &TracesService{conn: conn}
}

// ListTraces returns a paginated list of traces
func (s *TracesService) ListTraces(ctx context.Context, params *TraceQueryParams) ([]TraceListItem, int64, error) {
	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Limit > 1000 {
		params.Limit = 1000
	}
	if params.StartTime.IsZero() {
		params.StartTime = time.Now().Add(-24 * time.Hour)
	}
	if params.EndTime.IsZero() {
		params.EndTime = time.Now()
	}

	// Build WHERE clause with positional parameters
	conditions := []string{"Timestamp >= ?", "Timestamp <= ?"}
	args := []interface{}{params.StartTime, params.EndTime}

	if params.ServiceName != "" {
		conditions = append(conditions, "ServiceName = ?")
		args = append(args, params.ServiceName)
	}
	if params.SpanName != "" {
		conditions = append(conditions, "SpanName LIKE ?")
		args = append(args, "%"+params.SpanName+"%")
	}
	if params.TraceID != "" {
		conditions = append(conditions, "TraceId LIKE ?")
		args = append(args, params.TraceID+"%")
	}
	if params.MinDuration != nil {
		conditions = append(conditions, "Duration / 1000000 >= ?")
		args = append(args, *params.MinDuration)
	}
	if params.MaxDuration != nil {
		conditions = append(conditions, "Duration / 1000000 <= ?")
		args = append(args, *params.MaxDuration)
	}

	whereClause := ""
	for i, cond := range conditions {
		if i > 0 {
			whereClause += " AND "
		}
		whereClause += cond
	}

	// Query for trace summaries - use argMin to get the span name of the span with empty ParentSpanId (root span)
	// If no span has empty ParentSpanId, fall back to the earliest span
	query := fmt.Sprintf(`
		SELECT
			TraceId,
			argMinIf(SpanName, Timestamp, ParentSpanId = '') as RootSpanName,
			argMinIf(ServiceName, Timestamp, ParentSpanId = '') as ServiceName,
			max(Duration) / 1000000 as DurationMs,
			count() as SpanCount,
			min(Timestamp) as StartTime,
			countIf(StatusCode = 'STATUS_CODE_ERROR') > 0 as HasErrors,
			countIf(StatusCode = 'STATUS_CODE_ERROR') as ErrorCount
		FROM otel.otel_traces
		WHERE %s
		GROUP BY TraceId
		ORDER BY StartTime DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	// Add limit and offset to args
	queryArgs := append(args, params.Limit, params.Offset)

	rows, err := s.conn.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query traces: %w", err)
	}
	defer rows.Close()

	var traces []TraceListItem
	for rows.Next() {
		var t TraceListItem
		if err := rows.Scan(
			&t.TraceID,
			&t.RootSpanName,
			&t.ServiceName,
			&t.Duration,
			&t.SpanCount,
			&t.StartTime,
			&t.HasErrors,
			&t.ErrorCount,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan trace row: %w", err)
		}
		traces = append(traces, t)
	}

	// Get total count
	countQuery := fmt.Sprintf(`
		SELECT count(DISTINCT TraceId)
		FROM otel.otel_traces
		WHERE %s
	`, whereClause)

	var totalCount uint64
	if err := s.conn.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to get trace count: %w", err)
	}

	return traces, int64(totalCount), nil
}

// GetTrace returns a complete trace with all spans organized hierarchically
func (s *TracesService) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	query := `
		SELECT
			TraceId,
			SpanId,
			ParentSpanId,
			SpanName,
			SpanKind,
			ServiceName,
			Duration / 1000000 as DurationMs,
			Timestamp as StartTime,
			Timestamp + toIntervalMicrosecond(toUInt64(Duration / 1000)) as EndTime,
			StatusCode,
			StatusMessage,
			SpanAttributes,
			ResourceAttributes
		FROM otel.otel_traces
		WHERE TraceId = ?
		ORDER BY Timestamp ASC
	`

	rows, err := s.conn.Query(ctx, query, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query trace spans: %w", err)
	}
	defer rows.Close()

	spanMap := make(map[string]*Span)
	var spans []Span
	var rootSpan *Span
	services := make(map[string]bool)
	var hasErrors bool
	var errorCount int

	for rows.Next() {
		var span Span
		var spanAttrs, resourceAttrs map[string]string

		if err := rows.Scan(
			&span.TraceID,
			&span.SpanID,
			&span.ParentSpanID,
			&span.Name,
			&span.Kind,
			&span.ServiceName,
			&span.Duration,
			&span.StartTime,
			&span.EndTime,
			&span.StatusCode,
			&span.StatusMessage,
			&spanAttrs,
			&resourceAttrs,
		); err != nil {
			return nil, fmt.Errorf("failed to scan span row: %w", err)
		}

		span.Attributes = spanAttrs
		span.ResourceAttributes = resourceAttrs
		spans = append(spans, span)
		spanMap[span.SpanID] = &spans[len(spans)-1]
		services[span.ServiceName] = true

		if span.StatusCode == "STATUS_CODE_ERROR" {
			hasErrors = true
			errorCount++
		}

		if span.ParentSpanID == "" {
			rootSpan = &spans[len(spans)-1]
		}
	}

	if len(spans) == 0 {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	// Build service list
	var serviceList []string
	for svc := range services {
		serviceList = append(serviceList, svc)
	}

	// Calculate trace duration and times
	var minStart, maxEnd time.Time
	for i, span := range spans {
		if i == 0 || span.StartTime.Before(minStart) {
			minStart = span.StartTime
		}
		if i == 0 || span.EndTime.After(maxEnd) {
			maxEnd = span.EndTime
		}
	}

	trace := &Trace{
		TraceID:    traceID,
		RootSpan:   rootSpan,
		Spans:      spans,
		SpanCount:  len(spans),
		Duration:   float64(maxEnd.Sub(minStart).Milliseconds()),
		StartTime:  minStart,
		EndTime:    maxEnd,
		Services:   serviceList,
		HasErrors:  hasErrors,
		ErrorCount: errorCount,
	}

	return trace, nil
}

// GetServiceStats returns aggregated statistics for services
func (s *TracesService) GetServiceStats(ctx context.Context, startTime, endTime time.Time) ([]ServiceStats, error) {
	if startTime.IsZero() {
		startTime = time.Now().Add(-24 * time.Hour)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	query := `
		SELECT
			ServiceName,
			count() as SpanCount,
			countIf(StatusCode = 'STATUS_CODE_ERROR') as ErrorCount,
			countIf(StatusCode = 'STATUS_CODE_ERROR') / count() * 100 as ErrorRate,
			avg(Duration / 1000000) as AvgDurationMs,
			max(Duration / 1000000) as MaxDurationMs,
			min(Duration / 1000000) as MinDurationMs,
			quantile(0.50)(Duration / 1000000) as P50DurationMs,
			quantile(0.95)(Duration / 1000000) as P95DurationMs,
			quantile(0.99)(Duration / 1000000) as P99DurationMs
		FROM otel.otel_traces
		WHERE Timestamp >= ? AND Timestamp <= ?
		GROUP BY ServiceName
		ORDER BY SpanCount DESC
	`

	rows, err := s.conn.Query(ctx, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query service stats: %w", err)
	}
	defer rows.Close()

	var stats []ServiceStats
	for rows.Next() {
		var st ServiceStats
		if err := rows.Scan(
			&st.ServiceName,
			&st.SpanCount,
			&st.ErrorCount,
			&st.ErrorRate,
			&st.AvgDuration,
			&st.MaxDuration,
			&st.MinDuration,
			&st.P50Duration,
			&st.P95Duration,
			&st.P99Duration,
		); err != nil {
			return nil, fmt.Errorf("failed to scan service stats: %w", err)
		}
		stats = append(stats, st)
	}

	return stats, nil
}

// GetEndpointStats returns aggregated statistics for endpoints
func (s *TracesService) GetEndpointStats(ctx context.Context, serviceName string, startTime, endTime time.Time) ([]EndpointStats, error) {
	if startTime.IsZero() {
		startTime = time.Now().Add(-24 * time.Hour)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	conditions := []string{"Timestamp >= ?", "Timestamp <= ?"}
	args := []interface{}{startTime, endTime}

	if serviceName != "" {
		conditions = append(conditions, "ServiceName = ?")
		args = append(args, serviceName)
	}

	whereClause := ""
	for i, cond := range conditions {
		if i > 0 {
			whereClause += " AND "
		}
		whereClause += cond
	}

	query := fmt.Sprintf(`
		SELECT
			ServiceName,
			SpanName,
			count() as CallCount,
			countIf(StatusCode = 'STATUS_CODE_ERROR') as ErrorCount,
			countIf(StatusCode = 'STATUS_CODE_ERROR') / count() * 100 as ErrorRate,
			avg(Duration / 1000000) as AvgDurationMs,
			quantile(0.50)(Duration / 1000000) as P50DurationMs,
			quantile(0.95)(Duration / 1000000) as P95DurationMs,
			quantile(0.99)(Duration / 1000000) as P99DurationMs
		FROM otel.otel_traces
		WHERE %s
		GROUP BY ServiceName, SpanName
		ORDER BY CallCount DESC
		LIMIT 100
	`, whereClause)

	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query endpoint stats: %w", err)
	}
	defer rows.Close()

	var stats []EndpointStats
	for rows.Next() {
		var st EndpointStats
		if err := rows.Scan(
			&st.ServiceName,
			&st.SpanName,
			&st.CallCount,
			&st.ErrorCount,
			&st.ErrorRate,
			&st.AvgDuration,
			&st.P50Duration,
			&st.P95Duration,
			&st.P99Duration,
		); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint stats: %w", err)
		}
		stats = append(stats, st)
	}

	return stats, nil
}

// GetServices returns a list of unique service names
func (s *TracesService) GetServices(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT ServiceName
		FROM otel.otel_traces
		WHERE Timestamp >= now() - INTERVAL 7 DAY
		ORDER BY ServiceName
	`

	rows, err := s.conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query services: %w", err)
	}
	defer rows.Close()

	var services []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan service name: %w", err)
		}
		services = append(services, name)
	}

	return services, nil
}
