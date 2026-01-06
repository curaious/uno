import { api } from '../../api';
import { getTrace, listTraces } from '../Traces/api';
import { Trace, TraceListItem } from '../Traces/types';
import { ConversationTraceItem, ConversationTracesData } from './types';
import {
  Conversation,
  ConversationMessage,
  EasyMessage,
  isEasyMessage,
  MessageUnion
} from "@curaious/uno-converse";

// Convert UUID to trace ID format (remove dashes)
const uuidToTraceId = (uuid: string): string => uuid.replace(/-/g, '');

// Get content preview from input messages
const getContentPreview = (input: MessageUnion[]): string => {
  if (!input || input.length === 0) return '';
  
  // Find the first user message
  for (const msg of input) {
    if (isEasyMessage(msg)) {
      const easyMsg = msg as EasyMessage;
      if (easyMsg.role === 'user') {
        const content = easyMsg.content;
        if (typeof content === 'string') {
          return content.substring(0, 100) + (content.length > 100 ? '...' : '');
        }
        if (Array.isArray(content) && content.length > 0) {
          const text = content.map(c => ('text' in c ? c.text : '')).join('');
          return text.substring(0, 100) + (text.length > 100 ? '...' : '');
        }
      }
    }
  }
  
  return 'No preview available';
};

// Get role from input messages
const getRole = (input: MessageUnion[]): string => {
  if (!input || input.length === 0) return 'unknown';
  
  for (const msg of input) {
    if (isEasyMessage(msg)) {
      return (msg as EasyMessage).role || 'unknown';
    }
  }
  
  return 'unknown';
};

export async function loadConversationTraces(conversationId: string, namespace: string = 'ns'): Promise<ConversationTracesData> {
  try {
    // First, get the conversation details
    const conversationsResponse = await api.get('/conversations', { params: { namespace } });
    const conversations: Conversation[] = conversationsResponse.data.data || [];
    const conversation = conversations.find(c => c.conversation_id === conversationId);
    
    // Get all threads for this conversation
    const threadsResponse = await api.get('/threads', { 
      params: { conversation_id: conversationId, namespace } 
    });
    const threads: any[] = threadsResponse.data.data || [];
    
    // Collect all messages from all threads
    const allMessages: ConversationTraceItem[] = [];
    
    for (const thread of threads) {
      const messagesResponse = await api.get('/messages', {
        params: { thread_id: thread.thread_id, namespace }
      });
      const messages: ConversationMessage[] = messagesResponse.data.data || [];
      
      for (const msg of messages) {
        const traceId = uuidToTraceId(msg.meta.run_state.traceid);
        
        // Try to get trace info
        let hasTrace = false;
        let trace: Trace | undefined;
        let durationMs: number | undefined;
        let spanCount: number | undefined;
        let hasErrors: boolean | undefined;
        let errorCount: number | undefined;
        
        try {
          const fetchedTrace = await getTrace(traceId);
          if (fetchedTrace) {
            hasTrace = true;
            trace = fetchedTrace;
            durationMs = fetchedTrace.duration_ms;
            spanCount = fetchedTrace.span_count;
            hasErrors = fetchedTrace.has_errors;
            errorCount = fetchedTrace.error_count;
          }
        } catch (e) {
          // Trace might not exist, that's okay
        }
        
        allMessages.push({
          message_id: msg.message_id,
          trace_id: traceId,
          role: '' as any,
          content_preview: '' as any,
          created_at: (msg.meta?.created_at as string) || '',
          has_trace: hasTrace,
          trace,
          duration_ms: durationMs,
          span_count: spanCount,
          has_errors: hasErrors,
          error_count: errorCount,
        });
      }
    }
    
    // Sort by created_at if available, otherwise keep original order
    allMessages.sort((a, b) => {
      if (a.created_at && b.created_at) {
        return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
      }
      return 0;
    });
    
    return {
      conversation_id: conversationId,
      conversation_name: conversation?.name || 'Unknown Conversation',
      messages: allMessages,
    };
  } catch (error) {
    console.error('Failed to load conversation traces:', error);
    throw error;
  }
}

export async function searchConversations(namespace: string = 'ns'): Promise<Conversation[]> {
  const response = await api.get('/conversations', { params: { namespace } });
  return response.data.data || [];
}

