import {useCallback, useEffect, useRef, useState} from 'react';
import {v4 as uuidv4} from 'uuid';
import {api} from '../../../api';
import {ChunkProcessor, ConversationMessage, ConverseConfig} from './streaming';
import {streamSSE} from './useSSEStream';
import {
  Conversation,
  MessageType,
  MessageUnion,
  Thread
} from '../types/types';

export interface UseConversationOptions {
  namespace: string;
  /** Auto-load conversations on mount */
  autoLoad?: boolean;
}

export interface UseConversationReturn {
  // Conversation list state
  conversations: Conversation[];
  conversationsLoading: boolean;
  
  // Thread state
  threads: Thread[];
  threadsLoading: boolean;
  currentThread: Thread | null;
  
  // Message state
  messages: ConversationMessage[];
  streamingMessage: ConversationMessage | null;
  messagesLoading: boolean;
  isStreaming: boolean;
  
  // Current selection
  currentConversationId: string | null;
  currentThreadId: string | null;

  // Actions - Conversations
  loadConversations: () => Promise<void>;
  selectConversation: (conversationId: string) => void;
  
  // Actions - Threads
  loadThreads: (conversationId: string) => Promise<void>;
  selectThread: (threadId: string) => void;
  
  // Actions - Messages
  sendMessage: (userMessages: MessageUnion[], config: ConverseConfig) => Promise<void>;
  
  // Actions - Utility
  startNewChat: () => void;
  
  // Combined messages (loaded + streaming)
  allMessages: ConversationMessage[];
}

/**
 * A comprehensive hook for managing conversations, threads, messages, and streaming.
 * Abstracts all chat-related data fetching and state management.
 */
export function useConversation(options: UseConversationOptions): UseConversationReturn {
  const { namespace, autoLoad = true } = options;

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
      const response = await api.get('/conversations', {
        params: { namespace },
      });
      setConversations(response.data.data || []);
    } catch (error) {
      console.error('Failed to load conversations:', error);
      throw error;
    } finally {
      setConversationsLoading(false);
    }
  }, [namespace]);

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
      const response = await api.get('/threads', {
        params: { conversation_id: conversationId, namespace },
      });
      const loadedThreads: Thread[] = response.data.data || [];
      
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
  }, [namespace]);

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
      const response = await api.get('/messages', {
        params: { namespace, thread_id: threadId },
      });
      
      const loadedMessages: ConversationMessage[] = response.data.data || [];
      
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
  }, [namespace]);

  /**
   * Send a user message and stream the response
   */
  const sendMessage = useCallback(async (userMessages: MessageUnion[], config: ConverseConfig) => {
    const messageId = uuidv4();

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
                // This removes the pending_tool_calls UI and appends the tool results
                setMessages(prev => {
                  const newMessages = [...prev];
                  if (newMessages.length > 0) {
                    const lastMsg = newMessages[newMessages.length - 1];
                    newMessages[newMessages.length - 1] = {
                      ...lastMsg,
                      messages: [...lastMsg.messages, ...finalConversation.messages],
                      meta: finalConversation.meta, // Update meta to clear pending_tool_calls
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
      if (previousMessageId === '') {
        try {
          const response = await api.get(`/messages/${processorRef.current.getConversation().message_id}`, {
            params: { namespace },
          });
          const newConversationId = response.data.data?.conversation_id;
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
  }, [currentConversationId, currentThreadId, previousMessageId, namespace]);

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
