-- Create database for OpenTelemetry data
CREATE DATABASE IF NOT EXISTS otel;

-- The OTEL collector will create the tables automatically with create_schema: true
-- But we can also create custom views and materialized views for better query performance

-- Custom view for trace spans with flattened attributes
CREATE VIEW IF NOT EXISTS otel.traces_view AS
SELECT
    TraceId,
    SpanId,
    ParentSpanId,
    SpanName,
    SpanKind,
    ServiceName,
    Duration / 1000000 AS DurationMs,
    StatusCode,
    StatusMessage,
    Timestamp,
    SpanAttributes,
    ResourceAttributes,
    Events.Timestamp AS EventTimestamps,
    Events.Name AS EventNames,
    Events.Attributes AS EventAttributes
FROM otel.otel_traces
ORDER BY Timestamp DESC;

-- Materialized view for service-level statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS otel.service_stats
ENGINE = SummingMergeTree()
ORDER BY (ServiceName, toStartOfHour(Timestamp))
AS SELECT
    ServiceName,
    toStartOfHour(Timestamp) AS Hour,
    count() AS SpanCount,
    countIf(StatusCode = 'STATUS_CODE_ERROR') AS ErrorCount,
    avg(Duration / 1000000) AS AvgDurationMs,
    max(Duration / 1000000) AS MaxDurationMs,
    min(Duration / 1000000) AS MinDurationMs
FROM otel.otel_traces
GROUP BY ServiceName, Hour;

-- Materialized view for endpoint statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS otel.endpoint_stats
ENGINE = SummingMergeTree()
ORDER BY (ServiceName, SpanName, toStartOfHour(Timestamp))
AS SELECT
    ServiceName,
    SpanName,
    toStartOfHour(Timestamp) AS Hour,
    count() AS CallCount,
    countIf(StatusCode = 'STATUS_CODE_ERROR') AS ErrorCount,
    avg(Duration / 1000000) AS AvgDurationMs,
    quantile(0.95)(Duration / 1000000) AS P95DurationMs,
    quantile(0.99)(Duration / 1000000) AS P99DurationMs
FROM otel.otel_traces
GROUP BY ServiceName, SpanName, Hour;

