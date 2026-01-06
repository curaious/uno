import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router';
import {
  CircularProgress,
  IconButton,
  Tooltip,
  Autocomplete,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  Timeline as TimelineIcon,
  Error as ErrorIcon,
  CheckCircle as CheckCircleIcon,
  ArrowBack as ArrowBackIcon,
  Search as SearchIcon,
  Chat as ChatIcon,
  ArrowForward as ArrowForwardIcon,
  OpenInNew as OpenInNewIcon,
  Launch as LaunchIcon,
} from '@mui/icons-material';
import { loadConversationTraces, searchConversations } from './api';
import { ConversationTraceItem, ConversationTracesData } from './types';
import { Input, InputLabel } from '../../components/shared/Input';
import { Button } from '../../components/shared/Buttons';
import styles from './ConversationTraces.module.css';
import { Conversation } from "@curaious/uno-converse";

export function ConversationTraces() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const initialConversationId = searchParams.get('conversation_id') || '';
  
  const [conversationId, setConversationId] = useState(initialConversationId);
  const [inputValue, setInputValue] = useState(initialConversationId);
  const [tracesData, setTracesData] = useState<ConversationTracesData | null>(null);
  const [loading, setLoading] = useState(false);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [loadingConversations, setLoadingConversations] = useState(true);

  // Load conversations for autocomplete
  useEffect(() => {
    const loadConversations = async () => {
      setLoadingConversations(true);
      try {
        const convos = await searchConversations();
        setConversations(convos);
      } catch (error) {
        console.error('Failed to load conversations:', error);
      } finally {
        setLoadingConversations(false);
      }
    };
    loadConversations();
  }, []);

  // Auto-load if conversation_id is in URL
  useEffect(() => {
    if (initialConversationId) {
      fetchTraces(initialConversationId);
    }
  }, [initialConversationId]);

  const fetchTraces = useCallback(async (convId: string) => {
    if (!convId) return;
    
    setLoading(true);
    try {
      const data = await loadConversationTraces(convId);
      setTracesData(data);
      setConversationId(convId);
      // Update URL
      navigate(`/agent-framework/conversation-traces?conversation_id=${convId}`, { replace: true });
    } catch (error) {
      console.error('Failed to load conversation traces:', error);
      setTracesData(null);
    } finally {
      setLoading(false);
    }
  }, [navigate]);

  const handleSearch = () => {
    if (inputValue) {
      fetchTraces(inputValue);
    }
  };

  const handleBack = () => {
    setTracesData(null);
    setConversationId('');
    setInputValue('');
    navigate('/agent-framework/conversation-traces', { replace: true });
  };

  const handleConversationSelect = (convId: string) => {
    setInputValue(convId);
    fetchTraces(convId);
  };

  const handleViewTrace = (traceId: string) => {
    // Include return_to parameter to navigate back to conversation traces
    const returnUrl = conversationId 
      ? `/agent-framework/conversation-traces?conversation_id=${conversationId}`
      : '/agent-framework/conversation-traces';
    navigate(`/agent-framework/traces/${traceId}?return_to=${encodeURIComponent(returnUrl)}`);
  };

  const formatDuration = (ms: number) => {
    if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`;
    if (ms < 1000) return `${ms.toFixed(1)}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  const getDurationClass = (ms: number) => {
    if (ms > 5000) return styles.durationVerySlow;
    if (ms > 1000) return styles.durationSlow;
    return styles.duration;
  };

  const getRoleBadgeClass = (role: string) => {
    switch (role.toLowerCase()) {
      case 'user':
        return styles.roleUser;
      case 'assistant':
        return styles.roleAssistant;
      case 'system':
        return styles.roleSystem;
      case 'tool':
        return styles.roleTool;
      default:
        return '';
    }
  };

  // Calculate stats
  const totalDuration = tracesData?.messages.reduce((sum, m) => sum + (m.duration_ms || 0), 0) || 0;
  const totalErrors = tracesData?.messages.reduce((sum, m) => sum + (m.error_count || 0), 0) || 0;
  const tracedMessages = tracesData?.messages.filter(m => m.has_trace).length || 0;

  return (
    <div className={styles.container}>
      {tracesData && (
        <div className={styles.backButton} onClick={handleBack}>
          <ArrowBackIcon sx={{ fontSize: 18 }} />
          Back to search
        </div>
      )}
      <div className={styles.header}>
        <div className={styles.title}>
          <TimelineIcon />
          Conversation Traces
        </div>
        {tracesData && (
          <Tooltip title="Refresh">
            <IconButton onClick={() => fetchTraces(conversationId)} disabled={loading}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        )}
      </div>

      {/* Search Section */}
      <div className={styles.searchSection}>
        <div className={styles.searchGroup}>
          <InputLabel className={styles.searchLabel}>Conversation ID</InputLabel>
          <Autocomplete
            freeSolo
            options={conversations}
            getOptionLabel={(option) => 
              typeof option === 'string' ? option : option.conversation_id
            }
            renderOption={(props, option) => (
              <li {...props} key={option.conversation_id}>
                <div>
                  <div style={{ fontWeight: 500, color: '#fff' }}>New Conversation</div>
                  <div style={{ fontSize: '0.75rem', color: '#666', fontFamily: 'monospace' }}>
                    {option.conversation_id}
                  </div>
                </div>
              </li>
            )}
            inputValue={inputValue}
            onInputChange={(_, value) => setInputValue(value)}
            onChange={(_, value) => {
              if (value && typeof value !== 'string') {
                handleConversationSelect(value.conversation_id);
              }
            }}
            loading={loadingConversations}
            renderInput={(params) => (
              <Input
                {...params}
                size="small"
                placeholder="Enter or select a conversation ID..."
                InputProps={{
                  ...params.InputProps,
                  startAdornment: <SearchIcon sx={{ color: '#666', mr: 1, fontSize: 18 }} />,
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    handleSearch();
                  }
                }}
              />
            )}
          />
        </div>
        <div className={styles.searchActions}>
          <Button
            className={styles.searchButton}
            onClick={handleSearch}
            disabled={!inputValue || loading}
          >
            {loading ? <CircularProgress size={18} color="inherit" /> : 'Search'}
          </Button>
        </div>
      </div>

      {/* Show conversations list when no conversation is selected */}
      {!tracesData && !loading && conversations.length > 0 && (
        <div className={styles.conversationsList}>
          <div className={styles.conversationsHeader}>
            <ChatIcon sx={{ fontSize: 18, mr: 1, verticalAlign: 'middle' }} />
            Recent Conversations
          </div>
          {conversations.slice(0, 10).map((conv) => (
            <div
              key={conv.conversation_id}
              className={styles.conversationItem}
              onClick={() => handleConversationSelect(conv.conversation_id)}
            >
              <div>
                <div className={styles.conversationItemName}>New Conversation</div>
                <div className={styles.conversationItemId}>{conv.conversation_id}</div>
              </div>
              <ArrowForwardIcon className={styles.conversationItemArrow} />
            </div>
          ))}
        </div>
      )}

      {/* Loading State */}
      {loading && (
        <div className={styles.loading}>
          <CircularProgress />
        </div>
      )}

      {/* Conversation Info & Messages */}
      {tracesData && !loading && (
        <>
          <div className={styles.conversationInfo}>
            <div className={styles.conversationHeader}>
              <div className={styles.conversationName}>
                <ChatIcon sx={{ fontSize: 20 }} />
                {tracesData.conversation_name}
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <div className={styles.conversationId}>{tracesData.conversation_id}</div>
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => navigate(`/agent-framework/chat?conversation_id=${tracesData.conversation_id}`)}
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px',
                    fontSize: '0.8rem',
                    padding: '4px 12px',
                    textTransform: 'none',
                  }}
                >
                  <LaunchIcon sx={{ fontSize: 16 }} />
                  Open Conversation
                </Button>
              </div>
            </div>
            <div className={styles.statsRow}>
              <div className={styles.statItem}>
                Messages: <span className={styles.statValue}>{tracesData.messages.length}</span>
              </div>
              <div className={styles.statItem}>
                Traced: <span className={styles.statValue}>{tracedMessages}</span>
              </div>
              <div className={styles.statItem}>
                Total Duration: <span className={styles.statValue}>{formatDuration(totalDuration)}</span>
              </div>
              {totalErrors > 0 && (
                <div className={styles.statItem}>
                  Errors: <span className={styles.statValue} style={{ color: '#ff6666' }}>{totalErrors}</span>
                </div>
              )}
            </div>
          </div>

          <div className={styles.messagesList}>
            {tracesData.messages.length === 0 ? (
              <div className={styles.emptyState}>
                <div className={styles.emptyIcon}>📭</div>
                <div>No messages found in this conversation</div>
              </div>
            ) : (
              <table className={styles.messagesTable}>
                <thead>
                  <tr>
                    <th>#</th>
                    <th>Trace ID</th>
                    <th>Content</th>
                    <th>Duration</th>
                    <th>Spans</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {tracesData.messages.map((msg, index) => (
                    <tr key={msg.message_id}>
                      <td>
                        <span className={styles.messageIndex}>{index + 1}</span>
                      </td>
                      <td>
                        {msg.has_trace ? (
                          <span
                            className={styles.traceId}
                            onClick={() => handleViewTrace(msg.trace_id)}
                            style={{ cursor: 'pointer' }}
                          >
                            {msg.trace_id}
                          </span>
                        ) : (
                          <span className={styles.noTrace}>No trace</span>
                        )}
                      </td>
                      <td>
                        <span className={styles.contentPreview}>{msg.content_preview}</span>
                      </td>
                      <td>
                        {msg.has_trace && msg.duration_ms !== undefined ? (
                          <span className={getDurationClass(msg.duration_ms)}>
                            {formatDuration(msg.duration_ms)}
                          </span>
                        ) : (
                          <span className={styles.duration}>-</span>
                        )}
                      </td>
                      <td>
                        {msg.has_trace ? (
                          <span className={styles.spanCount}>{msg.span_count}</span>
                        ) : (
                          <span className={styles.spanCount}>-</span>
                        )}
                      </td>
                      <td>
                        {msg.has_trace ? (
                          msg.has_errors ? (
                            <span className={styles.errorBadge}>
                              <ErrorIcon sx={{ fontSize: 14 }} />
                              {msg.error_count} error{msg.error_count && msg.error_count > 1 ? 's' : ''}
                            </span>
                          ) : (
                            <span className={styles.successBadge}>
                              <CheckCircleIcon sx={{ fontSize: 14 }} />
                              OK
                            </span>
                          )
                        ) : (
                          <span className={styles.noTrace}>-</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </>
      )}
    </div>
  );
}

