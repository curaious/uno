import axios from 'axios';
import { Trace, TraceListResponse, TraceQueryParams, ServiceStats } from './types';

const tracesApi = axios.create({
  baseURL: '/api/agent-server',
  withCredentials: true,
});

export async function listTraces(params: TraceQueryParams = {}): Promise<TraceListResponse> {
  const queryParams = new URLSearchParams();
  
  if (params.service_name) queryParams.set('service_name', params.service_name);
  if (params.span_name) queryParams.set('span_name', params.span_name);
  if (params.trace_id) queryParams.set('trace_id', params.trace_id);
  if (params.min_duration !== undefined) queryParams.set('min_duration', params.min_duration.toString());
  if (params.max_duration !== undefined) queryParams.set('max_duration', params.max_duration.toString());
  if (params.has_errors !== undefined) queryParams.set('has_errors', params.has_errors.toString());
  if (params.start_time) queryParams.set('start_time', params.start_time);
  if (params.end_time) queryParams.set('end_time', params.end_time);
  if (params.limit) queryParams.set('limit', params.limit.toString());
  if (params.offset) queryParams.set('offset', params.offset.toString());

  const response = await tracesApi.get(`/traces?${queryParams.toString()}`);
  return response.data.data;
}

export async function getTrace(traceId: string): Promise<Trace> {
  const response = await tracesApi.get(`/traces/${traceId}`);
  return response.data.data;
}

export async function getServiceStats(startTime?: string, endTime?: string): Promise<ServiceStats[]> {
  const queryParams = new URLSearchParams();
  if (startTime) queryParams.set('start_time', startTime);
  if (endTime) queryParams.set('end_time', endTime);
  
  const response = await tracesApi.get(`/traces/stats/services?${queryParams.toString()}`);
  return response.data.data;
}

export async function getServices(): Promise<string[]> {
  const response = await tracesApi.get('/traces/services');
  return response.data.data;
}

