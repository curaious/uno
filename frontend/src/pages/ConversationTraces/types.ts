import { Trace } from '../Traces/types';

export interface ConversationTraceItem {
  message_id: string;
  trace_id: string; // message_id without dashes
  role: string;
  content_preview: string;
  created_at: string;
  trace?: Trace;
  has_trace: boolean;
  duration_ms?: number;
  span_count?: number;
  has_errors?: boolean;
  error_count?: number;
}

export interface ConversationTracesData {
  conversation_id: string;
  conversation_name: string;
  messages: ConversationTraceItem[];
}


