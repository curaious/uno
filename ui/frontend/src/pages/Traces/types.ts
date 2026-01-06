export interface Span {
  trace_id: string;
  span_id: string;
  parent_span_id?: string;
  name: string;
  kind: string;
  service_name: string;
  duration_ms: number;
  start_time: string;
  end_time: string;
  status_code: string;
  status_message?: string;
  attributes?: Record<string, string>;
  resource_attributes?: Record<string, string>;
  events?: SpanEvent[];
  children?: Span[];
}

export interface SpanEvent {
  timestamp: string;
  name: string;
  attributes?: Record<string, string>;
}

export interface Trace {
  trace_id: string;
  root_span?: Span;
  spans: Span[];
  span_count: number;
  duration_ms: number;
  start_time: string;
  end_time: string;
  services: string[];
  has_errors: boolean;
  error_count: number;
}

export interface TraceListItem {
  trace_id: string;
  root_span_name: string;
  service_name: string;
  duration_ms: number;
  span_count: number;  // uint64 from backend, but JS handles as number
  start_time: string;
  has_errors: boolean;
  error_count: number; // uint64 from backend
}

export interface ServiceStats {
  service_name: string;
  span_count: number;
  error_count: number;
  error_rate: number;
  avg_duration_ms: number;
  max_duration_ms: number;
  min_duration_ms: number;
  p50_duration_ms: number;
  p95_duration_ms: number;
  p99_duration_ms: number;
}

export interface TraceQueryParams {
  service_name?: string;
  span_name?: string;
  trace_id?: string;
  min_duration?: number;
  max_duration?: number;
  has_errors?: boolean;
  start_time?: string;
  end_time?: string;
  limit?: number;
  offset?: number;
}

export interface TraceListResponse {
  traces: TraceListItem[];
  total: number;
  limit: number;
  offset: number;
}

