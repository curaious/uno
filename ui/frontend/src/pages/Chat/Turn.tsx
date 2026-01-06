import React from 'react';
import {MessageRenderer} from "./Message";
import styles from './Message.module.css';
import {Box, Button, CircularProgress, Collapse, Popover, Tooltip, Typography} from '@mui/material';
import IconButton from '@mui/material/IconButton';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import {SlideDialog} from '../../components/shared/Dialog';
import {TraceWaterfall} from '../Traces/TraceWaterfall';
import {Span, Trace} from '../Traces/types';
import {SpanDetailsPanel} from '../../components/Traces/SpanDetailsPanel';
import TimelineIcon from '@mui/icons-material/Timeline';
import {
  ContentType,
  ConversationMessage,
  FunctionCallMessage,
  MessageType,
  MessageUnion,
  Role,
  Usage,
} from "@curaious/uno-converse";
import {getTrace} from "../Traces/api";
import ThumbDownOffAltIcon from '@mui/icons-material/ThumbDownAlt';
import ThumbUpOffAltIcon from '@mui/icons-material/ThumbUpOffAlt';
import DataUsageIcon from '@mui/icons-material/DataUsage';
import {v4 as uuidv4} from "uuid";
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import CancelOutlinedIcon from '@mui/icons-material/CancelOutlined';
import BuildIcon from '@mui/icons-material/Build';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';

// Tool Call Approval Component
interface PendingToolCallsProps {
  toolCalls: FunctionCallMessage[];
  onApprove: () => void;
  onDecline: () => void;
}

const ToolCallCard: React.FC<{ fnCall: FunctionCallMessage }> = ({ fnCall }) => {
  const [expanded, setExpanded] = React.useState(false);

  const parseArgs = () => {
    try {
      return JSON.parse(fnCall.arguments);
    } catch {
      return fnCall.arguments;
    }
  };

  const args = parseArgs();
  const hasArgs = args && (typeof args === 'object' ? Object.keys(args).length > 0 : args.length > 0);

  return (
    <Box
      sx={{
        background: 'oklch(24% .006 285.885)',
        borderRadius: '6px',
        border: '1px solid #333',
        overflow: 'hidden',
        transition: 'border-color 0.15s ease',
        '&:hover': {
          borderColor: '#444',
        },
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          px: 1.5,
          py: 1,
          cursor: hasArgs ? 'pointer' : 'default',
        }}
        onClick={() => hasArgs && setExpanded(!expanded)}
      >
        <BuildIcon sx={{ fontSize: 14, color: '#888', flexShrink: 0 }} />

        <Typography
          sx={{
            fontFamily: 'source-code-pro, Menlo, Monaco, Consolas, monospace',
            fontSize: '13px',
            color: '#fff',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            flex: 1,
          }}
        >
          {fnCall.name.replaceAll('_', ' ')}
        </Typography>

        {hasArgs && (
          <ExpandMoreIcon 
            sx={{ 
              fontSize: 18, 
              color: '#888',
              transition: 'transform 0.15s ease',
              transform: expanded ? 'rotate(180deg)' : 'rotate(0deg)',
            }} 
          />
        )}
      </Box>

      <Collapse in={expanded}>
        <Box
          sx={{
            px: 1.5,
            pb: 1.5,
            pt: 0,
          }}
        >
          <Box
            sx={{
              background: '#121212',
              borderRadius: '4px',
              p: 1.5,
              maxHeight: '180px',
              overflow: 'auto',
            }}
          >
            <pre
              style={{
                margin: 0,
                fontFamily: 'source-code-pro, Menlo, Monaco, Consolas, monospace',
                fontSize: '12px',
                color: '#888',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
              }}
            >
              {typeof args === 'object' ? JSON.stringify(args, null, 2) : args}
            </pre>
          </Box>
        </Box>
      </Collapse>
    </Box>
  );
};

const PendingToolCalls: React.FC<PendingToolCallsProps> = ({ toolCalls, onApprove, onDecline }) => {
  return (
    <Box
      sx={{
        mt: 1.5,
        mb: 0.5,
        p: 1.5,
        background: 'oklch(21% .006 285.885)',
        borderRadius: '6px',
        border: '1px solid #333',
      }}
    >
      {/* Header */}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          mb: 1.5,
        }}
      >
        <Box
          sx={{
            width: 6,
            height: 6,
            borderRadius: '50%',
            background: '#10a37f',
          }}
        />
        <Typography
          sx={{
            fontSize: '13px',
            fontWeight: 500,
            color: '#fff',
          }}
        >
          Pending approval
        </Typography>
        <Typography
          sx={{
            fontSize: '12px',
            color: '#888',
            ml: 'auto',
          }}
        >
          {toolCalls.length} {toolCalls.length === 1 ? 'tool' : 'tools'}
        </Typography>
      </Box>

      {/* Tool Cards */}
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: 0.75,
          mb: 1.5,
        }}
      >
        {toolCalls.map((fnCall) => (
          <ToolCallCard key={fnCall.call_id || fnCall.id} fnCall={fnCall} />
        ))}
      </Box>

      {/* Action Buttons */}
      <Box
        sx={{
          display: 'flex',
          gap: 1,
        }}
      >
        <Button
          variant="contained"
          size="small"
          onClick={onApprove}
          startIcon={<CheckCircleOutlineIcon sx={{ fontSize: 16 }} />}
          sx={{
            background: '#10a37f',
            color: '#fff',
            fontWeight: 500,
            fontSize: '13px',
            textTransform: 'none',
            borderRadius: '6px',
            px: 2,
            py: 0.75,
            boxShadow: 'none',
            '&:hover': {
              background: '#0d8a6a',
              boxShadow: 'none',
            },
          }}
        >
          Approve{toolCalls.length > 1 ? ' all' : ''}
        </Button>
        <Button
          variant="text"
          size="small"
          onClick={onDecline}
          startIcon={<CancelOutlinedIcon sx={{ fontSize: 16 }} />}
          sx={{
            color: '#888',
            fontWeight: 500,
            fontSize: '13px',
            textTransform: 'none',
            borderRadius: '6px',
            px: 2,
            py: 0.75,
            '&:hover': {
              background: 'rgba(255, 255, 255, 0.05)',
              color: '#fff',
            },
          }}
        >
          Decline{toolCalls.length > 1 ? ' all' : ''}
        </Button>
      </Box>
    </Box>
  );
};

interface Props {
  message: ConversationMessage
  completed: boolean;
  onUserMessage: (userMessages: MessageUnion[]) => void;
}

export const Turn: React.FC<Props> = props => {
  const { message, completed, onUserMessage } = props;

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
      const traceData = await getTrace(message.meta.run_state.traceid);
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

  const usage = message.meta?.run_state?.usage as Usage;
  let hasAssistantMessage = false;
  if (message.messages.length > 0) {
    message.messages.forEach((msg) => {
      if (msg.type === MessageType.Message && msg.role === Role.Assistant) {
        hasAssistantMessage = true;
      }
    })
  }

  const approveToolCalls = () => {
    const fnCalls = message.meta.run_state.pending_tool_calls as FunctionCallMessage[];
    onUserMessage([
      {
        id: `msg_` + uuidv4(),
        type: MessageType.FunctionCallApprovalResponse,
        approved_call_ids: fnCalls.map(fnCall => fnCall.call_id!),
        rejected_call_ids: []
      }
    ])
  }

  const declineToolCalls = () => {
    const fnCalls = message.meta.run_state.pending_tool_calls as FunctionCallMessage[];
    onUserMessage([
      {
        id: `msg_` + uuidv4(),
        type: MessageType.FunctionCallApprovalResponse,
        approved_call_ids: [],
        rejected_call_ids: fnCalls.map(fnCall => fnCall.call_id!),
      }
    ])
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: '4px' }}>
      {message.messages.map(m => <MessageRenderer key={m.id+(m.type||"")} message={m}/>)}

      {message.meta?.run_state?.pending_tool_calls && (message.meta.run_state.pending_tool_calls as FunctionCallMessage[]).length > 0 && (
        <PendingToolCalls
          toolCalls={message.meta.run_state.pending_tool_calls as FunctionCallMessage[]}
          onApprove={approveToolCalls}
          onDecline={declineToolCalls}
        />
      )}

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

        {message.meta.run_state.traceid &&
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