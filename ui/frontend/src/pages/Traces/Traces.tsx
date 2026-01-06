import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, useSearchParams, useLocation } from 'react-router';
import {
  CircularProgress,
  MenuItem,
  IconButton,
  Tooltip,
  Chip,
  Box,
  Typography,
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  Timeline as TimelineIcon,
  Error as ErrorIcon,
  CheckCircle as CheckCircleIcon,
  ArrowBack as ArrowBackIcon,
  Search as SearchIcon,
  FilterList as FilterIcon,
} from '@mui/icons-material';
import { listTraces, getTrace, getServiceStats, getServices } from './api';
import { TraceListItem, Trace, Span, ServiceStats, TraceQueryParams } from './types';
import { TraceWaterfall } from './TraceWaterfall';
import { SlideDialog } from '../../components/shared/Dialog';
import { Input, Select, InputLabel } from '../../components/shared/Input';
import { Button } from '../../components/shared/Buttons';
import { SpanDetailsPanel } from '../../components/Traces/SpanDetailsPanel';
import { OpenAIIcon } from '../../Icons/OpenAI';
import { AnthropicIcon } from '../../Icons/Anthropic';
import { GeminiIcon } from '../../Icons/Gemini';
import { XAIIcon } from '../../Icons/XAI';
import styles from './Traces.module.css';

export function Traces() {
  const { traceId: urlTraceId } = useParams<{ traceId?: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const location = useLocation();
  
  // Check if we're in gateway mode (route starts with /gateway/traces)
  const isGatewayMode = location.pathname.startsWith('/gateway/traces');
  
  // Provider icons mapping
  const providerIcons: Record<string, React.ReactElement> = {
    'OpenAI': <OpenAIIcon />,
    'openai': <OpenAIIcon />,
    'Anthropic': <AnthropicIcon />,
    'anthropic': <AnthropicIcon />,
    'Gemini': <GeminiIcon />,
    'gemini': <GeminiIcon />,
    'xAI': <XAIIcon />,
    'xai': <XAIIcon />,
  };
  
  const getProviderIcon = (providerName?: string) => {
    if (!providerName) return null;
    // Normalize provider name (handle various formats)
    const normalized = providerName.trim();
    return providerIcons[normalized] || providerIcons[normalized.charAt(0).toUpperCase() + normalized.slice(1).toLowerCase()] || null;
  };
  
  const [traces, setTraces] = useState<TraceListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [services, setServices] = useState<string[]>([]);
  const [stats, setStats] = useState<ServiceStats[]>([]);
  const [selectedTrace, setSelectedTrace] = useState<Trace | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [selectedSpan, setSelectedSpan] = useState<Span | null>(null);
  const [spanDialogOpen, setSpanDialogOpen] = useState(false);
  // Cache for trace details (provider/model) in gateway mode
  const [traceDetailsCache, setTraceDetailsCache] = useState<Map<string, { provider?: string; model?: string }>>(new Map());
  
  // Filters
  const [serviceFilter, setServiceFilter] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [timeRange, setTimeRange] = useState('1h');
  const [page, setPage] = useState(0);
  const limit = 50;
  
  // Load trace from URL parameter if present
  useEffect(() => {
    if (urlTraceId && !selectedTrace) {
      handleTraceClick(urlTraceId);
    }
  }, [urlTraceId]);

  const getTimeRangeParams = useCallback(() => {
    const end = new Date();
    let start = new Date();
    
    switch (timeRange) {
      case '15m':
        start.setMinutes(end.getMinutes() - 15);
        break;
      case '1h':
        start.setHours(end.getHours() - 1);
        break;
      case '6h':
        start.setHours(end.getHours() - 6);
        break;
      case '24h':
        start.setDate(end.getDate() - 1);
        break;
      case '7d':
        start.setDate(end.getDate() - 7);
        break;
      default:
        start.setHours(end.getHours() - 1);
    }
    
    return {
      start_time: start.toISOString(),
      end_time: end.toISOString(),
    };
  }, [timeRange]);

  const fetchTraces = useCallback(async () => {
    setLoading(true);
    try {
      const timeParams = getTimeRangeParams();
      const params: TraceQueryParams = {
        ...timeParams,
        limit,
        offset: page * limit,
      };
      
      if (serviceFilter) params.service_name = serviceFilter;
      if (searchQuery) params.trace_id = searchQuery;
      
      // In gateway mode, filter by span_name starting with "LLM." (matches any span, not just root)
      if (isGatewayMode) {
        params.span_name = 'LLM.';
      }
      
      const response = await listTraces(params);
      
      // No frontend filtering needed - backend returns traces containing LLM spans
      setTraces(response.traces || []);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to fetch traces:', error);
      setTraces([]);
    } finally {
      setLoading(false);
    }
  }, [serviceFilter, searchQuery, timeRange, page, getTimeRangeParams, isGatewayMode]);

  const fetchStats = useCallback(async () => {
    try {
      const timeParams = getTimeRangeParams();
      const [statsData, servicesData] = await Promise.all([
        getServiceStats(timeParams.start_time, timeParams.end_time),
        getServices(),
      ]);
      setStats(statsData || []);
      setServices(servicesData || []);
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    }
  }, [getTimeRangeParams]);

  useEffect(() => {
    fetchTraces();
    fetchStats();
  }, [fetchTraces, fetchStats]);

  // Fetch trace details for gateway mode
  useEffect(() => {
    if (!isGatewayMode || traces.length === 0) return;
    
    const fetchTraceDetails = async () => {
      setTraceDetailsCache(currentCache => {
        const tracesToFetch = traces.filter(trace => !currentCache.has(trace.trace_id));
        
        if (tracesToFetch.length === 0) {
          return currentCache;
        }
        
        // Fetch details for traces not in cache
        const detailsPromises = tracesToFetch.map(async (trace) => {
          try {
            const fullTrace = await getTrace(trace.trace_id);
            // Find the LLM span (could be root or any child span)
            const findLLMSpan = (spans: Span[]): Span | null => {
              for (const span of spans) {
                if (span.name.startsWith('LLM.')) {
                  return span;
                }
                if (span.children) {
                  const found = findLLMSpan(span.children);
                  if (found) return found;
                }
              }
              return null;
            };
            
            const llmSpan = findLLMSpan(fullTrace.spans);
            if (llmSpan?.attributes) {
              return {
                traceId: trace.trace_id,
                provider: llmSpan.attributes['llm.provider'] || llmSpan.attributes['llm_provider'],
                model: llmSpan.attributes['llm.model'] || llmSpan.attributes['llm_model'],
              };
            }
          } catch (e) {
            // Ignore errors
          }
          return null;
        });
        
        Promise.all(detailsPromises).then(details => {
          setTraceDetailsCache(prevCache => {
            const updatedCache = new Map(prevCache);
            details.forEach(detail => {
              if (detail && (detail.provider || detail.model)) {
                updatedCache.set(detail.traceId, { provider: detail.provider, model: detail.model });
              }
            });
            return updatedCache;
          });
        });
        
        return currentCache;
      });
    };
    
    fetchTraceDetails();
  }, [traces, isGatewayMode]);

  const handleTraceClick = async (traceId: string, updateUrl = true) => {
    // In gateway mode, directly open span details dialog
    if (isGatewayMode) {
      setLoadingDetail(true);
      try {
        const trace = await getTrace(traceId);
        // Find the LLM span (could be root or any child span)
        const findLLMSpan = (spans: Span[]): Span | null => {
          for (const span of spans) {
            if (span.name.startsWith('LLM.')) {
              return span;
            }
            if (span.children) {
              const found = findLLMSpan(span.children);
              if (found) return found;
            }
          }
          return null;
        };
        
        const llmSpan = findLLMSpan(trace.spans);
        if (llmSpan) {
          setSelectedSpan(llmSpan);
          setSpanDialogOpen(true);
        } else if (trace.root_span) {
          // Fallback to root span if no LLM span found
          setSelectedSpan(trace.root_span);
          setSpanDialogOpen(true);
        }
      } catch (error) {
        console.error('Failed to fetch trace:', error);
      } finally {
        setLoadingDetail(false);
      }
      return;
    }
    
    // Normal mode: show full trace detail view
    setLoadingDetail(true);
    try {
      const trace = await getTrace(traceId);
      setSelectedTrace(trace);
      // Update URL to include trace ID, preserving query parameters
      if (updateUrl) {
        const returnTo = searchParams.get('return_to');
        const basePath = isGatewayMode ? '/gateway/traces' : '/agent-framework/traces';
        const url = returnTo 
          ? `${basePath}/${traceId}?return_to=${encodeURIComponent(returnTo)}`
          : `${basePath}/${traceId}`;
        navigate(url, { replace: true });
      }
    } catch (error) {
      console.error('Failed to fetch trace:', error);
    } finally {
      setLoadingDetail(false);
    }
  };
  
  const handleBackToList = () => {
    setSelectedTrace(null);
    // Check if we came from conversation traces
    const returnTo = searchParams.get('return_to');
    if (returnTo) {
      navigate(decodeURIComponent(returnTo), { replace: true });
    } else {
      // Navigate back to the appropriate traces page based on current route
      const backPath = isGatewayMode ? '/gateway/traces' : '/agent-framework/traces';
      navigate(backPath, { replace: true });
    }
  };

  const formatDuration = (ms: number) => {
    if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`;
    if (ms < 1000) return `${ms.toFixed(1)}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleString();
  };

  const getDurationClass = (ms: number) => {
    if (ms > 5000) return styles.durationVerySlow;
    if (ms > 1000) return styles.durationSlow;
    return styles.duration;
  };

  // Detail View
  if (selectedTrace) {
    const returnTo = searchParams.get('return_to');
    let backButtonText = 'Back to traces';
    if (returnTo) {
      backButtonText = 'Back to conversation traces';
    } else if (isGatewayMode) {
      backButtonText = 'Back to LLM traces';
    }
    
    return (
      <TraceDetailView
        trace={selectedTrace}
        onBack={handleBackToList}
        formatDuration={formatDuration}
        formatTimestamp={formatTimestamp}
        backButtonText={backButtonText}
      />
    );
  }

  // Stats Summary
  const totalSpans = stats.reduce((sum, s) => sum + s.span_count, 0);
  const totalErrors = stats.reduce((sum, s) => sum + s.error_count, 0);
  const avgDuration = stats.length > 0 
    ? stats.reduce((sum, s) => sum + s.avg_duration_ms, 0) / stats.length 
    : 0;

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div className={styles.title}>
          <TimelineIcon />
          {isGatewayMode ? 'LLM Traces' : 'Traces'}
        </div>
        <Tooltip title="Refresh">
          <IconButton onClick={fetchTraces} disabled={loading}>
            <RefreshIcon />
          </IconButton>
        </Tooltip>
      </div>

      {/* Stats Cards */}
      <div className={styles.statsGrid}>
        <div className={styles.statCard}>
          <div className={styles.statHeader}>
            <span className={styles.statTitle}>Total Spans</span>
          </div>
          <div className={styles.statValue}>{totalSpans.toLocaleString()}</div>
          <div className={styles.statSubtext}>in selected time range</div>
        </div>
        <div className={styles.statCard}>
          <div className={styles.statHeader}>
            <span className={styles.statTitle}>Error Rate</span>
          </div>
          <div className={styles.statValue}>
            {totalSpans > 0 ? ((totalErrors / totalSpans) * 100).toFixed(2) : 0}%
          </div>
          <div className={styles.statSubtext}>{totalErrors} errors</div>
        </div>
        <div className={styles.statCard}>
          <div className={styles.statHeader}>
            <span className={styles.statTitle}>Avg Duration</span>
          </div>
          <div className={styles.statValue}>{formatDuration(avgDuration)}</div>
          <div className={styles.statSubtext}>across all services</div>
        </div>
        <div className={styles.statCard}>
          <div className={styles.statHeader}>
            <span className={styles.statTitle}>Services</span>
          </div>
          <div className={styles.statValue}>{stats.length}</div>
          <div className={styles.statSubtext}>active services</div>
        </div>
      </div>

      {/* Filters */}
      <div className={styles.filters}>
        <div className={styles.filterGroup}>
          <InputLabel className={styles.filterLabel}>Search</InputLabel>
          <Input
            size="small"
            placeholder="Search by trace ID..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            InputProps={{
              startAdornment: <SearchIcon sx={{ color: '#666', mr: 1, fontSize: 18 }} />,
            }}
          />
        </div>
        <div className={styles.filterGroup}>
          <InputLabel className={styles.filterLabel}>Service</InputLabel>
          <Select
            size="small"
            value={serviceFilter}
            displayEmpty
            onChange={(e) => setServiceFilter(e.target.value as string)}
          >
            <MenuItem value="">All Services</MenuItem>
            {services.map((service) => (
              <MenuItem key={service} value={service}>{service}</MenuItem>
            ))}
          </Select>
        </div>
        <div className={styles.filterGroup}>
          <InputLabel className={styles.filterLabel}>Time Range</InputLabel>
          <Select
            size="small"
            value={timeRange}
            onChange={(e) => setTimeRange(e.target.value as string)}
          >
            <MenuItem value="15m">Last 15 min</MenuItem>
            <MenuItem value="1h">Last 1 hour</MenuItem>
            <MenuItem value="6h">Last 6 hours</MenuItem>
            <MenuItem value="24h">Last 24 hours</MenuItem>
            <MenuItem value="7d">Last 7 days</MenuItem>
          </Select>
        </div>
        <div className={styles.filterActions}>
          <Button
            className={styles.clearButton}
            onClick={() => {
              setServiceFilter('');
              setSearchQuery('');
              setPage(0);
            }}
          >
            Clear
          </Button>
          <Button
            className={styles.applyButton}
            onClick={fetchTraces}
          >
            <FilterIcon sx={{ fontSize: 16, mr: 0.5 }} />
            Apply
          </Button>
        </div>
      </div>

      {/* Traces List */}
      <div className={styles.tracesList}>
        {loading ? (
          <div className={styles.loading}>
            <CircularProgress />
          </div>
        ) : traces.length === 0 ? (
          <div className={styles.emptyState}>
            <div className={styles.emptyIcon}>📭</div>
            <div>No traces found</div>
            <div style={{ fontSize: '0.875rem', marginTop: 8 }}>
              Try adjusting your filters or time range
            </div>
          </div>
        ) : (
          <>
            <table className={styles.tracesTable}>
              <thead>
                <tr>
                  {isGatewayMode ? (
                    <>
                      <th>Trace ID</th>
                      <th>Provider</th>
                      <th>Model</th>
                      <th>Duration</th>
                      <th>Status</th>
                      <th>Time</th>
                    </>
                  ) : (
                    <>
                      <th>Trace ID</th>
                      <th>Root Span</th>
                      <th>Service</th>
                      <th>Duration</th>
                      <th>Spans</th>
                      <th>Status</th>
                      <th>Time</th>
                    </>
                  )}
                </tr>
              </thead>
              <tbody>
                {traces.map((trace) => {
                  const details = traceDetailsCache.get(trace.trace_id);
                  return (
                    <tr key={trace.trace_id} onClick={() => handleTraceClick(trace.trace_id)}>
                      {isGatewayMode ? (
                        <>
                          <td>
                            <span className={styles.traceId}>
                              {trace.trace_id}
                            </span>
                          </td>
                          <td>
                            <Box display="flex" alignItems="center" gap={1}>
                              {getProviderIcon(details?.provider)}
                              <span className={styles.spanName}>
                                {details?.provider || '-'}
                              </span>
                            </Box>
                          </td>
                          <td>
                            <span className={styles.spanName}>
                              {details?.model || '-'}
                            </span>
                          </td>
                          <td>
                            <span className={getDurationClass(trace.duration_ms)}>
                              {formatDuration(trace.duration_ms)}
                            </span>
                          </td>
                          <td>
                            {trace.has_errors ? (
                              <span className={styles.errorBadge}>
                                <ErrorIcon sx={{ fontSize: 14 }} />
                                {trace.error_count} error{trace.error_count > 1 ? 's' : ''}
                              </span>
                            ) : (
                              <span className={styles.successBadge}>
                                <CheckCircleIcon sx={{ fontSize: 14 }} />
                                OK
                              </span>
                            )}
                          </td>
                          <td>
                            <span className={styles.timestamp}>
                              {formatTimestamp(trace.start_time)}
                            </span>
                          </td>
                        </>
                      ) : (
                        <>
                          <td>
                            <span className={styles.traceId}>
                              {trace.trace_id.substring(0, 16)}...
                            </span>
                          </td>
                          <td>
                            <span className={styles.spanName}>{trace.root_span_name}</span>
                          </td>
                          <td>
                            <span className={styles.serviceBadge}>{trace.service_name}</span>
                          </td>
                          <td>
                            <span className={getDurationClass(trace.duration_ms)}>
                              {formatDuration(trace.duration_ms)}
                            </span>
                          </td>
                          <td>
                            <span className={styles.spanCount}>{trace.span_count}</span>
                          </td>
                          <td>
                            {trace.has_errors ? (
                              <span className={styles.errorBadge}>
                                <ErrorIcon sx={{ fontSize: 14 }} />
                                {trace.error_count} error{trace.error_count > 1 ? 's' : ''}
                              </span>
                            ) : (
                              <span className={styles.successBadge}>
                                <CheckCircleIcon sx={{ fontSize: 14 }} />
                                OK
                              </span>
                            )}
                          </td>
                          <td>
                            <span className={styles.timestamp}>
                              {formatTimestamp(trace.start_time)}
                            </span>
                          </td>
                        </>
                      )}
                    </tr>
                  );
                })}
              </tbody>
            </table>
            <div className={styles.pagination}>
              <span className={styles.paginationInfo}>
                Showing {page * limit + 1} - {Math.min((page + 1) * limit, total)} of {total}
              </span>
              <div className={styles.paginationButtons}>
                <Button
                  className={styles.paginationButton}
                  disabled={page === 0}
                  onClick={() => setPage(p => p - 1)}
                >
                  Previous
                </Button>
                <Button
                  className={styles.paginationButton}
                  disabled={(page + 1) * limit >= total}
                  onClick={() => setPage(p => p + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          </>
        )}
      </div>

      {loadingDetail && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: 'rgba(0,0,0,0.5)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 1000,
        }}>
          <CircularProgress />
        </div>
      )}

      {/* Span Details Dialog for Gateway Mode */}
      {isGatewayMode && (
        <SlideDialog
          open={spanDialogOpen}
          onClose={() => {
            setSpanDialogOpen(false);
            setSelectedSpan(null);
          }}
          title={
            selectedSpan ? (
              <Box>
                <Typography sx={{ fontWeight: 600, fontSize: '1rem' }}>
                  {selectedSpan.name}
                </Typography>
                <Typography variant="body2" sx={{ color: '#888', mt: 0.5, fontSize: '0.8rem' }}>
                  {selectedSpan.service_name}
                </Typography>
              </Box>
            ) : 'Span Details'
          }
          width="500px"
        >
          {selectedSpan && (
            <SpanDetailsPanel
              span={selectedSpan}
              formatDuration={formatDuration}
              formatTimestamp={formatTimestamp}
            />
          )}
        </SlideDialog>
      )}
    </div>
  );
}

interface TraceDetailViewProps {
  trace: Trace;
  onBack: () => void;
  formatDuration: (ms: number) => string;
  formatTimestamp: (timestamp: string) => string;
  backButtonText?: string;
}

function TraceDetailView({ trace, onBack, formatDuration, formatTimestamp, backButtonText = 'Back to traces' }: TraceDetailViewProps) {
  const [selectedSpan, setSelectedSpan] = useState<Span | null>(null);

  const traceStart = new Date(trace.start_time).getTime();
  const traceDuration = trace.duration_ms || 1; // Avoid division by zero

  return (
    <div className={styles.detailContainer}>
      <div className={styles.backButton} onClick={onBack}>
        <ArrowBackIcon sx={{ fontSize: 18 }} />
        {backButtonText}
      </div>

      <div className={styles.detailHeader}>
        <div className={styles.detailTitle}>
          <TimelineIcon />
          {trace.root_span?.name || 'Trace'}
          {trace.has_errors && (
            <Chip
              size="small"
              icon={<ErrorIcon />}
              label={`${trace.error_count} error${trace.error_count > 1 ? 's' : ''}`}
              color="error"
            />
          )}
        </div>
        <div className={styles.detailMeta}>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Trace ID</span>
            <span className={styles.metaValue} style={{ fontFamily: 'monospace', fontSize: '0.85rem' }}>
              {trace.trace_id}
            </span>
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Duration</span>
            <span className={styles.metaValue}>{formatDuration(trace.duration_ms)}</span>
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Spans</span>
            <span className={styles.metaValue}>{trace.span_count}</span>
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Services</span>
            <span className={styles.metaValue}>{trace.services.join(', ')}</span>
          </div>
          <div className={styles.metaItem}>
            <span className={styles.metaLabel}>Start Time</span>
            <span className={styles.metaValue}>{formatTimestamp(trace.start_time)}</span>
          </div>
        </div>
      </div>

      {/* Professional Waterfall Timeline */}
      <TraceWaterfall
        spans={trace.spans}
        traceStartTime={traceStart}
        traceDuration={traceDuration}
        onSpanClick={(span) => setSelectedSpan(span.span_id === selectedSpan?.span_id ? null : span)}
        selectedSpanId={selectedSpan?.span_id}
      />

      {/* Sliding Dialog for Span Details */}
      <SlideDialog
        open={selectedSpan !== null}
        onClose={() => setSelectedSpan(null)}
        title={
          selectedSpan ? (
            <Box>
              <Typography sx={{ fontWeight: 600, fontSize: '1rem' }}>
                {selectedSpan.name}
              </Typography>
              <Typography variant="body2" sx={{ color: '#888', mt: 0.5, fontSize: '0.8rem' }}>
                {selectedSpan.service_name}
              </Typography>
            </Box>
          ) : 'Span Details'
        }
        width="500px"
      >
        {selectedSpan && (
          <SpanDetailsPanel
            span={selectedSpan}
            formatDuration={formatDuration}
            formatTimestamp={formatTimestamp}
          />
        )}
      </SlideDialog>
    </div>
  );
}
