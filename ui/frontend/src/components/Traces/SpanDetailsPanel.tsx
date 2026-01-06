import React from 'react';
import { Box, Divider, IconButton, Typography } from '@mui/material';
import ErrorIcon from '@mui/icons-material/Error';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { a11yDark } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { Span } from '../../pages/Traces/types';

interface SpanDetailsPanelProps {
  span: Span;
  formatDuration: (ms: number) => string;
  formatTimestamp: (timestamp: string) => string;
}

export const SpanDetailsPanel: React.FC<SpanDetailsPanelProps> = ({
  span,
  formatDuration,
  formatTimestamp,
}) => {
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  // Priority attributes to show first in specific order
  const priorityAttributes = [
    'llm.provider',
    'llm.model',
    'gen_ai.system_instructions',
    'gen_ai.input.messages',
    'gen_ai.output.messages',
  ];

  // Extract priority attributes and remaining attributes
  const priorityAttrs: Array<[string, string]> = [];
  const remainingAttrs: Array<[string, string]> = [];
  
  if (span.attributes) {
    // First, collect priority attributes in order
    priorityAttributes.forEach(key => {
      if (span.attributes![key]) {
        priorityAttrs.push([key, span.attributes![key]]);
      }
    });
    
    // Then collect remaining attributes
    Object.entries(span.attributes).forEach(([key, value]) => {
      if (!priorityAttributes.includes(key) && key !== 'llm.input_tokens' && key !== 'llm.output_tokens' && key !== 'llm.total_tokens') {
        remainingAttrs.push([key, value]);
      }
    });
  }

  // Extract usage information
  const inputTokens = span.attributes?.['llm.input_tokens'] || span.attributes?.['llm_input_tokens'];
  const outputTokens = span.attributes?.['llm.output_tokens'] || span.attributes?.['llm_output_tokens'];
  const totalTokens = span.attributes?.['llm.total_tokens'] || span.attributes?.['llm_total_tokens'];
  const hasUsage = inputTokens || outputTokens || totalTokens;

  return (
    <>
      {/* Status Banner */}
      {span.status_code === 'STATUS_CODE_ERROR' && (
        <Box sx={{ 
          background: 'rgba(239, 68, 68, 0.1)', 
          border: '1px solid rgba(239, 68, 68, 0.3)',
          borderRadius: 1,
          p: 1.5,
          mb: 2,
          display: 'flex',
          alignItems: 'center',
          gap: 1,
        }}>
          <ErrorIcon sx={{ color: '#ef4444', fontSize: 20 }} />
          <Box>
            <Typography sx={{ color: '#ef4444', fontWeight: 500, fontSize: '0.875rem' }}>
              Error
            </Typography>
            {span.status_message && (
              <Typography sx={{ color: '#ff8888', fontSize: '0.8rem', mt: 0.5 }}>
                {span.status_message}
              </Typography>
            )}
          </Box>
        </Box>
      )}

      {/* Basic Info */}
      <Typography sx={{ color: '#888', fontSize: '0.75rem', textTransform: 'uppercase', mb: 1.5, fontWeight: 600 }}>
        Span Information
      </Typography>
      
      <Box sx={{ display: 'grid', gap: 2, mb: 3 }}>
        <DetailRow label="Span ID" value={span.span_id} mono onCopy={() => copyToClipboard(span.span_id)} />
        <DetailRow label="Parent Span ID" value={span.parent_span_id || '— (root span)'} mono={!!span.parent_span_id} onCopy={span.parent_span_id ? () => copyToClipboard(span.parent_span_id!) : undefined} />
        <DetailRow label="Kind" value={span.kind || 'INTERNAL'} />
        <DetailRow label="Status" value={span.status_code} />
        <DetailRow label="Duration" value={formatDuration(span.duration_ms)} highlight />
        <DetailRow label="Start Time" value={formatTimestamp(span.start_time)} />
        <DetailRow label="End Time" value={formatTimestamp(span.end_time)} />
      </Box>

      <Divider sx={{ borderColor: 'var(--border-color)', my: 2 }} />

      {/* Priority Attributes */}
      {(priorityAttrs.length > 0 || hasUsage) && (
        <>
          <Typography sx={{ color: '#888', fontSize: '0.75rem', textTransform: 'uppercase', mb: 1.5, fontWeight: 600 }}>
            LLM Details
          </Typography>
          <Box sx={{ 
            background: 'var(--background-default)', 
            borderRadius: 1, 
            border: '1px solid var(--border-color)',
            overflow: 'hidden',
            mb: 2,
          }}>
            {priorityAttrs.map(([key, value], idx) => (
              <AttributeRow 
                key={key}
                attrKey={key}
                value={value}
                isLast={idx === priorityAttrs.length - 1 && !hasUsage}
                keyColor="#10a37f"
              />
            ))}
            {hasUsage && (
              <>
                {priorityAttrs.length > 0 && (
                  <Box sx={{ borderTop: '1px solid var(--border-color)' }} />
                )}
                <Box sx={{ p: 1.5 }}>
                  <Typography sx={{ 
                    color: '#10a37f', 
                    fontSize: '0.75rem', 
                    fontFamily: '"JetBrains Mono", monospace',
                    mb: 1,
                    fontWeight: 600,
                  }}>
                    Usage
                  </Typography>
                  <Box sx={{ display: 'grid', gap: 1, pl: 1 }}>
                    {inputTokens && (
                      <Typography sx={{ 
                        color: 'var(--text-primary)', 
                        fontSize: '0.8rem',
                        fontFamily: '"JetBrains Mono", monospace',
                      }}>
                        Input tokens: {inputTokens}
                      </Typography>
                    )}
                    {outputTokens && (
                      <Typography sx={{ 
                        color: 'var(--text-primary)', 
                        fontSize: '0.8rem',
                        fontFamily: '"JetBrains Mono", monospace',
                      }}>
                        Output tokens: {outputTokens}
                      </Typography>
                    )}
                    {totalTokens && (
                      <Typography sx={{ 
                        color: 'var(--text-primary)', 
                        fontSize: '0.8rem',
                        fontFamily: '"JetBrains Mono", monospace',
                        fontWeight: 600,
                      }}>
                        Total tokens: {totalTokens}
                      </Typography>
                    )}
                  </Box>
                </Box>
              </>
            )}
          </Box>
        </>
      )}

      {/* Remaining Attributes */}
      {remainingAttrs.length > 0 && (
        <>
          <Typography sx={{ color: '#888', fontSize: '0.75rem', textTransform: 'uppercase', mb: 1.5, fontWeight: 600 }}>
            Other Attributes ({remainingAttrs.length})
          </Typography>
          <Box sx={{ 
            background: 'var(--background-default)', 
            borderRadius: 1, 
            border: '1px solid var(--border-color)',
            overflow: 'hidden',
            mb: 2,
          }}>
            {remainingAttrs.map(([key, value], idx) => (
              <AttributeRow 
                key={key}
                attrKey={key}
                value={value}
                isLast={idx === remainingAttrs.length - 1}
                keyColor="#10a37f"
              />
            ))}
          </Box>
        </>
      )}

      {/* Resource Attributes */}
      {span.resource_attributes && Object.keys(span.resource_attributes).length > 0 && (
        <>
          <Typography sx={{ color: '#888', fontSize: '0.75rem', textTransform: 'uppercase', mb: 1.5, fontWeight: 600 }}>
            Resource Attributes ({Object.keys(span.resource_attributes).length})
          </Typography>
          <Box sx={{ 
            background: 'var(--background-default)', 
            borderRadius: 1, 
            border: '1px solid var(--border-color)',
            overflow: 'hidden',
          }}>
            {Object.entries(span.resource_attributes).map(([key, value], idx) => (
              <AttributeRow 
                key={key}
                attrKey={key}
                value={value}
                isLast={idx === Object.keys(span.resource_attributes!).length - 1}
                keyColor="#8b5cf6"
              />
            ))}
          </Box>
        </>
      )}
    </>
  );
};

// Helper component for detail rows
function DetailRow({ 
  label, 
  value, 
  mono = false, 
  highlight = false,
  onCopy,
}: { 
  label: string; 
  value: string; 
  mono?: boolean; 
  highlight?: boolean;
  onCopy?: () => void;
}) {
  return (
    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
      <Typography sx={{ color: '#666', fontSize: '0.8rem', flexShrink: 0 }}>
        {label}
      </Typography>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
        <Typography sx={{ 
          color: highlight ? '#10a37f' : '#ddd', 
          fontSize: '0.8rem',
          fontFamily: mono ? '"JetBrains Mono", monospace' : 'inherit',
          textAlign: 'right',
          wordBreak: 'break-all',
          fontWeight: highlight ? 600 : 400,
        }}>
          {value}
        </Typography>
        {onCopy && (
          <IconButton size="small" onClick={onCopy} sx={{ p: 0.25, color: '#555', '&:hover': { color: '#888' } }}>
            <ContentCopyIcon sx={{ fontSize: 14 }} />
          </IconButton>
        )}
      </Box>
    </Box>
  );
}

// Helper component for attribute rows with JSON highlighting
function AttributeRow({
  attrKey,
  value,
  isLast,
  keyColor,
}: {
  attrKey: string;
  value: string;
  isLast: boolean;
  keyColor: string;
}) {
  const isJsonAttribute = attrKey === 'gen_ai.output.messages' || 
                          attrKey === 'gen_ai.input.messages' || 
                          attrKey === 'gen_ai.system_instructions';
  
  const formatJsonValue = (val: string) => {
    try {
      return JSON.stringify(JSON.parse(val), null, 2);
    } catch {
      return val;
    }
  };

  return (
    <Box 
      sx={{ 
        p: 1.5,
        borderBottom: !isLast ? '1px solid var(--border-color)' : 'none',
        '&:hover': { background: 'var(--background-hover)' },
      }}
    >
      <Typography sx={{ 
        color: keyColor, 
        fontSize: '0.75rem', 
        fontFamily: '"JetBrains Mono", monospace',
        mb: 0.5,
      }}>
        {attrKey}
      </Typography>
      {isJsonAttribute ? (
        <Box sx={{ maxHeight: '250px', overflow: 'auto' }}>
          <SyntaxHighlighter 
            language="json" 
            style={a11yDark} 
            customStyle={{ 
              fontSize: '11px', 
              margin: 0, 
              padding: '8px', 
              borderRadius: '4px',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
              overflowWrap: 'break-word',
            }}
            wrapLongLines={true}
          >
            {formatJsonValue(value)}
          </SyntaxHighlighter>
        </Box>
      ) : (
        <Typography sx={{ 
          color: 'var(--text-primary)', 
          fontSize: '0.8rem',
          fontFamily: '"JetBrains Mono", monospace',
          wordBreak: 'break-all',
          whiteSpace: 'pre-wrap',
          maxHeight: '150px',
          overflow: 'auto',
        }}>
          {value}
        </Typography>
      )}
    </Box>
  );
}

export default SpanDetailsPanel;

