/**
 * Wrapper hook for useConversation that provides the API client
 * configured with the frontend's API settings.
 */
import { useMemo } from 'react';
import {
  useConversation,
  UseConversationOptions,
  UseConversationReturn,
  UnoApiClient,
  Conversation,
  Thread,
  ConversationMessage,
} from '@praveen001/uno-converse';
import { api } from '../api';
import { STORAGE_KEY } from '../contexts/ProjectContext';

/**
 * Creates an API client using the frontend's axios instance
 */
function createFrontendApiClient(): UnoApiClient {
  const getProjectId = (): string | null => {
    try {
      return localStorage.getItem(STORAGE_KEY);
    } catch {
      return null;
    }
  };

  return {
    baseUrl: api.defaults.baseURL || '',

    async getConversations(namespace: string): Promise<Conversation[]> {
      const response = await api.get('/conversations', {
        params: { namespace },
      });
      return response.data.data || [];
    },

    async getThreads(conversationId: string, namespace: string): Promise<Thread[]> {
      const response = await api.get('/threads', {
        params: { conversation_id: conversationId, namespace },
      });
      return response.data.data || [];
    },

    async getMessages(threadId: string, namespace: string): Promise<ConversationMessage[]> {
      const response = await api.get('/messages', {
        params: { thread_id: threadId, namespace },
      });
      return response.data.data || [];
    },

    async getMessage(messageId: string, namespace: string): Promise<ConversationMessage | null> {
      try {
        const response = await api.get(`/messages/${messageId}`, {
          params: { namespace },
        });
        return response.data.data || null;
      } catch {
        return null;
      }
    },
  };
}

export interface UseChatOptions {
  namespace: string;
  autoLoad?: boolean;
}

/**
 * Custom hook that wraps useConversation with the frontend's API client.
 * This is the hook that frontend components should use.
 */
export function useChat(options: UseChatOptions): UseConversationReturn {
  const { namespace, autoLoad = true } = options;

  // Create the API client - memoized to avoid recreating on every render
  const client = useMemo(() => createFrontendApiClient(), []);

  return useConversation({
    namespace,
    client,
    autoLoad,
  });
}

// Re-export types from the package for convenience
export type { UseConversationReturn } from '@praveen001/uno-converse';

