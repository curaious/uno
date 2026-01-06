import React, { useMemo, useState } from 'react';
import { Box, Tooltip, Collapse, IconButton, Chip } from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  ChevronRight as ChevronRightIcon,
  Error as ErrorIcon,
} from '@mui/icons-material';
import { Span } from './types';

interface TraceWaterfallProps {
  spans: Span[];
  traceStartTime: number;
  traceDuration: number;
  onSpanClick?: (span: Span) => void;
  selectedSpanId?: string | null;
}

interface SpanNode extends Span {
  children: SpanNode[];
  depth: number;
}

// Color palette for services
const SERVICE_COLORS: Record<string, string> = {};
const COLOR_PALETTE = [
  '#10a37f', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6',
  '#ec4899', '#06b6d4', '#84cc16', '#f97316', '#6366f1',
];

function getServiceColor(serviceName: string): string {
  if (!SERVICE_COLORS[serviceName]) {
    const index = Object.keys(SERVICE_COLORS).length % COLOR_PALETTE.length;
    SERVICE_COLORS[serviceName] = COLOR_PALETTE[index];
  }
  return SERVICE_COLORS[serviceName];
}

function formatDuration(ms: number): string {
  if (ms < 0.001) return '<1µs';
  if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`;
  if (ms < 1000) return `${ms.toFixed(2)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

export function TraceWaterfall({
  spans,
  traceStartTime,
  traceDuration,
  onSpanClick,
  selectedSpanId,
}: TraceWaterfallProps) {
  const [expandedSpans, setExpandedSpans] = useState<Set<string>>(new Set(spans.map(s => s.span_id)));

  // Build span tree
  const spanTree = useMemo(() => {
    const spanMap = new Map<string, SpanNode>();
    const roots: SpanNode[] = [];

    // Create nodes
    spans.forEach((span) => {
      spanMap.set(span.span_id, { ...span, children: [], depth: 0 });
    });

    // Build tree
    spans.forEach((span) => {
      const node = spanMap.get(span.span_id)!;
      if (span.parent_span_id && spanMap.has(span.parent_span_id)) {
        const parent = spanMap.get(span.parent_span_id)!;
        parent.children.push(node);
        node.depth = parent.depth + 1;
      } else {
        roots.push(node);
      }
    });

    // Sort children by start time
    const sortChildren = (node: SpanNode) => {
      node.children.sort((a, b) => 
        new Date(a.start_time).getTime() - new Date(b.start_time).getTime()
      );
      node.children.forEach(sortChildren);
    };
    roots.forEach(sortChildren);

    // Update depths
    const updateDepth = (node: SpanNode, depth: number) => {
      node.depth = depth;
      node.children.forEach(child => updateDepth(child, depth + 1));
    };
    roots.forEach(root => updateDepth(root, 0));

    return roots;
  }, [spans]);

  // Flatten tree for rendering
  const flattenedSpans = useMemo(() => {
    const result: SpanNode[] = [];
    const flatten = (nodes: SpanNode[]) => {
      nodes.forEach((node) => {
        result.push(node);
        if (expandedSpans.has(node.span_id)) {
          flatten(node.children);
        }
      });
    };
    flatten(spanTree);
    return result;
  }, [spanTree, expandedSpans]);

  const toggleExpand = (spanId: string) => {
    setExpandedSpans((prev) => {
      const next = new Set(prev);
      if (next.has(spanId)) {
        next.delete(spanId);
      } else {
        next.add(spanId);
      }
      return next;
    });
  };

  const hasChildren = (spanId: string): boolean => {
    return spans.some(s => s.parent_span_id === spanId);
  };

  return (
    <Box sx={{ 
      background: '#1a1a1d', 
      borderRadius: 2, 
      overflow: 'hidden',
      border: '1px solid #2a2a2d',
    }}>
      {/* Header */}
      <Box sx={{ 
        display: 'grid', 
        gridTemplateColumns: '320px 1fr 100px',
        borderBottom: '1px solid #2a2a2d',
        background: '#141416',
        px: 2,
        py: 1.5,
      }}>
        <Box sx={{ fontSize: 12, fontWeight: 600, color: '#888', textTransform: 'uppercase' }}>
          Service & Operation
        </Box>
        <Box sx={{ fontSize: 12, fontWeight: 600, color: '#888', textTransform: 'uppercase', pl: 2 }}>
          Timeline
        </Box>
        <Box sx={{ fontSize: 12, fontWeight: 600, color: '#888', textTransform: 'uppercase', textAlign: 'right' }}>
          Duration
        </Box>
      </Box>

      {/* Timeline ruler */}
      <Box sx={{ 
        display: 'grid', 
        gridTemplateColumns: '320px 1fr 100px',
        borderBottom: '1px solid #2a2a2d',
        background: '#141416',
      }}>
        <Box />
        <Box sx={{ display: 'flex', justifyContent: 'space-between', px: 2, py: 0.5 }}>
          {[0, 25, 50, 75, 100].map((pct) => (
            <Box key={pct} sx={{ fontSize: 10, color: '#555' }}>
              {formatDuration((traceDuration * pct) / 100)}
            </Box>
          ))}
        </Box>
        <Box />
      </Box>

      {/* Spans */}
      <Box sx={{ maxHeight: 600, overflowY: 'auto' }}>
        {flattenedSpans.map((span) => {
          const spanStart = new Date(span.start_time).getTime();
          const leftPct = ((spanStart - traceStartTime) / traceDuration) * 100;
          const widthPct = (span.duration_ms / traceDuration) * 100;
          const isError = span.status_code === 'STATUS_CODE_ERROR';
          const isSelected = selectedSpanId === span.span_id;
          const serviceColor = getServiceColor(span.service_name);
          const hasKids = hasChildren(span.span_id);
          const isExpanded = expandedSpans.has(span.span_id);

          return (
            <Box
              key={span.span_id}
              onClick={() => onSpanClick?.(span)}
              sx={{
                display: 'grid',
                gridTemplateColumns: '320px 1fr 100px',
                alignItems: 'center',
                borderBottom: '1px solid #222',
                cursor: 'pointer',
                background: isSelected ? '#252530' : 'transparent',
                '&:hover': {
                  background: isSelected ? '#252530' : '#1e1e22',
                },
                transition: 'background 0.15s',
              }}
            >
              {/* Service & Operation */}
              <Box sx={{ 
                display: 'flex', 
                alignItems: 'center', 
                gap: 0.5,
                py: 1,
                px: 1,
                pl: `${span.depth * 20 + 8}px`,
              }}>
                {hasKids ? (
                  <IconButton 
                    size="small" 
                    onClick={(e) => { e.stopPropagation(); toggleExpand(span.span_id); }}
                    sx={{ p: 0.25, color: '#666' }}
                  >
                    {isExpanded ? <ExpandMoreIcon fontSize="small" /> : <ChevronRightIcon fontSize="small" />}
                  </IconButton>
                ) : (
                  <Box sx={{ width: 24 }} />
                )}
                
                <Box sx={{ 
                  width: 4, 
                  height: 28, 
                  borderRadius: 1, 
                  background: serviceColor,
                  flexShrink: 0,
                }} />
                
                <Box sx={{ overflow: 'hidden', ml: 1 }}>
                  <Box sx={{ 
                    fontSize: 11, 
                    color: serviceColor, 
                    fontWeight: 500,
                    whiteSpace: 'nowrap',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                  }}>
                    {span.service_name}
                  </Box>
                  <Box sx={{ 
                    fontSize: 13, 
                    color: '#ddd',
                    fontWeight: 500,
                    whiteSpace: 'nowrap',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1,
                  }}>
                    {span.name}
                    {isError && (
                      <Chip 
                        size="small" 
                        icon={<ErrorIcon sx={{ fontSize: '14px !important' }} />}
                        label="Error" 
                        sx={{ 
                          height: 18, 
                          fontSize: 10,
                          background: 'rgba(239, 68, 68, 0.2)',
                          color: '#ef4444',
                          '& .MuiChip-icon': { color: '#ef4444' },
                        }} 
                      />
                    )}
                  </Box>
                </Box>
              </Box>

              {/* Timeline bar */}
              <Box sx={{ px: 2, py: 1 }}>
                <Tooltip 
                  title={
                    <Box>
                      <Box><strong>{span.name}</strong></Box>
                      <Box>Service: {span.service_name}</Box>
                      <Box>Duration: {formatDuration(span.duration_ms)}</Box>
                      <Box>Start: {new Date(span.start_time).toISOString()}</Box>
                    </Box>
                  }
                  arrow
                  placement="top"
                >
                  <Box sx={{ 
                    position: 'relative', 
                    height: 24, 
                    background: '#252529',
                    borderRadius: 1,
                  }}>
                    <Box
                      sx={{
                        position: 'absolute',
                        left: `${Math.max(0, leftPct)}%`,
                        width: `${Math.max(0.5, Math.min(widthPct, 100 - leftPct))}%`,
                        height: '100%',
                        background: isError 
                          ? 'linear-gradient(90deg, #ef4444, #dc2626)'
                          : `linear-gradient(90deg, ${serviceColor}, ${serviceColor}dd)`,
                        borderRadius: 1,
                        minWidth: 4,
                        boxShadow: isSelected ? `0 0 0 2px ${serviceColor}66` : 'none',
                      }}
                    />
                  </Box>
                </Tooltip>
              </Box>

              {/* Duration */}
              <Box sx={{ 
                textAlign: 'right', 
                pr: 2,
                fontFamily: '"JetBrains Mono", monospace',
                fontSize: 12,
                color: isError ? '#ef4444' : '#888',
              }}>
                {formatDuration(span.duration_ms)}
              </Box>
            </Box>
          );
        })}
      </Box>

      {/* Legend */}
      <Box sx={{ 
        display: 'flex', 
        gap: 2, 
        p: 2, 
        borderTop: '1px solid #2a2a2d',
        background: '#141416',
        flexWrap: 'wrap',
      }}>
        {Object.entries(SERVICE_COLORS).map(([service, color]) => (
          <Box key={service} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ width: 12, height: 12, borderRadius: '50%', background: color }} />
            <Box sx={{ fontSize: 12, color: '#888' }}>{service}</Box>
          </Box>
        ))}
      </Box>
    </Box>
  );
}

