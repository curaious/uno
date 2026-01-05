import { useCallback, useEffect, useRef, useState } from 'react';
import { ChunkProcessor } from '../streaming/ChunkProcessor';
import { streamSSE } from '../streaming/streamSSE';
import {
  Conversation,
  ConversationMessage,
  ConverseConfig,
  MessageType,
  MessageUnion,
  Thread,
} from '../types';

/**
 * Simple ID generator for message IDs
 */
function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
}

/**
 * API client interface for Uno Agent Server.
 * Implement this interface to connect to your Uno server.
 */
export interface UnoApiClient {
  /**
   * Base URL of the Uno Agent Server (e.g., 'https://api.example.com')
   */
  baseUrl: string;

  /**
   * Fetch conversations for a namespace
   */
  getConversations(namespace: string): Promise<Conversation[]>;

  /**
   * Fetch threads for a conversation
   */
  getThreads(conversationId: string, namespace: string): Promise<Thread[]>;

  /**
   * Fetch messages for a thread
   */
  getMessages(threadId: string, namespace: string): Promise<ConversationMessage[]>;

  /**
   * Get message details by ID
   */
  getMessage(messageId: string, namespace: string): Promise<ConversationMessage | null>;

  /**
   * Optional headers to include in streaming requests
   */
  headers?: Record<string, string>;
}

/**
 * Creates a default API client using fetch.
 * This is a convenience function for simple setups.
 *
 * @example
 * ```ts
 * const client = createApiClient({
 *   baseUrl: 'https://my-uno-server.com/api/agent-server',
 *   headers: { 'Authorization': 'Bearer my-token' }
 * });
 * ```
 */
export function createApiClient(options: {
  baseUrl: string;
  headers?: Record<string, string>;
  projectId?: string;
}): UnoApiClient {
  const { baseUrl, headers = {}, projectId } = options;

  async function fetchJson<T>(endpoint: string, params?: Record<string, string>): Promise<T> {
    const url = new URL(`${baseUrl}${endpoint}`);
    if (projectId) {
      url.searchParams.set('project_id', projectId);
    }
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        url.searchParams.set(key, value);
      });
    }

    const response = await fetch(url.toString(), {
      headers: {
        'Content-Type': 'application/json',
        ...headers,
      },
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const data = await response.json();
    return data.data ?? data;
  }

  return {
    baseUrl,
    headers,

    async getConversations(namespace: string): Promise<Conversation[]> {
      return fetchJson<Conversation[]>('/conversations', { namespace });
    },

    async getThreads(conversationId: string, namespace: string): Promise<Thread[]> {
      return fetchJson<Thread[]>('/threads', {
        conversation_id: conversationId,
        namespace,
      });
    },

    async getMessages(threadId: string, namespace: string): Promise<ConversationMessage[]> {
      return fetchJson<ConversationMessage[]>('/messages', {
        thread_id: threadId,
        namespace,
      });
    },

    async getMessage(messageId: string, namespace: string): Promise<ConversationMessage | null> {
      try {
        return await fetchJson<ConversationMessage>(`/messages/${messageId}`, { namespace });
      } catch {
        return null;
      }
    },
  };
}

/**
 * Options for the useConversation hook
 */
export interface UseConversationOptions {
  /** The namespace for conversations */
  namespace: string;
  /** API client for communicating with Uno Agent Server */
  client: UnoApiClient;
  /** Auto-load conversations on mount (default: true) */
  autoLoad?: boolean;
}

/**
 * Return type for the useConversation hook
 */
export interface UseConversationReturn {
  // Conversation list state
  /** List of all conversations */
  conversations: Conversation[];
  /** Whether conversations are being loaded */
  conversationsLoading: boolean;

  // Thread state
  /** List of threads in the current conversation */
  threads: Thread[];
  /** Whether threads are being loaded */
  threadsLoading: boolean;
  /** Currently selected thread */
  currentThread: Thread | null;

  // Message state
  /** List of messages in the current thread */
  messages: ConversationMessage[];
  /** Message currently being streamed */
  streamingMessage: ConversationMessage | null;
  /** Whether messages are being loaded */
  messagesLoading: boolean;
  /** Whether a response is currently streaming */
  isStreaming: boolean;

  // Current selection
  /** ID of the currently selected conversation */
  currentConversationId: string | null;
  /** ID of the currently selected thread */
  currentThreadId: string | null;

  // Actions - Conversations
  /** Load all conversations */
  loadConversations: () => Promise<void>;
  /** Select a conversation by ID */
  selectConversation: (conversationId: string) => void;

  // Actions - Threads
  /** Load threads for a conversation */
  loadThreads: (conversationId: string) => Promise<void>;
  /** Select a thread by ID */
  selectThread: (threadId: string) => void;

  // Actions - Messages
  /** Send a message and stream the response */
  sendMessage: (userMessages: MessageUnion[], config: ConverseConfig) => Promise<void>;

  // Actions - Utility
  /** Start a new chat (clears current state) */
  startNewChat: () => void;

  // Combined messages (loaded + streaming)
  /** All messages including the currently streaming one */
  allMessages: ConversationMessage[];
}

/**
 * A comprehensive hook for managing conversations, threads, messages, and streaming
 * with Uno Agent Server.
 *
 * @example
 * ```tsx
 * import { useConversation, createApiClient } from '@praveen001/uno-converse';
 *
 * const client = createApiClient({
 *   baseUrl: 'https://my-uno-server.com/api/agent-server',
 * });
 *
 * function ChatComponent() {
 *   const {
 *     allMessages,
 *     isStreaming,
 *     sendMessage,
 *     startNewChat,
 *   } = useConversation({
 *     namespace: 'my-app',
 *     client,
 *   });
 *
 *   const handleSend = async (text: string) => {
 *     await sendMessage(
 *       [{ type: 'message', id: '1', content: text }],
 *       {
 *         baseUrl: client.baseUrl,
 *         namespace: 'my-app',
 *         agentName: 'my-agent',
 *       }
 *     );
 *   };
 *
 *   return (
 *     <div>
 *       {allMessages.map(msg => (
 *         <MessageComponent key={msg.message_id} message={msg} />
 *       ))}
 *       {isStreaming && <LoadingIndicator />}
 *     </div>
 *   );
 * }
 * ```
 */
export function useConversation(options: UseConversationOptions): UseConversationReturn {
  const { namespace, client, autoLoad = true } = options;

  // Conversation list state
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [conversationsLoading, setConversationsLoading] = useState(false);

  // Thread state
  const [threads, setThreads] = useState<Thread[]>([]);
  const [threadsLoading, setThreadsLoading] = useState(false);

  // Message state
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [streamingMessage, setStreamingMessage] = useState<ConversationMessage | null>(null);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [isStreaming, setIsStreaming] = useState(false);

  // Current selection
  const [currentConversationId, setCurrentConversationId] = useState<string | null>(null);
  const [currentThreadId, setCurrentThreadId] = useState<string | null>(null);
  const [previousMessageId, setPreviousMessageId] = useState<string>('');

  // Refs
  const processorRef = useRef<ChunkProcessor | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  // ============================================
  // Conversation Management
  // ============================================

  /**
   * Load all conversations for the namespace
   */
  const loadConversations = useCallback(async () => {
    setConversationsLoading(true);
    try {
      const data = await client.getConversations(namespace);
      setConversations(data || []);
    } catch (error) {
      console.error('Failed to load conversations:', error);
      throw error;
    } finally {
      setConversationsLoading(false);
    }
  }, [client, namespace]);

  /**
   * Select a conversation and load its threads
   */
  const selectConversation = useCallback((conversationId: string) => {
    setCurrentConversationId(conversationId);
    // Threads will be loaded via useEffect
  }, []);

  // ============================================
  // Thread Management
  // ============================================

  /**
   * Load threads for a conversation
   */
  const loadThreads = useCallback(async (conversationId: string) => {
    setThreadsLoading(true);
    try {
      const loadedThreads = await client.getThreads(conversationId, namespace);

      // Sort by created_at descending
      loadedThreads.sort(
        (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      );

      setThreads(loadedThreads);

      // Auto-select the latest thread
      if (loadedThreads.length > 0) {
        setCurrentThreadId(loadedThreads[0].thread_id);
      }
    } catch (error) {
      console.error('Failed to load threads:', error);
      throw error;
    } finally {
      setThreadsLoading(false);
    }
  }, [client, namespace]);

  /**
   * Select a thread and load its messages
   */
  const selectThread = useCallback((threadId: string) => {
    setCurrentThreadId(threadId);
    // Messages will be loaded via useEffect
  }, []);

  // ============================================
  // Message Management
  // ============================================

  /**
   * Load messages for a thread
   */
  const loadMessages = useCallback(async (threadId: string) => {
    setMessagesLoading(true);
    try {
      const loadedMessages = await client.getMessages(threadId, namespace);

      // Extract last message ID for continuation
      if (loadedMessages.length > 0) {
        const lastMsgId = loadedMessages[loadedMessages.length - 1].message_id;
        setPreviousMessageId(lastMsgId);
      } else {
        setPreviousMessageId('');
      }

      setMessages(loadedMessages);
    } catch (error) {
      console.error('Failed to load messages:', error);
      throw error;
    } finally {
      setMessagesLoading(false);
    }
  }, [client, namespace]);

  /**
   * Send a user message and stream the response
   */
  const sendMessage = useCallback(async (userMessages: MessageUnion[], config: ConverseConfig) => {
    const messageId = generateId();

    // Check if this is a tool approval response (resuming a run)
    const isToolApproval = userMessages.length === 1 &&
      userMessages[0].type === MessageType.FunctionCallApprovalResponse;

    // Only add user message for regular messages, not for tool approvals
    if (!isToolApproval) {
      const userConversation: ConversationMessage = {
        conversation_id: currentConversationId || '',
        thread_id: currentThreadId || '',
        message_id: messageId + '-user',
        messages: userMessages,
        meta: {},
      };
      setMessages(prev => [...prev, userConversation]);
    }

    // Initialize the chunk processor for the assistant response
    processorRef.current = new ChunkProcessor(
      currentConversationId || '',
      currentThreadId || '',
      (conversation) => {
        setStreamingMessage({ ...conversation, isStreaming: true });
      }
    );

    setIsStreaming(true);

    // Build URL with query parameters
    const params = new URLSearchParams();
    if (config.projectId) {
      params.append('project_id', config.projectId);
    }
    params.append('agent_name', config.agentName);

    const url = `${config.baseUrl}/converse?${params.toString()}`;

    // Prepare request body
    const body = JSON.stringify({
      namespace: config.namespace,
      previous_message_id: previousMessageId,
      message: userMessages[0],
      context: config.context || {},
    });

    // Abort any existing stream
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    abortControllerRef.current = new AbortController();

    try {
      await streamSSE(
        url,
        {
          method: 'POST',
          body,
          headers: {
            'Content-Type': 'application/json',
            ...(config.headers || {}),
            ...(client.headers || {}),
          },
        },
        {
          onChunk: (data) => {
            processorRef.current?.processChunk(data);
          },
          onComplete: () => {
            // Move streaming message to messages list
            if (processorRef.current) {
              const finalConversation = processorRef.current.getConversation();

              if (isToolApproval) {
                // For tool approvals, update the last message instead of appending
                setMessages(prev => {
                  const newMessages = [...prev];
                  if (newMessages.length > 0) {
                    const lastMsg = newMessages[newMessages.length - 1];
                    newMessages[newMessages.length - 1] = {
                      ...lastMsg,
                      messages: [...lastMsg.messages, ...finalConversation.messages],
                      meta: finalConversation.meta,
                      isStreaming: false,
                    };
                  }
                  return newMessages;
                });
              } else {
                setMessages(prev => [...prev, { ...finalConversation, isStreaming: false }]);
              }

              setStreamingMessage(null);
              setPreviousMessageId(finalConversation.message_id);
            }
          },
          onError: (error) => {
            console.error('Streaming error:', error);
            setStreamingMessage(null);
          },
        },
        abortControllerRef.current.signal
      );

      // If this was a new conversation, fetch the conversation info
      if (previousMessageId === '' && processorRef.current) {
        try {
          const messageData = await client.getMessage(
            processorRef.current.getConversation().message_id,
            namespace
          );
          const newConversationId = messageData?.conversation_id;
          if (newConversationId) {
            // Add new conversation to list
            setConversations(prev => [{
              conversation_id: newConversationId,
              name: "New Conversation",
              namespace_id: namespace,
              created_at: new Date().toISOString(),
              last_updated: new Date().toISOString(),
            } as Conversation, ...prev.filter(c => c.conversation_id !== newConversationId)]);

            setCurrentConversationId(newConversationId);
          }
        } catch (e) {
          console.error('Failed to get conversation ID:', e);
        }
      }
    } finally {
      setIsStreaming(false);
      abortControllerRef.current = null;
    }
  }, [currentConversationId, currentThreadId, previousMessageId, namespace, client]);

  // ============================================
  // Utility Actions
  // ============================================

  /**
   * Start a new chat (reset all state)
   */
  const startNewChat = useCallback(() => {
    // Abort any ongoing stream
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    setCurrentConversationId(null);
    setCurrentThreadId(null);
    setThreads([]);
    setMessages([]);
    setStreamingMessage(null);
    setPreviousMessageId('');
    setIsStreaming(false);
    processorRef.current = null;
  }, []);

  // ============================================
  // Effects for auto-loading
  // ============================================

  // Auto-load conversations on mount
  useEffect(() => {
    if (autoLoad) {
      loadConversations();
    }
  }, [autoLoad, loadConversations]);

  // Load threads when conversation changes
  useEffect(() => {
    if (currentConversationId) {
      loadThreads(currentConversationId);
    }
  }, [currentConversationId, loadThreads]);

  // Load messages when thread changes
  useEffect(() => {
    if (currentThreadId) {
      loadMessages(currentThreadId);
    }
  }, [currentThreadId, loadMessages]);

  // ============================================
  // Computed values
  // ============================================

  const currentThread = threads.find(t => t.thread_id === currentThreadId) || null;

  const allMessages = streamingMessage
    ? [...messages, streamingMessage]
    : messages;

  return {
    // Conversation list state
    conversations,
    conversationsLoading,

    // Thread state
    threads,
    threadsLoading,
    currentThread,

    // Message state
    messages,
    streamingMessage,
    messagesLoading,
    isStreaming,

    // Current selection
    currentConversationId,
    currentThreadId,

    // Actions - Conversations
    loadConversations,
    selectConversation,

    // Actions - Threads
    loadThreads,
    selectThread,

    // Actions - Messages
    sendMessage,

    // Actions - Utility
    startNewChat,

    // Combined messages
    allMessages,
  };
}

