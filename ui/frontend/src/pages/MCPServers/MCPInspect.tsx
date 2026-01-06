import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { api } from '../../api';
import { MCPServer, MCPInspectResponse, MCPTool, MCPPrompt, MCPResource } from '../../components/Chat/types';
import {Box, Tabs, Tab, styled, CircularProgress, Paper, Typography, IconButton, Collapse} from '@mui/material';
import {PageContainer, PageHeader, PageSubtitle, PageTitle} from "../../components/shared/Page";
import {Button} from '../../components/shared/Buttons';
import ErrorTwoTone from '@mui/icons-material/ErrorTwoTone';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import styles from './MCPInspect.module.css';

const TabContent = styled('div')(() => ({
  flex: 1,
  overflowY: 'auto',
  paddingRight: 8,
  paddingTop: 16,
}));

const ToolCardDiv = styled(Paper)(() => ({
  padding: '20px',
  boxShadow: 'none',
  background: 'var(--background-elevated)',
  borderRadius: 12,
  border: '1px solid var(--border-color)',
  overflow: 'hidden'
}))

// ToolCard component for individual tool display
const ToolCard: React.FC<{ tool: MCPTool }> = ({ tool }) => {
  const [schemaTab, setSchemaTab] = useState<'input' | 'output'>('input');
  const [expanded, setExpanded] = useState(false);

  const hasInputSchema = tool.inputSchema && Object.keys(tool.inputSchema).length > 0;
  const hasOutputSchema = tool.outputSchema && Object.keys(tool.outputSchema).length > 0;

  const handleExpandClick = (e?: React.MouseEvent) => {
    if (e) {
      e.stopPropagation();
    }
    setExpanded(!expanded);
  };

  return (
    <ToolCardDiv>
      <Box 
        display="flex" 
        justifyContent="space-between" 
        alignItems="center" 
        onClick={handleExpandClick} 
        sx={{ cursor: 'pointer' }}
      >
        <Box>
          <Typography variant="subtitle1">{tool.name}</Typography>
          {tool.description && (
            <Typography variant="body2" color="textSecondary">{tool.description}</Typography>
          )}
        </Box>
        <IconButton onClick={handleExpandClick} size="small">
          {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
        </IconButton>
      </Box>

      <Collapse in={expanded}>
        <Box sx={{ mt: 2 }}>
          <Tabs value={schemaTab} onChange={(_, v) => setSchemaTab(v)}>
            {hasInputSchema && <Tab label="Input Schema" value="input" />}
            {hasOutputSchema && <Tab label="Output Schema" value="output" />}
          </Tabs>
          
          <Box style={{ maxWidth: '100%' }}>
              {schemaTab === 'input' && hasInputSchema && (
                <SyntaxHighlighter
                  language="json"
                  style={vscDarkPlus}
                  showLineNumbers={true}
                  wrapLines={true}
                >
                  {JSON.stringify(tool.inputSchema, null, 2)}
                </SyntaxHighlighter>
              )}
              {schemaTab === 'output' && hasOutputSchema && (
                <SyntaxHighlighter
                  language="json"
                  style={vscDarkPlus}
                  showLineNumbers={true}
                  wrapLines={true}
                >
                  {JSON.stringify(tool.outputSchema, null, 2)}
                </SyntaxHighlighter>
              )}
          </Box>
        </Box>
      </Collapse>
    </ToolCardDiv>
  );
};

export const MCPInspect: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [server, setServer] = useState<MCPServer | null>(null);
  const [inspectData, setInspectData] = useState<MCPInspectResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'tools' | 'prompts' | 'resources'>('tools');

  useEffect(() => {
    if (id) {
      loadServerAndInspect();
    }
  }, [id]);

  const loadServerAndInspect = async () => {
    try {
      setLoading(true);
      setError(null);

      // Load server details
      const serverResponse = await api.get(`/mcp-servers/${id}`);
      setServer(serverResponse.data.data);

      // Load inspect data
      const inspectResponse = await api.get(`/mcp-servers/${id}/inspect`);
      setInspectData(inspectResponse.data.data);
    } catch (err: any) {
      const errorMessage = err.response?.data?.message || 
                          err.response?.data?.errorDetails?.message || 
                          'Failed to load MCP server details';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleBack = () => {
    navigate('/mcp-servers');
  };

  const renderTools = () => {
    if (!inspectData?.tools || inspectData.tools.length === 0) {
      return (
        <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" height="100%">
          <h3>No Tools Available</h3>
          <p>This MCP server doesn't expose any tools.</p>
        </Box>
      );
    }

    return (
      <Box display="flex" flexDirection="column" gap="16px">
        {inspectData.tools.map((tool, index) => (
          <ToolCard key={index} tool={tool} />
        ))}
      </Box>
    );
  };

  if (loading) {
    return (
      <PageContainer>
        <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" height="100%">
          <CircularProgress />
          <p>Loading MCP server details...</p>
        </Box>
      </PageContainer>
    );
  }

  if (error) {
    return (
      <PageContainer>
        <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" height="100%">
          <ErrorTwoTone color="error" fontSize="large" />
          <h2>Error Loading MCP Server</h2>
          <p>{error}</p>
          <Button variant="contained" color="primary" onClick={loadServerAndInspect}>
            Try Again
          </Button>
          <div className={styles.backButton} onClick={handleBack}>
            <ArrowBackIcon sx={{ fontSize: 18 }} />
            Back to MCP Servers
          </div>
        </Box>
      </PageContainer>
    );
  }

  return (
    <PageContainer>
      <div className={styles.backButton} onClick={handleBack}>
        <ArrowBackIcon sx={{ fontSize: 18 }} />
        Back to MCP Servers
      </div>
      <PageHeader>
        <Box display="flex" alignItems="center" gap="20px">
          <div>
            <PageTitle>{server?.name}</PageTitle>
            <PageSubtitle>
              {server?.endpoint}
            </PageSubtitle>
          </div>
        </Box>
      </PageHeader>

      <Box display="flex" flexDirection="column" flex={1} overflow="hidden">
        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
          <Tab label={`Tools (${inspectData?.tools?.length || 0})`} value="tools" />
          <Tab label={`Prompts (${inspectData?.prompts?.length || 0})`} value="prompts" />
          <Tab label={`Resources (${inspectData?.resources?.length || 0})`} value="resources" />
        </Tabs>

        <TabContent>
          {activeTab === 'tools' && renderTools()}
        </TabContent>
      </Box>
    </PageContainer>
  );
};
