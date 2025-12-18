import React from 'react';
import {MessageRenderer} from "./Message";
import styles from './Message.module.css';
import {Box, CircularProgress, Popover, Tooltip, Typography} from '@mui/material';
import IconButton from '@mui/material/IconButton';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import {SlideDialog} from '../../components/shared/Dialog';
import {TraceWaterfall} from '../Traces/TraceWaterfall';
import {Span, Trace} from '../Traces/types';
import {SpanDetailsPanel} from '../../components/Traces/SpanDetailsPanel';
import TimelineIcon from '@mui/icons-material/Timeline';
import {ContentType, ConversationMessage, MessageType, Role, Usage} from "../../lib/converse/types/types";
import {getTrace} from "../Traces/api";
import ThumbDownOffAltIcon from '@mui/icons-material/ThumbDownAlt';
import ThumbUpOffAltIcon from '@mui/icons-material/ThumbUpOffAlt';
import DataUsageIcon from '@mui/icons-material/DataUsage';

interface Props {
  message: ConversationMessage
  completed: boolean;
}

export const Turn: React.FC<Props> = props => {
  const { message, completed } = props;

  const [usageOpen, setUsageOpen] = React.useState(false);
  const ref = React.useRef<HTMLButtonElement | null>(null);

  const [traceDialogOpen, setTraceDialogOpen] = React.useState(false);
  const [trace, setTrace] = React.useState<Trace | null>(null);
  const [traceLoading, setTraceLoading] = React.useState(false);
  const [selectedSpan, setSelectedSpan] = React.useState<Span | null>(null);

  const onCopy = () => {
    // Copy to clipboard
    const payload = message.messages.filter(x => x.type === MessageType.Message && x.role === Role.Assistant).map(x => {
      if (x.type === MessageType.Message) {
        if (typeof x.content === "string") {
          return x.content;
        } else if (Array.isArray(x.content)) {
          return x.content.map(c => {
            if (c.type === ContentType.InputText || c.type === ContentType.OutputText) {
              return c.text;
            }

            return ""
          }).filter(x => !!x).join("\n");
        }
      }
    }).join('\n');
    navigator.clipboard.writeText(payload);
  }

  const formatDuration = (ms: number) => {
    if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`;
    if (ms < 1000) return `${ms.toFixed(1)}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const handleOpenTrace = async () => {
    if (!message.message_id) return;
    setTraceDialogOpen(true);
    setTraceLoading(true);
    try {
      const traceData = await getTrace(message.message_id.replaceAll("-", ""));
      setTrace(traceData);
    } catch (error) {
      console.error('Failed to fetch trace:', error);
    } finally {
      setTraceLoading(false);
    }
  };

  const handleCloseTrace = () => {
    setTraceDialogOpen(false);
    setTrace(null);
    setSelectedSpan(null);
  };

  const usage = message.meta?.usage as Usage;
  let hasAssistantMessage = false;
  if (message.messages.length > 0) {
    message.messages.forEach((msg) => {
      if (msg.type === MessageType.Message && msg.role === Role.Assistant) {
        hasAssistantMessage = true;
      }
    })
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: '4px' }}>
      {message.messages.map(m => <MessageRenderer key={m.id+(m.type||"")} message={m}/>)}

      {completed && hasAssistantMessage && <div className={styles.messageActions}>
        <Tooltip title="Copy to clipboard">
          <IconButton onClick={onCopy}>
            <ContentCopyIcon fontSize="small"/>
          </IconButton>
        </Tooltip>

        {<Tooltip title="Down vote"><IconButton><ThumbDownOffAltIcon fontSize="small"/></IconButton></Tooltip>}

        {<Tooltip title="Up vote"><IconButton><ThumbUpOffAltIcon fontSize="small"/></IconButton></Tooltip>}

        {/*{role === "user" &&*/}
        {/*    <Tooltip title="Edit"><IconButton><EditIcon fontSize="small"/></IconButton></Tooltip>}*/}

        {usage &&
            <Tooltip title="Usage"><IconButton onClick={() => setUsageOpen(true)} ref={ref}><DataUsageIcon fontSize="small"/></IconButton></Tooltip>}

        {message.message_id &&
            <Tooltip title="View Trace">
              <IconButton onClick={handleOpenTrace}>
                <TimelineIcon fontSize="small"/>
              </IconButton>
            </Tooltip>}

        <Popover open={usageOpen} onClose={() => setUsageOpen(false)} anchorEl={ref.current}>
          <Box display="flex" gap="8px" style={{ padding: 8 }}>
            <Box display="flex" flexDirection="column" alignItems="flex-end" gap="8px">
              <Typography fontSize="12px">Input Tokens: </Typography>
              <Typography fontSize="12px">Output Tokens: </Typography>
              <Typography fontSize="12px">Total Tokens: </Typography>
            </Box>
            <Box display="flex" flexDirection="column" gap="8px">
              <Typography fontSize="12px" fontWeight="bold">{usage?.input_tokens} ({usage?.input_tokens_details.cached_tokens || 0} cached)</Typography>
              <Typography fontSize="12px" fontWeight="bold">{usage?.output_tokens}</Typography>
              <Typography fontSize="12px" fontWeight="bold">{usage?.total_tokens}</Typography>
            </Box>
          </Box>
        </Popover>

        {/* Trace Dialog */}
        <SlideDialog
          open={traceDialogOpen}
          onClose={handleCloseTrace}
          title={
            <Box>
              <Typography sx={{ fontWeight: 600, fontSize: '1rem', display: 'flex', alignItems: 'center', gap: 1 }}>
                <TimelineIcon sx={{ fontSize: 20 }} />
                Trace Details
              </Typography>
              {trace && (
                <Typography variant="body2" sx={{ color: '#888', mt: 0.5, fontSize: '0.75rem', fontFamily: 'monospace' }}>
                  {trace.trace_id}
                </Typography>
              )}
            </Box>
          }
          width="100%"
        >
          {traceLoading ? (
            <Box display="flex" justifyContent="center" alignItems="center" py={4}>
              <CircularProgress />
            </Box>
          ) : trace ? (
            <Box>
              {/* Trace Summary */}
              <Box sx={{ display: 'flex', gap: 3, mb: 2, flexWrap: 'wrap' }}>
                <Box>
                  <Typography sx={{ color: '#666', fontSize: '0.7rem', textTransform: 'uppercase' }}>Duration</Typography>
                  <Typography sx={{ color: '#10a37f', fontWeight: 600 }}>{formatDuration(trace.duration_ms)}</Typography>
                </Box>
                <Box>
                  <Typography sx={{ color: '#666', fontSize: '0.7rem', textTransform: 'uppercase' }}>Spans</Typography>
                  <Typography sx={{ fontWeight: 600 }}>{trace.span_count}</Typography>
                </Box>
                <Box>
                  <Typography sx={{ color: '#666', fontSize: '0.7rem', textTransform: 'uppercase' }}>Services</Typography>
                  <Typography sx={{ fontWeight: 600 }}>{trace.services.join(', ')}</Typography>
                </Box>
                {trace.has_errors && (
                  <Box>
                    <Typography sx={{ color: '#666', fontSize: '0.7rem', textTransform: 'uppercase' }}>Errors</Typography>
                    <Typography sx={{ color: '#ef4444', fontWeight: 600 }}>{trace.error_count}</Typography>
                  </Box>
                )}
              </Box>

              {/* Waterfall Timeline */}
              <TraceWaterfall
                spans={trace.spans}
                traceStartTime={new Date(trace.start_time).getTime()}
                traceDuration={trace.duration_ms || 1}
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
            </Box>
          ) : (
            <Box display="flex" justifyContent="center" alignItems="center" py={4}>
              <Typography color="error">Failed to load trace</Typography>
            </Box>
          )}
        </SlideDialog>
      </div>}
    </div>
  );
}