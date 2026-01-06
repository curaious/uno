import React, {useEffect, useState} from 'react';
import {api} from '../../api';
import {
  CreateAgentRequest,
  AgentWithDetails,
  UpdateAgentRequest,
  Model,
  PromptWithLatestVersion,
  PromptVersion,
  MCPServer,
  AgentMCPServerReq,
  Schema
} from '../../components/Chat/types';
import {Action, Column, DataTable} from '../../components/DataTable/DataTable';
import {PageContainer, PageHeader, PageSubtitle, PageTitle} from "../../components/shared/Page";
import {Button} from "../../components/shared/Buttons";
import {Box, Chip, IconButton, MenuItem, Checkbox, ListItemText, CircularProgress, FormControlLabel, Switch, TextField} from '@mui/material';
import Edit from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import Add from '@mui/icons-material/Add';
import {Input, Select, InputGroup, InputLabel} from '../../components/shared/Input';
import {SlideDialog} from "../../components/shared/Dialog";
import {MCPTool} from '../../components/Chat/types';

export const Agents: React.FC = props => {
  const [agents, setAgents] = useState<AgentWithDetails[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [prompts, setPrompts] = useState<PromptWithLatestVersion[]>([]);
  const [promptVersions, setPromptVersions] = useState<PromptVersion[]>([]);
  const [summarizerPromptVersions, setSummarizerPromptVersions] = useState<PromptVersion[]>([]);
  const [mcpServers, setMcpServers] = useState<MCPServer[]>([]);
  const [schemas, setSchemas] = useState<Schema[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showDialog, setShowDialog] = useState(false);
  const [editingAgent, setEditingAgent] = useState<AgentWithDetails | null>(null);
  const [formData, setFormData] = useState<CreateAgentRequest>({
    name: '',
    model_id: '',
    prompt_id: '',
    prompt_label: undefined,
    schema_id: undefined,
    enable_history: false,
    summarizer_type: undefined,
    llm_summarizer_token_threshold: undefined,
    llm_summarizer_keep_recent_count: undefined,
    llm_summarizer_prompt_id: undefined,
    llm_summarizer_prompt_label: undefined,
      llm_summarizer_model_id: undefined,
      sliding_window_keep_count: undefined,
      mcp_servers: [],
  });
  const [formErrors, setFormErrors] = useState<{ [key: string]: string }>({});
  // Map to store tools for each MCP server: key is MCP server ID, value is tools array
  const [mcpServerTools, setMcpServerTools] = useState<{ [key: string]: MCPTool[] }>({});
  // Map to store loading state for each MCP server: key is MCP server ID
  const [loadingTools, setLoadingTools] = useState<{ [key: string]: boolean }>({});
  // Map to store errors when fetching tools: key is MCP server ID
  const [toolErrors, setToolErrors] = useState<{ [key: string]: string | null }>({});

  // Load data on component mount
  useEffect(() => {
    loadModels();
    loadPrompts();
    loadMcpServers();
    loadSchemas();
    loadAgents();
  }, []);

  const loadModels = async () => {
    try {
      const response = await api.get('/models');
      setModels(response.data.data || []);
    } catch (err: any) {
      console.error('Failed to load models', err);
    }
  };

  const loadPrompts = async () => {
    try {
      const response = await api.get('/prompts');
      setPrompts(response.data.data || []);
    } catch (err: any) {
      console.error('Failed to load prompts', err);
    }
  };

  const loadMcpServers = async () => {
    try {
      const response = await api.get('/mcp-servers');
      setMcpServers(response.data.data || []);
    } catch (err: any) {
      console.error('Failed to load MCP servers', err);
    }
  };

  const loadSchemas = async () => {
    try {
      const response = await api.get('/schemas');
      setSchemas(response.data.data || []);
    } catch (err: any) {
      console.error('Failed to load schemas', err);
    }
  };

  const loadAgents = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await api.get('/agents');
      setAgents(response.data.data || []);
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to load agents';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const loadPromptVersions = async (promptName: string) => {
    try {
      const response = await api.get(`/prompts/${encodeURIComponent(promptName)}/versions`);
      const versions = response.data.data || [];
      setPromptVersions(versions);
      // Default to latest label if available
      const latestVersion = versions.find((v: any) => v.label === 'latest') || versions[0];
      if (latestVersion?.label) {
        setFormData(prev => ({ ...prev, prompt_label: latestVersion.label || undefined }));
      }
    } catch (err: any) {
      console.error('Failed to load prompt versions', err);
      setPromptVersions([]);
    }
  };

  const loadSummarizerPromptVersions = async (promptName: string) => {
    try {
      const response = await api.get(`/prompts/${encodeURIComponent(promptName)}/versions`);
      const versions = response.data.data || [];
      setSummarizerPromptVersions(versions);
      // Default to latest label if available
      const latestVersion = versions.find((v: any) => v.label === 'latest') || versions[0];
      if (latestVersion?.label && !formData.llm_summarizer_prompt_label) {
        setFormData(prev => ({ ...prev, llm_summarizer_prompt_label: latestVersion.label || undefined }));
      }
    } catch (err: any) {
      console.error('Failed to load summarizer prompt versions', err);
      setSummarizerPromptVersions([]);
    }
  };

  const handleCreate = () => {
    setEditingAgent(null);
    setFormData({
      name: '',
      model_id: '',
      prompt_id: '',
      prompt_label: undefined,
      schema_id: undefined,
      enable_history: false,
      summarizer_type: undefined,
      llm_summarizer_token_threshold: undefined,
      llm_summarizer_keep_recent_count: undefined,
      llm_summarizer_prompt_id: undefined,
      llm_summarizer_prompt_label: undefined,
      sliding_window_keep_count: undefined,
      mcp_servers: [],
    });
    setPromptVersions([]);
    setFormErrors({});
    // Clear tools cache when creating new agent
    setMcpServerTools({});
    setLoadingTools({});
    setToolErrors({});
    setShowDialog(true);
  };

  const handleEdit = async (agent: AgentWithDetails) => {
    setEditingAgent(agent);
    const mcpServersData = (agent.mcp_servers || []).map(ms => ({
      mcp_server_id: ms.mcp_server_id,
      tool_filters: ms.tool_filters || [],
    }));
    
    setFormData({
      name: agent.name,
      model_id: agent.model_id,
      prompt_id: agent.prompt_id,
      prompt_label: agent.prompt_label || undefined,
      schema_id: agent.schema_id || undefined,
      enable_history: agent.enable_history ?? false,
      summarizer_type: agent.summarizer_type || undefined,
      llm_summarizer_token_threshold: agent.llm_summarizer_token_threshold || undefined,
      llm_summarizer_keep_recent_count: agent.llm_summarizer_keep_recent_count || undefined,
      llm_summarizer_prompt_id: agent.llm_summarizer_prompt_id || undefined,
      llm_summarizer_prompt_label: agent.llm_summarizer_prompt_label || undefined,
      llm_summarizer_model_id: agent.llm_summarizer_model_id || undefined,
      sliding_window_keep_count: agent.sliding_window_keep_count || undefined,
      mcp_servers: mcpServersData,
    });
    
    // Fetch tools for all MCP servers in parallel
    const fetchPromises = mcpServersData
      .filter(ms => ms.mcp_server_id)
      .map(ms => fetchMcpServerTools(ms.mcp_server_id));
    await Promise.all(fetchPromises);
    
    // Load prompt versions for the selected prompt
    const selectedPrompt = prompts.find(p => p.id === agent.prompt_id);
    if (selectedPrompt) {
      await loadPromptVersions(selectedPrompt.name);
    }
    
    // Load prompt versions for the summarizer prompt if it exists
    if (agent.llm_summarizer_prompt_id) {
      const selectedSummarizerPrompt = prompts.find(p => p.id === agent.llm_summarizer_prompt_id);
      if (selectedSummarizerPrompt) {
        await loadSummarizerPromptVersions(selectedSummarizerPrompt.name);
      }
    }
    
    setFormErrors({});
    setShowDialog(true);
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this agent?')) {
      return;
    }

    try {
      await api.delete(`/agents/${id}`);
      await loadAgents();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to delete agent';
      setError(errorMessage);
    }
  };

  const handleAddMcpServer = () => {
    setFormData({
      ...formData,
      mcp_servers: [
        ...(formData.mcp_servers || []),
        { mcp_server_id: '', tool_filters: [] }
      ],
    });
  };

  const handleRemoveMcpServer = (index: number) => {
    const mcpServerToRemove = formData.mcp_servers?.[index];
    const updatedServers = formData.mcp_servers?.filter((_, i) => i !== index) || [];
    
    // Clear tools cache for removed server if it's not used by any other server in the form
    if (mcpServerToRemove?.mcp_server_id) {
      const isUsedElsewhere = updatedServers.some(ms => ms.mcp_server_id === mcpServerToRemove.mcp_server_id);
      if (!isUsedElsewhere) {
        setMcpServerTools(prev => {
          const newTools = { ...prev };
          delete newTools[mcpServerToRemove.mcp_server_id];
          return newTools;
        });
        setLoadingTools(prev => {
          const newLoading = { ...prev };
          delete newLoading[mcpServerToRemove.mcp_server_id];
          return newLoading;
        });
        setToolErrors(prev => {
          const newErrors = { ...prev };
          delete newErrors[mcpServerToRemove.mcp_server_id];
          return newErrors;
        });
      }
    }
    
    setFormData({
      ...formData,
      mcp_servers: updatedServers,
    });
  };

  const fetchMcpServerTools = async (mcpServerId: string) => {
    if (!mcpServerId) {
      return;
    }

    // If tools are already loaded, don't fetch again
    if (mcpServerTools[mcpServerId]) {
      return;
    }

    try {
      setLoadingTools(prev => ({ ...prev, [mcpServerId]: true }));
      setToolErrors(prev => ({ ...prev, [mcpServerId]: null }));
      
      const response = await api.get(`/mcp-servers/${mcpServerId}/inspect`);
      const tools = response.data.data?.tools || [];
      
      setMcpServerTools(prev => ({ ...prev, [mcpServerId]: tools }));
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
                          err.response?.data?.errorDetails?.message ||
                          'Failed to load tools';
      setToolErrors(prev => ({ ...prev, [mcpServerId]: errorMessage }));
    } finally {
      setLoadingTools(prev => ({ ...prev, [mcpServerId]: false }));
    }
  };

  const handleMcpServerChange = async (index: number, field: 'mcp_server_id' | 'tool_filters', value: string | string[]) => {
    const updated = [...(formData.mcp_servers || [])];
    updated[index] = {
      ...updated[index],
      [field]: value,
    };
    
    // If MCP server is being changed, fetch tools and reset tool_filters
    if (field === 'mcp_server_id' && typeof value === 'string') {
      updated[index].tool_filters = [];
      if (value) {
        await fetchMcpServerTools(value);
      }
    }
    
    setFormData({
      ...formData,
      mcp_servers: updated,
    });
  };

  const handleToolFiltersChange = (index: number, selectedTools: string[]) => {
    handleMcpServerChange(index, 'tool_filters', selectedTools);
  };

  const validateForm = (): boolean => {
    const errors: { [key: string]: string } = {};

    if (!formData.name.trim()) {
      errors.name = 'Name is required';
    }

    if (!formData.model_id) {
      errors.model_id = 'Model is required';
    }

    if (!formData.prompt_id) {
      errors.prompt_id = 'Prompt is required';
    }

    // Validate conversation history configuration
    if (formData.enable_history) {
      if (!formData.summarizer_type) {
        errors.summarizer_type = 'Summarizer type is required when history is enabled';
      } else if (formData.summarizer_type === 'llm') {
        if (!formData.llm_summarizer_token_threshold || formData.llm_summarizer_token_threshold <= 0) {
          errors.llm_summarizer_token_threshold = 'Token threshold is required and must be greater than 0';
        }
        if (formData.llm_summarizer_keep_recent_count === undefined || formData.llm_summarizer_keep_recent_count < 0) {
          errors.llm_summarizer_keep_recent_count = 'Keep recent count is required and must be non-negative';
        }
        if (!formData.llm_summarizer_prompt_id) {
          errors.llm_summarizer_prompt_id = 'Summarization prompt is required';
        }
        if (!formData.llm_summarizer_model_id) {
          errors.llm_summarizer_model_id = 'Summarization model is required';
        }
      } else if (formData.summarizer_type === 'sliding_window') {
        if (!formData.sliding_window_keep_count || formData.sliding_window_keep_count <= 0) {
          errors.sliding_window_keep_count = 'Keep count is required and must be greater than 0';
        }
      }
    }

    // Validate MCP servers
    if (formData.mcp_servers) {
      formData.mcp_servers.forEach((mcpServer, index) => {
        if (!mcpServer.mcp_server_id) {
          errors[`mcp_server_${index}`] = 'MCP Server is required';
        }
      });
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    try {
      if (editingAgent) {
        // Update existing agent
        const updateData: UpdateAgentRequest = {
          name: formData.name,
          model_id: formData.model_id,
          prompt_id: formData.prompt_id,
          prompt_label: formData.prompt_label,
          schema_id: formData.schema_id || undefined,
          clear_schema_id: !formData.schema_id && !!editingAgent.schema_id,
          enable_history: formData.enable_history,
          summarizer_type: formData.summarizer_type,
          llm_summarizer_token_threshold: formData.llm_summarizer_token_threshold,
          llm_summarizer_keep_recent_count: formData.llm_summarizer_keep_recent_count,
          llm_summarizer_prompt_id: formData.llm_summarizer_prompt_id,
          llm_summarizer_prompt_label: formData.llm_summarizer_prompt_label,
          llm_summarizer_model_id: formData.llm_summarizer_model_id,
          sliding_window_keep_count: formData.sliding_window_keep_count,
          mcp_servers: formData.mcp_servers,
        };
        await api.put(`/agents/${editingAgent.id}`, updateData);
      } else {
        // Create new agent
        await api.post('/agents', formData);
      }

      setShowDialog(false);
      await loadAgents();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to save agent';
      setError(errorMessage);
    }
  };

  const handleCloseDialog = () => {
    setShowDialog(false);
    setEditingAgent(null);
    setFormData({
      name: '',
      model_id: '',
      prompt_id: '',
      prompt_label: undefined,
      schema_id: undefined,
      enable_history: false,
      summarizer_type: undefined,
      llm_summarizer_token_threshold: undefined,
      llm_summarizer_keep_recent_count: undefined,
      llm_summarizer_prompt_id: undefined,
      llm_summarizer_prompt_label: undefined,
      sliding_window_keep_count: undefined,
      mcp_servers: [],
    });
    setPromptVersions([]);
    setSummarizerPromptVersions([]);
    setFormErrors({});
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  // Table configuration
  const columns: Column<AgentWithDetails>[] = [
    {
      key: 'name',
      label: 'Name',
      render: (value, item) => (
        <div>
          <span>{item.name}</span>
        </div>
      )
    },
    {
      key: 'model_name',
      label: 'Model',
      render: (value, item) => (
        <div>
          <span>{item.model_name}</span>
        </div>
      )
    },
    {
      key: 'prompt_name',
      label: 'Prompt',
      render: (value, item) => (
        <div>
          <span>{item.prompt_name}</span>
          {item.prompt_label && (
            <Chip 
              label={item.prompt_label} 
              size="small" 
              sx={{ ml: 1 }} 
              color={item.prompt_label === 'production' ? 'primary' : 'default'}
            />
          )}
        </div>
      )
    },
    {
      key: 'schema_name',
      label: 'Schema',
      render: (value, item) => (
        <div>
          {item.schema_name ? (
            <Chip label={item.schema_name} size="small" variant="outlined" />
          ) : (
            <span style={{ color: '#999' }}>None</span>
          )}
        </div>
      )
    },
    {
      key: 'mcp_servers',
      label: 'MCP Servers',
      render: (value, item) => (
        <div>
          {item.mcp_servers && item.mcp_servers.length > 0 ? (
            <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
              {item.mcp_servers.map((ms, idx) => (
                <Chip
                  key={idx}
                  label={ms.mcp_server_name + (ms.tool_filters && ms.tool_filters.length > 0 ? ` (${ms.tool_filters.length} tools)` : '')}
                  size="small"
                  variant="outlined"
                />
              ))}
            </Box>
          ) : (
            <span style={{ color: '#999' }}>None</span>
          )}
        </div>
      )
    },
    {
      key: 'created_at',
      label: 'Created',
      render: (value, item) => (
        <div>
          <span>{formatDate(item.created_at)}</span>
        </div>
      )
    },
  ];

  const actions: Action<AgentWithDetails>[] = [
    {
      icon: <Edit />,
      label: 'Edit',
      onClick: handleEdit,
    },
    {
      icon: <DeleteIcon />,
      label: 'Delete',
      onClick: (item) => handleDelete(item.id),
    },
  ];

  return (
    <PageContainer>
      <PageHeader>
        <div>
          <PageTitle>Agents</PageTitle>
          <PageSubtitle>Manage AI agents with models, prompts, and MCP servers</PageSubtitle>
        </div>

        <Button variant="contained" color="primary" onClick={handleCreate}>
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path
              d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
          </svg>
          Create Agent
        </Button>
      </PageHeader>

      {error && (
        <Box sx={{ mb: 2, p: 2, bgcolor: 'error.light', color: 'error.contrastText', borderRadius: 1 }}>
          {error}
        </Box>
      )}

      <DataTable
        columns={columns}
        data={agents}
        actions={actions}
        loading={loading}
      />

      {/* Create/Edit Dialog */}
      <SlideDialog
        open={showDialog}
        onClose={handleCloseDialog}
        title={editingAgent ? 'Edit Agent' : 'Create Agent'}
        maxWidth="700px"
        actions={
          <>
            <Button type="button" onClick={handleCloseDialog}>
              Cancel
            </Button>
            <Button type="submit" variant="contained" form="agent-form">
              {editingAgent ? 'Update' : 'Create'}
            </Button>
          </>
        }
      >
        <form id="agent-form" onSubmit={handleSubmit}>
            <InputGroup>
              <InputLabel>Name</InputLabel>
              <Input
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                error={!!formErrors.name}
                required
                fullWidth
              />
            </InputGroup>

            <InputGroup>
              <InputLabel>Model *</InputLabel>
              <Select
                id="model_id"
                value={formData.model_id}
                onChange={(e) => setFormData({ ...formData, model_id: e.target.value })}
                error={!!formErrors.model_id}
                fullWidth
                MenuProps={{
                  style: { zIndex: 1500 },
                  PaperProps: {
                    style: { zIndex: 1500 }
                  }
                }}
              >
                <MenuItem value="" disabled>Select a model</MenuItem>
                {models.map((model) => (
                  <MenuItem key={model.id} value={model.id}>
                    {model.name} ({model.model_id})
                  </MenuItem>
                ))}
              </Select>
              {formErrors.model_id && <span style={{ color: 'red', fontSize: '0.875rem' }}>{formErrors.model_id}</span>}
            </InputGroup>

            <InputGroup>
              <InputLabel>Prompt *</InputLabel>
              <Select
                id="prompt_id"
                value={formData.prompt_id}
                onChange={async (e) => {
                  const promptId = e.target.value;
                  setFormData({ ...formData, prompt_id: promptId, prompt_label: undefined });
                  // Load prompt versions when prompt is selected
                  const selectedPrompt = prompts.find(p => p.id === promptId);
                  if (selectedPrompt) {
                    await loadPromptVersions(selectedPrompt.name);
                  } else {
                    setPromptVersions([]);
                  }
                }}
                error={!!formErrors.prompt_id}
                fullWidth
                MenuProps={{
                  style: { zIndex: 1500 },
                  PaperProps: {
                    style: { zIndex: 1500 }
                  }
                }}
              >
                <MenuItem value="" disabled>Select a prompt</MenuItem>
                {prompts.map((prompt) => (
                  <MenuItem key={prompt.id} value={prompt.id}>
                    {prompt.name}
                  </MenuItem>
                ))}
              </Select>
              {formErrors.prompt_id && <span style={{ color: 'red', fontSize: '0.875rem' }}>{formErrors.prompt_id}</span>}
            </InputGroup>

            {formData.prompt_id && (
              <InputGroup>
                <InputLabel>Prompt Label</InputLabel>
                <Select
                  id="prompt_label"
                  value={formData.prompt_label || ''}
                  onChange={(e) => setFormData({ ...formData, prompt_label: e.target.value || undefined })}
                  fullWidth
                  MenuProps={{
                    style: { zIndex: 1500 },
                    PaperProps: {
                      style: { zIndex: 1500 }
                    }
                  }}
                >
                  <MenuItem value="">Latest version (default)</MenuItem>
                  {promptVersions
                    .filter(v => v.label && (v.label === 'latest' || v.label === 'production'))
                    .map((version) => (
                      <MenuItem key={version.id} value={version.label || ''}>
                        {version.label} (v{version.version})
                      </MenuItem>
                    ))}
                </Select>
                <p style={{ fontSize: '0.75rem', color: '#666', marginTop: '4px' }}>
                  Select a specific label (latest/production) or leave as default to use the latest version
                </p>
              </InputGroup>
            )}

            <InputGroup>
              <InputLabel>Output Schema (Optional)</InputLabel>
              <Select
                id="schema_id"
                value={formData.schema_id || ''}
                onChange={(e) => setFormData({ ...formData, schema_id: e.target.value || undefined })}
                fullWidth
                MenuProps={{
                  style: { zIndex: 1500 },
                  PaperProps: {
                    style: { zIndex: 1500 }
                  }
                }}
              >
                <MenuItem value="">No structured output</MenuItem>
                {schemas.map((schema) => (
                  <MenuItem key={schema.id} value={schema.id}>
                    {schema.name}
                  </MenuItem>
                ))}
              </Select>
              <p style={{ fontSize: '0.75rem', color: '#666', marginTop: '4px' }}>
                Select a JSON schema to enforce structured output from the agent
              </p>
            </InputGroup>

            <InputGroup>
              <FormControlLabel
                control={
                  <Switch
                    checked={formData.enable_history || false}
                    onChange={(e) => {
                      const enabled = e.target.checked;
                      setFormData({
                        ...formData,
                        enable_history: enabled,
                        // Clear summarizer fields when disabling history
                        summarizer_type: enabled ? formData.summarizer_type : undefined,
                        llm_summarizer_token_threshold: enabled ? formData.llm_summarizer_token_threshold : undefined,
                        llm_summarizer_keep_recent_count: enabled ? formData.llm_summarizer_keep_recent_count : undefined,
                        llm_summarizer_prompt_id: enabled ? formData.llm_summarizer_prompt_id : undefined,
                        llm_summarizer_prompt_label: enabled ? formData.llm_summarizer_prompt_label : undefined,
                        llm_summarizer_model_id: enabled ? formData.llm_summarizer_model_id : undefined,
                        sliding_window_keep_count: enabled ? formData.sliding_window_keep_count : undefined,
                      });
                    }}
                  />
                }
                label="Enable Conversation History"
              />
            </InputGroup>

            {formData.enable_history && (
              <>
                <InputGroup>
                  <InputLabel>Summarizer Type *</InputLabel>
                  <Select
                    id="summarizer_type"
                    value={formData.summarizer_type || ''}
                    onChange={(e) => {
                      const type = e.target.value as 'llm' | 'sliding_window' | 'none' | '';
                      setFormData({
                        ...formData,
                        summarizer_type: (type === '' ? undefined : type) as 'llm' | 'sliding_window' | 'none' | undefined,
                        // Clear fields when switching summarizer types
                        llm_summarizer_token_threshold: type === 'llm' ? formData.llm_summarizer_token_threshold : undefined,
                        llm_summarizer_keep_recent_count: type === 'llm' ? formData.llm_summarizer_keep_recent_count : undefined,
                        llm_summarizer_prompt_id: type === 'llm' ? formData.llm_summarizer_prompt_id : undefined,
                        llm_summarizer_prompt_label: type === 'llm' ? formData.llm_summarizer_prompt_label : undefined,
                        llm_summarizer_model_id: type === 'llm' ? formData.llm_summarizer_model_id : undefined,
                        sliding_window_keep_count: type === 'sliding_window' ? formData.sliding_window_keep_count : undefined,
                      });
                    }}
                    error={!!formErrors.summarizer_type}
                    fullWidth
                    MenuProps={{
                      style: { zIndex: 1500 },
                      PaperProps: {
                        style: { zIndex: 1500 }
                      }
                    }}
                  >
                    <MenuItem value="" disabled>Select a summarizer type</MenuItem>
                    <MenuItem value="none">No Summarization</MenuItem>
                    <MenuItem value="llm">LLM Based Summarizer</MenuItem>
                    <MenuItem value="sliding_window">Sliding Window Summarizer</MenuItem>
                  </Select>
                  {formErrors.summarizer_type && <span style={{ color: 'red', fontSize: '0.875rem' }}>{formErrors.summarizer_type}</span>}
                </InputGroup>

                {formData.summarizer_type === 'llm' && (
                  <>
                    <Box sx={{ display: 'flex', gap: 2 }}>
                      <Box sx={{ flex: 1 }}>
                        <InputGroup>
                          <InputLabel>Token Threshold *</InputLabel>
                          <Input
                            type="number"
                            value={formData.llm_summarizer_token_threshold || ''}
                            onChange={(e) => setFormData({
                              ...formData,
                              llm_summarizer_token_threshold: e.target.value ? parseInt(e.target.value, 10) : undefined
                            })}
                            error={!!formErrors.llm_summarizer_token_threshold}
                            helperText={formErrors.llm_summarizer_token_threshold || 'Summarization will trigger when token count exceeds this threshold'}
                            fullWidth
                            inputProps={{ min: 1 }}
                          />
                        </InputGroup>
                      </Box>
                      <Box sx={{ flex: 1 }}>
                        <InputGroup>
                          <InputLabel>Min Agent Run History to Keep *</InputLabel>
                          <Input
                            type="number"
                            value={formData.llm_summarizer_keep_recent_count || ''}
                            onChange={(e) => setFormData({
                              ...formData,
                              llm_summarizer_keep_recent_count: e.target.value ? parseInt(e.target.value, 10) : undefined
                            })}
                            error={!!formErrors.llm_summarizer_keep_recent_count}
                            helperText={formErrors.llm_summarizer_keep_recent_count || 'Minimum number of recent agent runs to keep as-is before summarization'}
                            fullWidth
                            inputProps={{ min: 0 }}
                          />
                        </InputGroup>
                      </Box>
                    </Box>

                    <InputGroup>
                      <InputLabel>Summarization Prompt *</InputLabel>
                      <Select
                        id="llm_summarizer_prompt_id"
                        value={formData.llm_summarizer_prompt_id || ''}
                        onChange={async (e) => {
                          const promptId = e.target.value;
                          setFormData({ 
                            ...formData, 
                            llm_summarizer_prompt_id: promptId || undefined,
                            llm_summarizer_prompt_label: undefined 
                          });
                          // Load prompt versions when prompt is selected
                          const selectedPrompt = prompts.find(p => p.id === promptId);
                          if (selectedPrompt) {
                            await loadSummarizerPromptVersions(selectedPrompt.name);
                          } else {
                            setSummarizerPromptVersions([]);
                          }
                        }}
                        error={!!formErrors.llm_summarizer_prompt_id}
                        fullWidth
                        MenuProps={{
                          style: { zIndex: 1500 },
                          PaperProps: {
                            style: { zIndex: 1500 }
                          }
                        }}
                      >
                        <MenuItem value="" disabled>Select a prompt</MenuItem>
                        {prompts.map((prompt) => (
                          <MenuItem key={prompt.id} value={prompt.id}>
                            {prompt.name}
                          </MenuItem>
                        ))}
                      </Select>
                      {formErrors.llm_summarizer_prompt_id && <span style={{ color: 'red', fontSize: '0.875rem' }}>{formErrors.llm_summarizer_prompt_id}</span>}
                    </InputGroup>

                    {formData.llm_summarizer_prompt_id && (
                      <InputGroup>
                        <InputLabel>Summarization Prompt Label</InputLabel>
                        <Select
                          id="llm_summarizer_prompt_label"
                          value={formData.llm_summarizer_prompt_label || ''}
                          onChange={(e) => setFormData({ 
                            ...formData, 
                            llm_summarizer_prompt_label: e.target.value || undefined 
                          })}
                          fullWidth
                          MenuProps={{
                            style: { zIndex: 1500 },
                            PaperProps: {
                              style: { zIndex: 1500 }
                            }
                          }}
                        >
                          <MenuItem value="">Latest version (default)</MenuItem>
                          {summarizerPromptVersions
                            .filter(v => v.label && (v.label === 'latest' || v.label === 'production'))
                            .map((version) => (
                              <MenuItem key={version.id} value={version.label || ''}>
                                {version.label} (v{version.version})
                              </MenuItem>
                            ))}
                        </Select>
                        <p style={{ fontSize: '0.75rem', color: '#666', marginTop: '4px' }}>
                          Select a specific label (latest/production) or leave as default to use the latest version
                        </p>
                      </InputGroup>
                    )}

                    <InputGroup>
                      <InputLabel>Summarization Model *</InputLabel>
                      <Select
                        id="llm_summarizer_model_id"
                        value={formData.llm_summarizer_model_id || ''}
                        onChange={(e) => setFormData({ 
                          ...formData, 
                          llm_summarizer_model_id: e.target.value || undefined 
                        })}
                        error={!!formErrors.llm_summarizer_model_id}
                        fullWidth
                        MenuProps={{
                          style: { zIndex: 1500 },
                          PaperProps: {
                            style: { zIndex: 1500 }
                          }
                        }}
                      >
                        <MenuItem value="" disabled>Select a model for summarization</MenuItem>
                        {models.map((model) => (
                          <MenuItem key={model.id} value={model.id}>
                            {model.name} ({model.model_id})
                          </MenuItem>
                        ))}
                      </Select>
                      {formErrors.llm_summarizer_model_id && <span style={{ color: 'red', fontSize: '0.875rem' }}>{formErrors.llm_summarizer_model_id}</span>}
                      <p style={{ fontSize: '0.75rem', color: '#666', marginTop: '4px' }}>
                        Select a model to use for summarization
                      </p>
                    </InputGroup>
                  </>
                )}

                {formData.summarizer_type === 'sliding_window' && (
                  <InputGroup>
                    <InputLabel>Min Agent Run History to Keep *</InputLabel>
                    <Input
                      type="number"
                      value={formData.sliding_window_keep_count || ''}
                      onChange={(e) => setFormData({
                        ...formData,
                        sliding_window_keep_count: e.target.value ? parseInt(e.target.value, 10) : undefined
                      })}
                      error={!!formErrors.sliding_window_keep_count}
                      helperText={formErrors.sliding_window_keep_count || 'Minimum number of agent run history to keep (remaining will be discarded)'}
                      fullWidth
                      inputProps={{ min: 1 }}
                    />
                  </InputGroup>
                )}
              </>
            )}

            <InputGroup>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                <label style={{ fontSize: '0.875rem', fontWeight: 500 }}>MCP Servers</label>
                <Button
                  type="button"
                  onClick={handleAddMcpServer}
                  size="small"
                  startIcon={<Add />}
                >
                  Add MCP Server
                </Button>
              </Box>
              {formData.mcp_servers && formData.mcp_servers.map((mcpServer, index) => (
                <Box key={index} sx={{ mb: 2, p: 2, border: '1px solid rgba(255, 255, 255, 0.23);', borderRadius: 1 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                    <span style={{ fontSize: '0.875rem', fontWeight: 500 }}>MCP Server {index + 1}</span>
                    <IconButton
                      type="button"
                      onClick={() => handleRemoveMcpServer(index)}
                      size="small"
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Box>
                  <Box sx={{ mb: 1 }}>
                    <label htmlFor={`mcp_server_${index}`} style={{ fontSize: '0.875rem', fontWeight: 500, display: 'block', marginBottom: 0.5 }}>
                      MCP Server *
                    </label>
                    <Select
                      id={`mcp_server_${index}`}
                      value={mcpServer.mcp_server_id}
                      onChange={(e) => handleMcpServerChange(index, 'mcp_server_id', e.target.value)}
                      fullWidth
                      error={!!formErrors[`mcp_server_${index}`]}
                      MenuProps={{
                        style: { zIndex: 1500 },
                        PaperProps: {
                          style: { zIndex: 1500 }
                        }
                      }}
                    >
                      <MenuItem value="" disabled>Select an MCP server</MenuItem>
                      {mcpServers.map((server) => (
                        <MenuItem key={server.id} value={server.id}>
                          {server.name}
                        </MenuItem>
                      ))}
                    </Select>
                    {formErrors[`mcp_server_${index}`] && (
                      <span style={{ color: 'red', fontSize: '0.875rem' }}>{formErrors[`mcp_server_${index}`]}</span>
                    )}
                  </Box>
                  {mcpServer.mcp_server_id && (
                    <Box>
                      <label htmlFor={`tool_filters_${index}`} style={{ fontSize: '0.875rem', fontWeight: 500, display: 'block', marginBottom: 0.5 }}>
                        Tool Filters (Optional)
                      </label>
                      {loadingTools[mcpServer.mcp_server_id] ? (
                        <Box display="flex" alignItems="center" gap={1} sx={{ py: 1 }}>
                          <CircularProgress size={16} />
                          <span style={{ fontSize: '0.875rem', color: 'var(--text-secondary)' }}>Loading tools...</span>
                        </Box>
                      ) : toolErrors[mcpServer.mcp_server_id] ? (
                        <Box sx={{ py: 1 }}>
                          <span style={{ color: 'red', fontSize: '0.875rem' }}>{toolErrors[mcpServer.mcp_server_id]}</span>
                        </Box>
                      ) : (
                        <>
                          <Select
                            id={`tool_filters_${index}`}
                            multiple
                            value={mcpServer.tool_filters || []}
                            onChange={(e) => handleToolFiltersChange(index, e.target.value as string[])}
                            renderValue={(selected) => {
                              if ((selected as string[]).length === 0) {
                                return <span style={{ color: 'var(--text-secondary)' }}>All tools (no filter)</span>;
                              }
                              return (
                                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                                  {(selected as string[]).slice(0, 2).map((value) => (
                                    <Chip key={value} label={value} size="small" />
                                  ))}
                                  {(selected as string[]).length > 2 && (
                                    <Chip label={`+${(selected as string[]).length - 2} more`} size="small" />
                                  )}
                                </Box>
                              );
                            }}
                            fullWidth
                            size="small"
                            MenuProps={{
                              style: { zIndex: 1500 },
                              PaperProps: {
                                style: { zIndex: 1500, maxHeight: 300 }
                              }
                            }}
                          >
                            {mcpServerTools[mcpServer.mcp_server_id]?.length > 0 ? (
                              mcpServerTools[mcpServer.mcp_server_id].map((tool) => (
                                <MenuItem key={tool.name} value={tool.name}>
                                  <Checkbox checked={(mcpServer.tool_filters || []).indexOf(tool.name) > -1} />
                                  <ListItemText 
                                    primary={tool.name} 
                                    secondary={tool.description ? tool.description.substring(0, 80) + (tool.description.length > 80 ? '...' : '') : undefined}
                                  />
                                </MenuItem>
                              ))
                            ) : (
                              <MenuItem disabled>No tools available</MenuItem>
                            )}
                          </Select>
                          <Box sx={{ mt: 0.5 }}>
                            <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
                              {mcpServer.tool_filters && mcpServer.tool_filters.length > 0 
                                ? `${mcpServer.tool_filters.length} tool(s) selected`
                                : `All ${mcpServerTools[mcpServer.mcp_server_id]?.length || 0} tool(s) will be available`}
                            </span>
                          </Box>
                        </>
                      )}
                    </Box>
                  )}
                </Box>
              ))}
              {(!formData.mcp_servers || formData.mcp_servers.length === 0) && (
                <Box sx={{ p: 2, textAlign: 'center', color: '#999' }}>
                  No MCP servers added. Click "Add MCP Server" to add one.
                </Box>
              )}
            </InputGroup>

        </form>
      </SlideDialog>
    </PageContainer>
  );
};

