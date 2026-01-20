import React, { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router';
import { api } from '../../api';
import { AgentConfig, AgentConfigData, MCPServerConfig, ModelConfig, PromptConfig, SchemaConfig, HistoryConfig, SummarizerConfig, CreateAgentConfigRequest, UpdateAgentConfigRequest, AgentConfigAlias, CreateAliasRequest, UpdateAliasRequest } from './types';
import { ProviderType, ProviderModelsResponse, ModelParameters, ReasoningConfig, PromptWithLatestVersion, PromptVersion, JSONSchemaDefinition } from '../../components/Chat/types';
import { PageContainer, PageHeader, PageTitle } from '../../components/shared/Page';
import { Button } from '../../components/shared/Buttons';
import { Input, InputGroup, InputLabel, Select } from '../../components/shared/Input';
import { DataTable, Column, Action } from '../../components/DataTable/DataTable';
import { SlideDialog } from '../../components/shared/Dialog';
import {
  Box,
  Tabs,
  Tab,
  styled,
  MenuItem,
  Chip,
  IconButton,
  CircularProgress,
  FormControlLabel,
  Switch,
  alpha,
  Typography
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import SaveIcon from '@mui/icons-material/Save';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import CodeIcon from '@mui/icons-material/Code';
import VisibilityIcon from '@mui/icons-material/Visibility';
import Editor from '@monaco-editor/react';
import { SchemaBuilder } from './SchemaBuilder';

// Styled components
const TabsContainer = styled(Box)(() => ({
  borderBottom: '1px solid var(--border-color)',
  marginBottom: '24px',
}));

const TabPanel = styled(Box)(() => ({
  padding: '16px 0',
}));

const BackButton = styled(Box)(() => ({
  display: 'flex',
  alignItems: 'center',
  gap: '8px',
  color: 'var(--text-secondary)',
  cursor: 'pointer',
  fontSize: '14px',
  marginBottom: '16px',
  '&:hover': {
    color: 'var(--text-primary)',
  },
}));

const MCPServerCard = styled(Box)(() => ({
  border: '1px solid var(--border-color)',
  borderRadius: '8px',
  padding: '16px',
  marginBottom: '16px',
}));

const ModeToggleContainer = styled(Box)(({ theme }) => ({
  display: 'flex',
  gap: theme.spacing(1),
  marginBottom: theme.spacing(2),
  padding: theme.spacing(1),
  backgroundColor: alpha(theme.palette.background.paper, 0.5),
  borderRadius: theme.shape.borderRadius,
  border: `1px solid ${theme.palette.divider}`,
}));

const ModeButton = styled(Button)<{ active?: boolean }>(({ theme, active }) => ({
  flex: 1,
  backgroundColor: active ? theme.palette.primary.main : 'transparent',
  color: active ? theme.palette.primary.contrastText : theme.palette.text.secondary,
  '&:hover': {
    backgroundColor: active ? theme.palette.primary.dark : alpha(theme.palette.primary.main, 0.1),
  },
}));

const ConfigSection = styled(Box)(({ theme }) => ({
  border: `1px solid ${theme.palette.divider}`,
  borderRadius: theme.shape.borderRadius,
  padding: theme.spacing(2),
  marginBottom: theme.spacing(2),
}));

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function CustomTabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`agent-tabpanel-${index}`}
      aria-labelledby={`agent-tab-${index}`}
      {...other}
    >
      {value === index && <TabPanel>{children}</TabPanel>}
    </div>
  );
}

const defaultSchema: JSONSchemaDefinition = {
  type: 'object',
  properties: {},
  required: []
};

export const AgentBuilderDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isNew = id === 'new';
  
  const [config, setConfig] = useState<AgentConfig | null>(null);
  const [agentName, setAgentName] = useState<string>('');
  const [configId, setConfigId] = useState<string | null>(null);
  const [agentId, setAgentId] = useState<string | null>(null);
  const [formData, setFormData] = useState<AgentConfigData>({});
  const [originalData, setOriginalData] = useState<AgentConfigData>({});
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tabValue, setTabValue] = useState(0);
  const [hasChanges, setHasChanges] = useState(false);
  
  // Model parameters state (same as Models page)
  const [modelParameters, setModelParameters] = useState<ModelParameters>({});
  const [reasoningConfig, setReasoningConfig] = useState<ReasoningConfig>({});
  
  // Provider models for model selection
  const [providerModels, setProviderModels] = useState<ProviderModelsResponse['providers']>({});
  const [providerTypes, setProviderTypes] = useState<ProviderType[]>([]);

  // Prompts state
  const [prompts, setPrompts] = useState<PromptWithLatestVersion[]>([]);
  const [promptVersions, setPromptVersions] = useState<PromptVersion[]>([]);
  const [loadingPromptVersions, setLoadingPromptVersions] = useState(false);
  const [useRawPrompt, setUseRawPrompt] = useState(false);

  // Schema builder state
  const [useSchemaBuilder, setUseSchemaBuilder] = useState(true);
  const [schemaBuilderData, setSchemaBuilderData] = useState<JSONSchemaDefinition>(defaultSchema);

  // Summarizer prompt state
  const [summarizerPromptVersions, setSummarizerPromptVersions] = useState<PromptVersion[]>([]);
  const [loadingSummarizerPromptVersions, setLoadingSummarizerPromptVersions] = useState(false);
  const [useSummarizerRawPrompt, setUseSummarizerRawPrompt] = useState(false);

  // Summarizer model parameters state
  const [summarizerModelParameters, setSummarizerModelParameters] = useState<ModelParameters>({});
  const [summarizerReasoningConfig, setSummarizerReasoningConfig] = useState<ReasoningConfig>({});

  // Versions state
  const [versions, setVersions] = useState<AgentConfig[]>([]);
  const [loadingVersions, setLoadingVersions] = useState(false);

  // Aliases state
  const [aliases, setAliases] = useState<AgentConfigAlias[]>([]);
  const [loadingAliases, setLoadingAliases] = useState(false);
  const [editingAlias, setEditingAlias] = useState<AgentConfigAlias | null>(null);
  const [showAliasForm, setShowAliasForm] = useState(false);
  const [showVersion2, setShowVersion2] = useState(false);
  const [aliasFormData, setAliasFormData] = useState<CreateAliasRequest>({
    name: '',
    version1: 0,
    version2: undefined,
    weight: undefined
  });

  const loadConfig = useCallback(async () => {
    if (!id || isNew) return;

    try {
      setLoading(true);
      setError(null);
      const response = await api.get(`/agent-configs/${id}`);
      const configData = response.data.data;
      setConfig(configData);
      setConfigId(configData.id);
      setAgentName(configData.name);
      setAgentId(configData.agent_id);
      setFormData(configData.config || {});
      setOriginalData(JSON.parse(JSON.stringify(configData.config || {})));
      
      // Parse model parameters
      const params = configData.config?.model?.parameters || {};
      setModelParameters({
        temperature: params.temperature,
        top_p: params.top_p,
        max_output_tokens: params.max_output_tokens,
        max_tool_calls: params.max_tool_calls,
        parallel_tool_calls: params.parallel_tool_calls,
        top_logprobs: params.top_logprobs,
      });
      setReasoningConfig(params.reasoning || {});
      
      // Set prompt mode based on existing config
      const hasRawPrompt = configData.config?.prompt?.raw_prompt !== undefined;
      setUseRawPrompt(hasRawPrompt);
      
      // Set schema builder data
      if (configData.config?.schema?.schema) {
        setSchemaBuilderData(configData.config.schema.schema);
      }
      
      // Load prompt versions if a prompt is selected
      if (configData.config?.prompt?.prompt_id) {
        loadPromptVersions(configData.config.prompt.prompt_id);
      }

      // Load summarizer prompt versions and set mode
      const summarizerPrompt = configData.config?.history?.summarizer?.llm_summarizer_prompt;
      if (summarizerPrompt) {
        const hasSummarizerRawPrompt = summarizerPrompt.raw_prompt !== undefined;
        setUseSummarizerRawPrompt(hasSummarizerRawPrompt);
        if (summarizerPrompt.prompt_id) {
          loadSummarizerPromptVersions(summarizerPrompt.prompt_id);
        }
      }

      // Parse summarizer model parameters
      const summarizerParams = configData.config?.history?.summarizer?.llm_summarizer_model?.parameters || {};
      setSummarizerModelParameters({
        temperature: summarizerParams.temperature,
        top_p: summarizerParams.top_p,
        max_output_tokens: summarizerParams.max_output_tokens,
        max_tool_calls: summarizerParams.max_tool_calls,
        parallel_tool_calls: summarizerParams.parallel_tool_calls,
        top_logprobs: summarizerParams.top_logprobs,
      });
      setSummarizerReasoningConfig(summarizerParams.reasoning || {});
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to load agent config';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  }, [id, isNew]);

  const loadProviderModels = async () => {
    try {
      const response = await api.get('/providers/models');
      const data = response.data.data;
      if (data && data.providers) {
        setProviderModels(data.providers);
        setProviderTypes(Object.keys(data.providers) as ProviderType[]);
      }
    } catch (err: any) {
      console.error('Failed to load provider models:', err);
      setProviderTypes(['OpenAI', 'Anthropic', 'Gemini', 'xAI']);
    }
  };

  const loadPrompts = async () => {
    try {
      const response = await api.get('/prompts');
      setPrompts(response.data.data || []);
    } catch (err: any) {
      console.error('Failed to load prompts:', err);
    }
  };

  const loadPromptVersions = async (promptId: string) => {
    if (!promptId) {
      setPromptVersions([]);
      return;
    }
    
    try {
      setLoadingPromptVersions(true);
      const response = await api.get(`/prompts/versions?prompt_id=${promptId}`);
      const versions = response.data.data || [];
      setPromptVersions(versions);
    } catch (err: any) {
      console.error('Failed to load prompt versions:', err);
      setPromptVersions([]);
    } finally {
      setLoadingPromptVersions(false);
    }
  };

  const loadSummarizerPromptVersions = async (promptId: string) => {
    if (!promptId) {
      setSummarizerPromptVersions([]);
      return;
    }
    
    try {
      setLoadingSummarizerPromptVersions(true);
      const response = await api.get(`/prompts/versions?prompt_id=${promptId}`);
      const versions = response.data.data || [];
      setSummarizerPromptVersions(versions);
    } catch (err: any) {
      console.error('Failed to load summarizer prompt versions:', err);
      setSummarizerPromptVersions([]);
    } finally {
      setLoadingSummarizerPromptVersions(false);
    }
  };

  const loadVersions = async () => {
    if (!agentId || isNew) return;

    try {
      setLoadingVersions(true);
      const response = await api.get(`/agent-configs/${agentId}/versions`);
      const versionsData = response.data.data || [];
      setVersions(versionsData);
    } catch (err: any) {
      console.error('Failed to load versions:', err);
      setVersions([]);
    } finally {
      setLoadingVersions(false);
    }
  };

  const loadAliases = async () => {
    if (!agentId || isNew) return;

    try {
      setLoadingAliases(true);
      const response = await api.get(`/agent-configs/${agentId}/aliases`);
      const aliasesData = response.data.data || [];
      setAliases(aliasesData);
    } catch (err: any) {
      console.error('Failed to load aliases:', err);
      setAliases([]);
    } finally {
      setLoadingAliases(false);
    }
  };

  useEffect(() => {
    loadConfig();
    loadProviderModels();
    loadPrompts();
  }, [loadConfig, isNew, id]);

  // Load versions and aliases when configId is available
  useEffect(() => {
    if (!isNew && configId) {
      loadVersions();
      loadAliases();
    }
  }, [configId, isNew]);

  // Check for changes
  useEffect(() => {
    if (isNew) {
      // For new agents, has changes if name or config is set
      setHasChanges(agentName.trim() !== '' || JSON.stringify(formData) !== '{}');
    } else {
      const changed = JSON.stringify(formData) !== JSON.stringify(originalData);
      setHasChanges(changed);
    }
  }, [formData, originalData, isNew, agentName]);

  // Sync model parameters to formData
  useEffect(() => {
    const params: Record<string, any> = {};
    if (modelParameters.temperature !== undefined) params.temperature = modelParameters.temperature;
    if (modelParameters.top_p !== undefined) params.top_p = modelParameters.top_p;
    if (modelParameters.max_output_tokens !== undefined) params.max_output_tokens = modelParameters.max_output_tokens;
    if (modelParameters.max_tool_calls !== undefined) params.max_tool_calls = modelParameters.max_tool_calls;
    if (modelParameters.parallel_tool_calls !== undefined) params.parallel_tool_calls = modelParameters.parallel_tool_calls;
    if (modelParameters.top_logprobs !== undefined) params.top_logprobs = modelParameters.top_logprobs;
    
    if (reasoningConfig.effort || reasoningConfig.budget_tokens !== undefined) {
      params.reasoning = reasoningConfig;
    }

    setFormData(prev => ({
      ...prev,
      model: {
        ...prev.model,
        parameters: Object.keys(params).length > 0 ? params : undefined
      } as ModelConfig
    }));
  }, [modelParameters, reasoningConfig]);

  // Sync schema builder data to formData
  useEffect(() => {
    if (useSchemaBuilder) {
      setFormData(prev => ({
        ...prev,
        schema: {
          ...prev.schema,
          schema: schemaBuilderData
        } as SchemaConfig
      }));
    }
  }, [schemaBuilderData, useSchemaBuilder]);

  // Sync summarizer model parameters to formData
  useEffect(() => {
    if (formData.history?.summarizer?.type !== 'llm') return;

    const params: Record<string, any> = {};
    if (summarizerModelParameters.temperature !== undefined) params.temperature = summarizerModelParameters.temperature;
    if (summarizerModelParameters.top_p !== undefined) params.top_p = summarizerModelParameters.top_p;
    if (summarizerModelParameters.max_output_tokens !== undefined) params.max_output_tokens = summarizerModelParameters.max_output_tokens;
    if (summarizerModelParameters.max_tool_calls !== undefined) params.max_tool_calls = summarizerModelParameters.max_tool_calls;
    if (summarizerModelParameters.parallel_tool_calls !== undefined) params.parallel_tool_calls = summarizerModelParameters.parallel_tool_calls;
    if (summarizerModelParameters.top_logprobs !== undefined) params.top_logprobs = summarizerModelParameters.top_logprobs;
    
    if (summarizerReasoningConfig.effort || summarizerReasoningConfig.budget_tokens !== undefined) {
      params.reasoning = summarizerReasoningConfig;
    }

    setFormData(prev => ({
      ...prev,
      history: {
        ...prev.history,
        enabled: prev.history?.enabled ?? false,
        summarizer: {
          ...prev.history?.summarizer,
          type: prev.history?.summarizer?.type || 'none',
          llm_summarizer_model: {
            ...prev.history?.summarizer?.llm_summarizer_model,
            parameters: Object.keys(params).length > 0 ? params : undefined
          } as ModelConfig
        }
      }
    }));
  }, [summarizerModelParameters, summarizerReasoningConfig, formData.history?.summarizer?.type]);

  const handleSave = async () => {
    if (isNew) {
      // Create new agent
      if (!agentName.trim()) {
        setError('Agent name is required');
        return;
      }

      try {
        setSaving(true);
        setError(null);
        const request: CreateAgentConfigRequest = {
          name: agentName.trim(),
          config: formData
        };
        const response = await api.post('/agent-configs', request);
        // Navigate to the created agent using its ID
        const createdConfig = response.data.data;
        navigate(`/agent-framework/agents/${createdConfig.id}`, { replace: true });
      } catch (err: any) {
        const errorMessage = err.response?.data?.message ||
          err.response?.data?.errorDetails?.message ||
          'Failed to create agent config';
        setError(errorMessage);
      } finally {
        setSaving(false);
      }
    } else {
      // Update version 0 in place (mutable)
      if (!configId && !agentName) return;

      try {
        setSaving(true);
        setError(null);
        const request: UpdateAgentConfigRequest = {
          config: formData
        };
        if (configId) {
          await api.post(`/agent-configs/${configId}/versions`, request);
        } else {
          // Fallback to name-based API if configId is not available
          await api.post(`/agent-configs/by-name/versions?name=${encodeURIComponent(agentName)}`, request);
        }
        // Reload to get the updated version
        await loadConfig();
        await loadVersions();
      } catch (err: any) {
        const errorMessage = err.response?.data?.message ||
          err.response?.data?.errorDetails?.message ||
          'Failed to save agent config';
        setError(errorMessage);
      } finally {
        setSaving(false);
      }
    }
  };

  const handleCreateVersion = async () => {
    if (!configId || isNew) return;

    try {
      setSaving(true);
      setError(null);
      if (configId) {
        await api.post(`/agent-configs/${configId}/versions/create`);
      } else {
        // Fallback to name-based API if configId is not available
        await api.post(`/agent-configs/by-name/versions/create?name=${encodeURIComponent(agentName)}`);
      }
      // Reload to get the new version
      await loadConfig();
      await loadVersions();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to create version';
      setError(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  // Model config handlers
  const updateModelConfig = (field: keyof ModelConfig, value: any) => {
    setFormData(prev => ({
      ...prev,
      model: {
        ...prev.model,
        [field]: value
      } as ModelConfig
    }));
  };

  // Prompt config handlers
  const updatePromptConfig = (field: keyof PromptConfig, value: any) => {
    setFormData(prev => ({
      ...prev,
      prompt: {
        ...prev.prompt,
        [field]: value === '' ? undefined : value
      } as PromptConfig
    }));
  };

  // Schema config handlers
  const updateSchemaConfig = (field: keyof SchemaConfig, value: any) => {
    setFormData(prev => ({
      ...prev,
      schema: {
        ...prev.schema,
        [field]: value === '' ? undefined : value
      } as SchemaConfig
    }));
  };

  // MCP Server handlers
  const addMCPServer = () => {
    setFormData(prev => ({
      ...prev,
      mcp_servers: [
        ...(prev.mcp_servers || []),
        { name: '', endpoint: '', headers: {}, tool_filters: [], tools_requiring_human_approval: [] }
      ]
    }));
  };

  const removeMCPServer = (index: number) => {
    setFormData(prev => ({
      ...prev,
      mcp_servers: prev.mcp_servers?.filter((_, i) => i !== index)
    }));
  };

  const updateMCPServer = (index: number, field: keyof MCPServerConfig, value: any) => {
    setFormData(prev => ({
      ...prev,
      mcp_servers: prev.mcp_servers?.map((server, i) => 
        i === index ? { ...server, [field]: value } : server
      )
    }));
  };

  // Handle prompt mode switch
  const handlePromptModeChange = (useRaw: boolean) => {
    setUseRawPrompt(useRaw);
    if (useRaw) {
      setFormData(prev => ({
        ...prev,
        prompt: { raw_prompt: prev.prompt?.raw_prompt || '', prompt_id: undefined, label: undefined, version: undefined }
      }));
    } else {
      setFormData(prev => ({
        ...prev,
        prompt: { raw_prompt: undefined, prompt_id: prev.prompt?.prompt_id || '', label: prev.prompt?.label, version: prev.prompt?.version }
      }));
    }
  };

  // Handle prompt selection
  const handlePromptSelect = (promptId: string) => {
    updatePromptConfig('prompt_id', promptId);
    updatePromptConfig('label', undefined); // Reset label when changing prompt
    updatePromptConfig('version', undefined); // Reset version when changing prompt
    if (promptId) {
      loadPromptVersions(promptId);
    } else {
      setPromptVersions([]);
    }
  };

  // History config handlers
  const updateHistoryConfig = (enabled: boolean) => {
    setFormData(prev => ({
      ...prev,
      history: {
        enabled,
        summarizer: enabled ? (prev.history?.summarizer || { type: 'none' }) : undefined
      }
    }));
  };

  const updateSummarizerConfig = (field: keyof SummarizerConfig, value: any) => {
    setFormData(prev => ({
      ...prev,
      history: {
        ...prev.history,
        enabled: prev.history?.enabled ?? false,
        summarizer: {
          ...prev.history?.summarizer,
          type: prev.history?.summarizer?.type || 'none',
          [field]: value === '' ? undefined : value
        } as SummarizerConfig
      }
    }));
  };

  const updateSummarizerType = (type: 'none' | 'sliding_window' | 'llm') => {
    setFormData(prev => ({
      ...prev,
      history: {
        ...prev.history,
        enabled: prev.history?.enabled ?? false,
        summarizer: {
          type,
          // Clear fields when switching types
          sliding_window_keep_count: type === 'sliding_window' ? prev.history?.summarizer?.sliding_window_keep_count : undefined,
          llm_token_threshold: type === 'llm' ? prev.history?.summarizer?.llm_token_threshold : undefined,
          llm_keep_recent_count: type === 'llm' ? prev.history?.summarizer?.llm_keep_recent_count : undefined,
          llm_summarizer_prompt: type === 'llm' ? prev.history?.summarizer?.llm_summarizer_prompt : undefined,
          llm_summarizer_model: type === 'llm' ? prev.history?.summarizer?.llm_summarizer_model : undefined,
        }
      }
    }));

    // Reset summarizer prompt and model states when type changes
    if (type !== 'llm') {
      setSummarizerPromptVersions([]);
      setSummarizerModelParameters({});
      setSummarizerReasoningConfig({});
      setUseSummarizerRawPrompt(false);
    }
  };

  const updateSummarizerPromptConfig = (field: keyof PromptConfig, value: any) => {
    setFormData(prev => ({
      ...prev,
      history: {
        ...prev.history,
        enabled: prev.history?.enabled ?? false,
        summarizer: {
          ...prev.history?.summarizer,
          type: prev.history?.summarizer?.type || 'none',
          llm_summarizer_prompt: {
            ...prev.history?.summarizer?.llm_summarizer_prompt,
            [field]: value === '' ? undefined : value
          }
        } as SummarizerConfig
      }
    }));
  };

  const handleSummarizerPromptModeChange = (useRaw: boolean) => {
    setUseSummarizerRawPrompt(useRaw);
    if (useRaw) {
      setFormData(prev => ({
        ...prev,
        history: {
          ...prev.history,
          enabled: prev.history?.enabled ?? false,
          summarizer: {
            ...prev.history?.summarizer,
            type: prev.history?.summarizer?.type || 'none',
            llm_summarizer_prompt: { 
              raw_prompt: prev.history?.summarizer?.llm_summarizer_prompt?.raw_prompt || '', 
              prompt_id: undefined, 
              label: undefined, 
              version: undefined 
            }
          } as SummarizerConfig
        }
      }));
    } else {
      setFormData(prev => ({
        ...prev,
        history: {
          ...prev.history,
          enabled: prev.history?.enabled ?? false,
          summarizer: {
            ...prev.history?.summarizer,
            type: prev.history?.summarizer?.type || 'none',
            llm_summarizer_prompt: { 
              raw_prompt: undefined, 
              prompt_id: prev.history?.summarizer?.llm_summarizer_prompt?.prompt_id || '', 
              label: prev.history?.summarizer?.llm_summarizer_prompt?.label, 
              version: prev.history?.summarizer?.llm_summarizer_prompt?.version 
            }
          } as SummarizerConfig
        }
      }));
    }
  };

  const handleSummarizerPromptSelect = (promptId: string) => {
    updateSummarizerPromptConfig('prompt_id', promptId);
    updateSummarizerPromptConfig('label', undefined);
    updateSummarizerPromptConfig('version', undefined);
    if (promptId) {
      loadSummarizerPromptVersions(promptId);
    } else {
      setSummarizerPromptVersions([]);
    }
  };

  const updateSummarizerModelConfig = (field: keyof ModelConfig, value: any) => {
    setFormData(prev => ({
      ...prev,
      history: {
        ...prev.history,
        enabled: prev.history?.enabled ?? false,
        summarizer: {
          ...prev.history?.summarizer,
          type: prev.history?.summarizer?.type || 'none',
          llm_summarizer_model: {
            ...prev.history?.summarizer?.llm_summarizer_model,
            [field]: value
          } as ModelConfig
        } as SummarizerConfig
      }
    }));
  };

  console.log(formData);

  if (loading) {
    return (
      <PageContainer>
        <Box display="flex" justifyContent="center" alignItems="center" minHeight="200px">
          <CircularProgress />
        </Box>
      </PageContainer>
    );
  }

  if (error && !config && !isNew) {
    return (
      <PageContainer>
        <BackButton onClick={() => navigate('/agent-framework/agents')}>
          <ArrowBackIcon sx={{ fontSize: 18 }} />
          Back to Agent Builder
        </BackButton>
        <Box color="error.main" p={2}>
          {error}
        </Box>
      </PageContainer>
    );
  }

  return (
    <>
      <PageContainer>
        <BackButton onClick={() => navigate('/agent-framework/agents')}>
          <ArrowBackIcon sx={{ fontSize: 18 }} />
          Back to Agent Builder
        </BackButton>

        <PageHeader>
          <Box display="flex" alignItems="center" justifyContent="space-between" width="100%">
            <Box display="flex" alignItems="center" gap={2}>
              <PageTitle>{isNew ? 'New Agent' : (agentName || 'Agent')}</PageTitle>
              {config && !isNew && (
                <Chip 
                  label={`v${config.version}`} 
                  size="small" 
                  variant="outlined"
                  sx={{ borderColor: 'var(--border-color)' }}
                />
              )}
            </Box>
            <Box display="flex" alignItems="center" gap={2}>
              {hasChanges && (
                <Box display="flex" alignItems="center" gap={1} sx={{ 
                  px: 2, 
                  py: 1, 
                  borderRadius: 1, 
                  backgroundColor: alpha('#10a37f', 0.1),
                  color: '#10a37f'
                }}>
                  <Typography variant="body2" sx={{ fontSize: '14px' }}>
                    {isNew ? 'Create your new agent' : 'You have unsaved changes'}
                  </Typography>
                </Box>
              )}
              {hasChanges && (
                <Button
                  variant="contained"
                  onClick={handleSave}
                  disabled={saving || (isNew && !agentName.trim())}
                  startIcon={saving ? <CircularProgress size={16} sx={{ color: '#fff' }} /> : <SaveIcon />}
                  sx={{ 
                    backgroundColor: '#10a37f',
                    '&:hover': { backgroundColor: '#0d8a6d' }
                  }}
                >
                  {saving ? 'Saving...' : isNew ? 'Create Agent' : 'Save Changes'}
                </Button>
              )}
              {!isNew && config && !hasChanges && (
                <Button
                  variant="outlined"
                  onClick={handleCreateVersion}
                  disabled={saving || config.version !== 0}
                  startIcon={<AddIcon />}
                >
                  Create Version
                </Button>
              )}
            </Box>
          </Box>
        </PageHeader>

        {error && (
          <Box sx={{ mb: 2, p: 2, bgcolor: 'error.light', color: 'error.contrastText', borderRadius: 1 }}>
            {error}
          </Box>
        )}

        <TabsContainer>
          <Tabs 
            value={tabValue} 
            onChange={handleTabChange}
            sx={{
              '& .MuiTab-root': {
                textTransform: 'none',
                fontSize: '14px',
                fontWeight: 500,
              }
            }}
          >
            <Tab label="Agent Info" />
            <Tab label="Model" />
            <Tab label="System Prompt" />
            <Tab label="Output Schema" />
            <Tab label="MCP Servers" />
            <Tab label="History" />
            <Tab label="Versions" />
            <Tab label="Alias" />
          </Tabs>
        </TabsContainer>

        {/* Tab 0: Agent Information */}
        <CustomTabPanel value={tabValue} index={0}>
          <Box maxWidth="600px">
            <InputGroup>
              <InputLabel>Agent Name *</InputLabel>
              {isNew ? (
                <Input
                  value={agentName}
                  onChange={(e) => setAgentName(e.target.value)}
                  placeholder="e.g., Customer Support Agent"
                  fullWidth
                  autoFocus
                />
              ) : (
                <Input
                  value={agentName}
                  disabled
                  fullWidth
                  helperText="Agent name cannot be changed after creation"
                />
              )}
            </InputGroup>

            <InputGroup>
              <InputLabel>Max Iteration</InputLabel>
              <Input
                type="number"
                value={formData.max_iteration ?? ''}
                onChange={(e) => {
                  const value = e.target.value.trim();
                  const numValue = value === '' ? undefined : parseInt(value, 10);
                  setFormData(prev => ({
                    ...prev,
                    max_iteration: (numValue !== undefined && !isNaN(numValue) && numValue > 0) ? numValue : undefined
                  }));
                }}
                fullWidth
                inputProps={{ min: 1 }}
                helperText="Maximum number of iterations the agent can perform. Leave empty to run without limit."
                placeholder="e.g., 10"
              />
            </InputGroup>

            <InputGroup>
              <InputLabel>Runtime</InputLabel>
              <Select
                value={formData.runtime || ''}
                onChange={(e) => {
                  const runtime = e.target.value as 'Local' | 'Restate' | 'Temporal' | '';
                  setFormData(prev => ({
                    ...prev,
                    runtime: runtime === '' ? undefined : runtime
                  }));
                }}
                fullWidth
                displayEmpty
                MenuProps={{
                  style: { zIndex: 1500 },
                  PaperProps: { style: { zIndex: 1500 } }
                }}
              >
                <MenuItem value="">Select a runtime</MenuItem>
                <MenuItem value="Local">Local</MenuItem>
                <MenuItem value="Restate">Restate</MenuItem>
                <MenuItem value="Temporal">Temporal</MenuItem>
              </Select>
              <Box sx={{ mt: 1, color: 'var(--text-secondary)', fontSize: '12px' }}>
                Select the runtime environment for this agent.
              </Box>
            </InputGroup>
          </Box>
        </CustomTabPanel>

        {/* Tab 1: Model Configuration */}
        <CustomTabPanel value={tabValue} index={1}>
          <Box maxWidth="700px">
            <InputGroup>
              <InputLabel>Provider Type *</InputLabel>
              <Select
                value={formData.model?.provider_type || ''}
                onChange={(e) => {
                  updateModelConfig('provider_type', e.target.value);
                  updateModelConfig('model_id', ''); // Reset model when provider changes
                }}
                fullWidth
                displayEmpty
                MenuProps={{
                  style: { zIndex: 1500 },
                  PaperProps: { style: { zIndex: 1500 } }
                }}
              >
                <MenuItem value="" disabled>Select a provider</MenuItem>
                {providerTypes.map((provider) => (
                  <MenuItem key={provider} value={provider}>
                    {provider}
                  </MenuItem>
                ))}
              </Select>
            </InputGroup>

            <InputGroup>
              <InputLabel>Model ID *</InputLabel>
              <Select
                value={formData.model?.model_id || ''}
                onChange={(e) => updateModelConfig('model_id', e.target.value)}
                disabled={!formData.model?.provider_type}
                fullWidth
                displayEmpty
                MenuProps={{
                  style: { zIndex: 1500, maxHeight: 300 },
                  PaperProps: { style: { zIndex: 1500, maxHeight: 300 } }
                }}
              >
                <MenuItem value="" disabled>
                  {formData.model?.provider_type ? 'Select a model' : 'Select a provider first'}
                </MenuItem>
                {formData.model?.provider_type && providerModels[formData.model.provider_type]?.models?.map((modelName) => (
                  <MenuItem key={modelName} value={modelName}>
                    {modelName}
                  </MenuItem>
                ))}
              </Select>
            </InputGroup>

            <InputGroup>
              <InputLabel>Model Parameters (Optional)</InputLabel>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, p: 2, border: '1px solid rgba(255, 255, 255, 0.23)', borderRadius: 1 }}>
                <Box sx={{ display: 'flex', gap: 2 }}>
                  <Box sx={{ flex: 1 }}>
                    <InputLabel>Temperature</InputLabel>
                    <Input
                      type="number"
                      value={modelParameters.temperature ?? ''}
                      onChange={(e) => setModelParameters({
                        ...modelParameters,
                        temperature: e.target.value ? parseFloat(e.target.value) : undefined
                      })}
                      fullWidth
                      inputProps={{ min: 0, max: 2, step: 0.1 }}
                      helperText="Controls randomness (0.0-2.0)"
                    />
                  </Box>
                  <Box sx={{ flex: 1 }}>
                    <InputLabel>Top P</InputLabel>
                    <Input
                      type="number"
                      value={modelParameters.top_p ?? ''}
                      onChange={(e) => setModelParameters({
                        ...modelParameters,
                        top_p: e.target.value ? parseFloat(e.target.value) : undefined
                      })}
                      fullWidth
                      inputProps={{ min: 0, max: 1, step: 0.1 }}
                      helperText="Nucleus sampling (0.0-1.0)"
                    />
                  </Box>
                </Box>
                <Box sx={{ display: 'flex', gap: 2 }}>
                  <Box sx={{ flex: 1 }}>
                    <InputLabel>Max Output Tokens</InputLabel>
                    <Input
                      type="number"
                      value={modelParameters.max_output_tokens ?? ''}
                      onChange={(e) => setModelParameters({
                        ...modelParameters,
                        max_output_tokens: e.target.value ? parseInt(e.target.value, 10) : undefined
                      })}
                      fullWidth
                      inputProps={{ min: 1 }}
                      helperText="Maximum tokens in response"
                    />
                  </Box>
                  <Box sx={{ flex: 1 }}>
                    <InputLabel>Max Tool Calls</InputLabel>
                    <Input
                      type="number"
                      value={modelParameters.max_tool_calls ?? ''}
                      onChange={(e) => setModelParameters({
                        ...modelParameters,
                        max_tool_calls: e.target.value ? parseInt(e.target.value, 10) : undefined
                      })}
                      fullWidth
                      inputProps={{ min: 1 }}
                      helperText="Maximum tool calls per response"
                    />
                  </Box>
                </Box>
                <Box sx={{ display: 'flex', gap: 2 }}>
                  <Box sx={{ flex: 1 }}>
                    <InputLabel>Top Logprobs</InputLabel>
                    <Input
                      type="number"
                      value={modelParameters.top_logprobs ?? ''}
                      onChange={(e) => setModelParameters({
                        ...modelParameters,
                        top_logprobs: e.target.value ? parseInt(e.target.value, 10) : undefined
                      })}
                      fullWidth
                      inputProps={{ min: 0 }}
                      helperText="Number of most likely tokens to return"
                    />
                  </Box>
                  <Box sx={{ flex: 1 }}>
                    <FormControlLabel
                      control={
                        <Switch
                          checked={modelParameters.parallel_tool_calls ?? false}
                          onChange={(e) => setModelParameters({
                            ...modelParameters,
                            parallel_tool_calls: e.target.checked
                          })}
                        />
                      }
                      label="Parallel Tool Calls"
                    />
                    <p style={{ fontSize: '0.75rem', color: '#666', marginTop: '4px' }}>
                      Enable parallel tool call execution
                    </p>
                  </Box>
                </Box>
                <Box sx={{ display: 'flex', gap: 2 }}>
                  <Box sx={{ flex: 1 }}>
                    <InputLabel>Reasoning Effort</InputLabel>
                    <Select
                      value={reasoningConfig.effort || ''}
                      onChange={(e) => setReasoningConfig({
                        ...reasoningConfig,
                        effort: e.target.value as 'low' | 'medium' | 'high' | undefined || undefined
                      })}
                      fullWidth
                      displayEmpty
                      MenuProps={{
                        style: { zIndex: 1500 },
                        PaperProps: { style: { zIndex: 1500 } }
                      }}
                    >
                      <MenuItem value="">Default</MenuItem>
                      <MenuItem value="low">Low</MenuItem>
                      <MenuItem value="medium">Medium</MenuItem>
                      <MenuItem value="high">High</MenuItem>
                    </Select>
                    <p style={{ fontSize: '0.75rem', color: '#666', marginTop: '4px' }}>
                      Level of reasoning effort (low, medium, high)
                    </p>
                  </Box>
                  <Box sx={{ flex: 1 }}>
                    <InputLabel>Reasoning Budget (Tokens)</InputLabel>
                    <Input
                      type="number"
                      value={reasoningConfig.budget_tokens ?? ''}
                      onChange={(e) => setReasoningConfig({
                        ...reasoningConfig,
                        budget_tokens: e.target.value ? parseInt(e.target.value, 10) : undefined
                      })}
                      fullWidth
                      inputProps={{ min: 1 }}
                      helperText="Max tokens for reasoning (optional)"
                    />
                  </Box>
                </Box>
              </Box>
              <p style={{ fontSize: '0.75rem', color: '#666', marginTop: '4px' }}>
                Configure model parameters. Leave fields empty to use default values.
              </p>
            </InputGroup>
          </Box>
        </CustomTabPanel>

        {/* Tab 2: System Prompt */}
        <CustomTabPanel value={tabValue} index={2}>
          <Box>
            <ModeToggleContainer>
              <ModeButton 
                active={!useRawPrompt}
                onClick={() => handlePromptModeChange(false)}
                variant="text"
              >
                Use Prompt Reference
              </ModeButton>
              <ModeButton 
                active={useRawPrompt}
                onClick={() => handlePromptModeChange(true)}
                variant="text"
              >
                Use Raw Prompt Text
              </ModeButton>
            </ModeToggleContainer>

            {useRawPrompt ? (
              <InputGroup>
                <InputLabel>System Prompt</InputLabel>
                <Box sx={{ border: '1px solid var(--border-color)', borderRadius: 1, overflow: 'hidden' }}>
                  <Editor
                    height="400px"
                    defaultLanguage="markdown"
                    value={formData.prompt?.raw_prompt || ''}
                    onChange={(value) => updatePromptConfig('raw_prompt', value)}
                    options={{
                      minimap: { enabled: false },
                      scrollBeyondLastLine: false,
                      wordWrap: 'on',
                      fontSize: 14,
                      lineNumbers: 'on',
                      theme: 'vs-dark'
                    }}
                    theme="vs-dark"
                  />
                </Box>
                <Box sx={{ mt: 1, color: 'var(--text-secondary)', fontSize: '12px' }}>
                  Write your system prompt directly. Use {'{{variable}}'} for template variables.
                </Box>
              </InputGroup>
            ) : (
              <>
                <InputGroup>
                  <InputLabel>Select Prompt</InputLabel>
                  <Select
                    value={formData.prompt?.prompt_id || ''}
                    onChange={(e) => handlePromptSelect(e.target.value)}
                    fullWidth
                    displayEmpty
                    MenuProps={{
                      style: { zIndex: 1500 },
                      PaperProps: { style: { zIndex: 1500, maxHeight: 300 } }
                    }}
                  >
                    <MenuItem value="">Select a prompt</MenuItem>
                    {prompts.map((prompt) => (
                      <MenuItem key={prompt.id} value={prompt.id}>
                        <Box display="flex" alignItems="center" gap={1}>
                          <span>{prompt.name}</span>
                          {prompt.latest_label && (
                            <Chip label={prompt.latest_label} size="small" variant="outlined" />
                          )}
                        </Box>
                      </MenuItem>
                    ))}
                  </Select>
                  <Box sx={{ mt: 1, color: 'var(--text-secondary)', fontSize: '12px' }}>
                    Select an existing prompt from your library.
                  </Box>
                </InputGroup>

                {formData.prompt?.prompt_id && (
                  <InputGroup>
                    <InputLabel>Prompt Version</InputLabel>
                    {loadingPromptVersions ? (
                      <Box display="flex" alignItems="center" gap={1}>
                        <CircularProgress size={16} />
                        <span>Loading versions...</span>
                      </Box>
                    ) : (
                      <Select
                        value={formData.prompt?.version?.toString() || ''}
                        onChange={(e) => {
                          const versionNum = e.target.value ? parseInt(e.target.value, 10) : undefined;
                          updatePromptConfig('version', versionNum);
                          updatePromptConfig('label', undefined); // Clear label when selecting specific version
                        }}
                        fullWidth
                        displayEmpty
                        MenuProps={{
                          style: { zIndex: 1500 },
                          PaperProps: { style: { zIndex: 1500, maxHeight: 300 } }
                        }}
                      >
                        <MenuItem value="">Latest (default)</MenuItem>
                        {/* Show all versions sorted by version number (latest first) */}
                        {[...promptVersions]
                          .sort((a, b) => b.version - a.version)
                          .map((v) => (
                            <MenuItem key={v.version} value={v.version.toString()}>
                              <Box display="flex" alignItems="center" gap={1} width="100%">
                                <span style={{ fontWeight: 500 }}>v{v.version}</span>
                                {v.label && (
                                  <Chip label={v.label} size="small" color="primary" sx={{ ml: 1 }} />
                                )}
                                <span style={{ color: '#888', fontSize: '12px', marginLeft: 'auto' }}>
                                  {v.commit_message ? (v.commit_message.length > 40 ? v.commit_message.substring(0, 40) + '...' : v.commit_message) : ''}
                                </span>
                              </Box>
                            </MenuItem>
                          ))}
                      </Select>
                    )}
                    <Box sx={{ mt: 1, color: 'var(--text-secondary)', fontSize: '12px' }}>
                      Select a specific version or use "Latest" to always use the most recent version.
                    </Box>
                  </InputGroup>
                )}

                {/* Preview of selected prompt */}
                {formData.prompt?.prompt_id && promptVersions.length > 0 && (
                  <InputGroup>
                    <InputLabel>Preview</InputLabel>
                    <Box sx={{ 
                      border: '1px solid var(--border-color)', 
                      borderRadius: 1, 
                      p: 2, 
                      backgroundColor: 'rgba(0, 0, 0, 0.2)',
                      maxHeight: '200px',
                      overflow: 'auto'
                    }}>
                      <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: '13px', color: '#ccc' }}>
                        {(() => {
                          const selectedVersion = formData.prompt?.version 
                            ? promptVersions.find(v => v.version === formData.prompt?.version)
                            : promptVersions.reduce((latest, current) => 
                                current.version > latest.version ? current : latest
                              , promptVersions[0]);
                          return selectedVersion?.template || 'No preview available';
                        })()}
                      </pre>
                    </Box>
                  </InputGroup>
                )}
              </>
            )}
          </Box>
        </CustomTabPanel>

        {/* Tab 3: Output Schema */}
        <CustomTabPanel value={tabValue} index={3}>
          <Box>
            <ModeToggleContainer>
              <ModeButton 
                active={useSchemaBuilder}
                onClick={() => setUseSchemaBuilder(true)}
                variant="text"
              >
                Schema Builder
              </ModeButton>
              <ModeButton 
                active={!useSchemaBuilder}
                onClick={() => setUseSchemaBuilder(false)}
                variant="text"
                startIcon={<CodeIcon />}
              >
                Raw JSON
              </ModeButton>
            </ModeToggleContainer>

            <Box display="flex" gap={2} mb={3}>
              <Box flex={1}>
                <InputGroup>
                  <InputLabel>Schema Name</InputLabel>
                  <Input
                    value={formData.schema?.name || ''}
                    onChange={(e) => updateSchemaConfig('name', e.target.value)}
                    placeholder="e.g., CustomerResponse"
                    fullWidth
                  />
                </InputGroup>
              </Box>
              <Box flex={2}>
                <InputGroup>
                  <InputLabel>Description</InputLabel>
                  <Input
                    value={formData.schema?.description || ''}
                    onChange={(e) => updateSchemaConfig('description', e.target.value)}
                    placeholder="Optional description for the schema"
                    fullWidth
                  />
                </InputGroup>
              </Box>
            </Box>

            {useSchemaBuilder ? (
              <SchemaBuilder
                initialData={{
                  name: formData.schema?.name || '',
                  description: formData.schema?.description || '',
                  schema: schemaBuilderData,
                  source_type: 'manual'
                }}
                onSubmit={async () => {}}
                onCancel={() => {}}
                isEditing={true}
                embedded={true}
                onSchemaChange={(schema) => setSchemaBuilderData(schema)}
              />
            ) : (
              <InputGroup>
                <InputLabel>JSON Schema</InputLabel>
                <Box sx={{ border: '1px solid var(--border-color)', borderRadius: 1, overflow: 'hidden' }}>
                  <Editor
                    height="400px"
                    defaultLanguage="json"
                    value={JSON.stringify(formData.schema?.schema || {}, null, 2)}
                    onChange={(value) => {
                      try {
                        const parsed = JSON.parse(value || '{}');
                        updateSchemaConfig('schema', parsed);
                        setSchemaBuilderData(parsed);
                      } catch (e) {
                        // Invalid JSON, ignore
                      }
                    }}
                    options={{
                      minimap: { enabled: false },
                      scrollBeyondLastLine: false,
                      fontSize: 13,
                      lineNumbers: 'on',
                      theme: 'vs-dark'
                    }}
                    theme="vs-dark"
                  />
                </Box>
                <Box sx={{ mt: 1, color: 'var(--text-secondary)', fontSize: '12px' }}>
                  Define the JSON Schema for structured output. Leave empty for free-form responses.
                </Box>
              </InputGroup>
            )}
          </Box>
        </CustomTabPanel>

        {/* Tab 4: MCP Servers */}
        <CustomTabPanel value={tabValue} index={4}>
          <Box>
            <Box display="flex" justifyContent="flex-end" mb={2}>
              <Button
                variant="outlined"
                onClick={addMCPServer}
                startIcon={<AddIcon />}
              >
                Add MCP Server
              </Button>
            </Box>

            {formData.mcp_servers && formData.mcp_servers.length > 0 ? (
              formData.mcp_servers.map((server, index) => (
                <MCPServerCard key={index}>
                  <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                    <Box component="h4" sx={{ m: 0, fontSize: '14px', fontWeight: 500 }}>
                      MCP Server {index + 1}
                    </Box>
                    <IconButton onClick={() => removeMCPServer(index)} size="small">
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Box>

                  <Box display="flex" gap={2} mb={2}>
                    <Box flex={1}>
                      <InputLabel>Server Name *</InputLabel>
                      <Input
                        value={server.name}
                        onChange={(e) => updateMCPServer(index, 'name', e.target.value)}
                        placeholder="e.g., file-server"
                        fullWidth
                      />
                    </Box>
                    <Box flex={2}>
                      <InputLabel>Endpoint URL *</InputLabel>
                      <Input
                        value={server.endpoint}
                        onChange={(e) => updateMCPServer(index, 'endpoint', e.target.value)}
                        placeholder="https://mcp-server.example.com/sse"
                        fullWidth
                      />
                    </Box>
                  </Box>

                  <InputGroup>
                    <InputLabel>Headers (JSON)</InputLabel>
                    <Box sx={{ border: '1px solid var(--border-color)', borderRadius: 1, overflow: 'hidden' }}>
                      <Editor
                        height="100px"
                        defaultLanguage="json"
                        value={JSON.stringify(server.headers || {}, null, 2)}
                        onChange={(value) => {
                          try {
                            const parsed = JSON.parse(value || '{}');
                            updateMCPServer(index, 'headers', parsed);
                          } catch (e) {
                            // Invalid JSON, ignore
                          }
                        }}
                        options={{
                          minimap: { enabled: false },
                          scrollBeyondLastLine: false,
                          fontSize: 12,
                          lineNumbers: 'off',
                          theme: 'vs-dark'
                        }}
                        theme="vs-dark"
                      />
                    </Box>
                  </InputGroup>

                  <Box display="flex" gap={2}>
                    <Box flex={1}>
                      <InputLabel>Tool Filters (comma-separated)</InputLabel>
                      <Input
                        value={(server.tool_filters || []).join(', ')}
                        onChange={(e) => {
                          const filters = e.target.value
                            .split(',')
                            .map(s => s.trim())
                            .filter(s => s);
                          updateMCPServer(index, 'tool_filters', filters);
                        }}
                        placeholder="tool1, tool2, tool3"
                        fullWidth
                        helperText="Leave empty to allow all tools"
                      />
                    </Box>
                    <Box flex={1}>
                      <InputLabel>Tools Requiring Approval (comma-separated)</InputLabel>
                      <Input
                        value={(server.tools_requiring_human_approval || []).join(', ')}
                        onChange={(e) => {
                          const tools = e.target.value
                            .split(',')
                            .map(s => s.trim())
                            .filter(s => s);
                          updateMCPServer(index, 'tools_requiring_human_approval', tools);
                        }}
                        placeholder="dangerous_tool1, dangerous_tool2"
                        fullWidth
                        helperText="Tools that require human approval before execution"
                      />
                    </Box>
                  </Box>
                </MCPServerCard>
              ))
            ) : (
              <Box 
                sx={{ 
                  textAlign: 'center', 
                  py: 6, 
                  color: 'var(--text-secondary)',
                  border: '1px dashed var(--border-color)',
                  borderRadius: 1
                }}
              >
                <Box sx={{ fontSize: '14px', mb: 1 }}>No MCP servers configured</Box>
                <Box sx={{ fontSize: '12px' }}>Click "Add MCP Server" to connect to external tools</Box>
              </Box>
            )}
          </Box>
        </CustomTabPanel>

        {/* Tab 5: History */}
        <CustomTabPanel value={tabValue} index={5}>
          <Box maxWidth="800px">
            <InputGroup>
              <FormControlLabel
                control={
                  <Switch
                    checked={formData.history?.enabled || false}
                    onChange={(e) => updateHistoryConfig(e.target.checked)}
                  />
                }
                label="Enable Conversation History"
              />
              <Box sx={{ mt: 1, color: 'var(--text-secondary)', fontSize: '12px' }}>
                When enabled, the agent will maintain conversation history across interactions.
              </Box>
            </InputGroup>

            {formData.history?.enabled && (
              <>
                <InputGroup>
                  <InputLabel>Summarizer Type</InputLabel>
                  <Select
                    value={formData.history?.summarizer?.type || 'none'}
                    onChange={(e) => updateSummarizerType(e.target.value as 'none' | 'sliding_window' | 'llm')}
                    fullWidth
                    MenuProps={{
                      style: { zIndex: 1500 },
                      PaperProps: { style: { zIndex: 1500 } }
                    }}
                  >
                    <MenuItem value="none">No Summarization (Default)</MenuItem>
                    <MenuItem value="sliding_window">Sliding Window Summarizer</MenuItem>
                    <MenuItem value="llm">LLM Based Summarizer</MenuItem>
                  </Select>
                  <Box sx={{ mt: 1, color: 'var(--text-secondary)', fontSize: '12px' }}>
                    Choose how to manage conversation history as it grows.
                  </Box>
                </InputGroup>

                {/* Sliding Window Config */}
                {formData.history?.summarizer?.type === 'sliding_window' && (
                  <ConfigSection>
                    <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: 600 }}>
                      Sliding Window Configuration
                    </Typography>
                    <InputGroup>
                      <InputLabel>Min Run History to Keep *</InputLabel>
                      <Input
                        type="number"
                        value={formData.history?.summarizer?.sliding_window_keep_count ?? ''}
                        onChange={(e) => updateSummarizerConfig('sliding_window_keep_count', e.target.value ? parseInt(e.target.value, 10) : undefined)}
                        fullWidth
                        inputProps={{ min: 1 }}
                        helperText="Number of recent agent runs to keep (remaining will be discarded)"
                      />
                    </InputGroup>
                  </ConfigSection>
                )}

                {/* LLM Summarizer Config */}
                {formData.history?.summarizer?.type === 'llm' && (
                  <ConfigSection>
                    <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: 600 }}>
                      LLM Summarizer Configuration
                    </Typography>

                    <Box display="flex" gap={2}>
                      <Box flex={1}>
                        <InputGroup>
                          <InputLabel>Token Threshold *</InputLabel>
                          <Input
                            type="number"
                            value={formData.history?.summarizer?.llm_token_threshold ?? ''}
                            onChange={(e) => updateSummarizerConfig('llm_token_threshold', e.target.value ? parseInt(e.target.value, 10) : undefined)}
                            fullWidth
                            inputProps={{ min: 1 }}
                            helperText="Summarization triggers when token count exceeds this threshold"
                          />
                        </InputGroup>
                      </Box>
                      <Box flex={1}>
                        <InputGroup>
                          <InputLabel>Min Run History to Keep *</InputLabel>
                          <Input
                            type="number"
                            value={formData.history?.summarizer?.llm_keep_recent_count ?? ''}
                            onChange={(e) => updateSummarizerConfig('llm_keep_recent_count', e.target.value ? parseInt(e.target.value, 10) : undefined)}
                            fullWidth
                            inputProps={{ min: 0 }}
                            helperText="Minimum number of recent agent runs to keep as-is before summarization"
                          />
                        </InputGroup>
                      </Box>
                    </Box>

                    {/* Summarization Prompt */}
                    <Box sx={{ mt: 3, mb: 2 }}>
                      <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: 600 }}>
                        Summarization Prompt
                      </Typography>
                      
                      <ModeToggleContainer>
                        <ModeButton 
                          active={!useSummarizerRawPrompt}
                          onClick={() => handleSummarizerPromptModeChange(false)}
                          variant="text"
                        >
                          Use Prompt Reference
                        </ModeButton>
                        <ModeButton 
                          active={useSummarizerRawPrompt}
                          onClick={() => handleSummarizerPromptModeChange(true)}
                          variant="text"
                        >
                          Use Raw Prompt Text
                        </ModeButton>
                      </ModeToggleContainer>

                      {useSummarizerRawPrompt ? (
                        <InputGroup>
                          <InputLabel>Summarization Prompt</InputLabel>
                          <Box sx={{ border: '1px solid var(--border-color)', borderRadius: 1, overflow: 'hidden' }}>
                            <Editor
                              height="200px"
                              defaultLanguage="markdown"
                              value={formData.history?.summarizer?.llm_summarizer_prompt?.raw_prompt || ''}
                              onChange={(value) => updateSummarizerPromptConfig('raw_prompt', value)}
                              options={{
                                minimap: { enabled: false },
                                scrollBeyondLastLine: false,
                                wordWrap: 'on',
                                fontSize: 14,
                                lineNumbers: 'on',
                                theme: 'vs-dark'
                              }}
                              theme="vs-dark"
                            />
                          </Box>
                          <Box sx={{ mt: 1, color: 'var(--text-secondary)', fontSize: '12px' }}>
                            Write your summarization prompt directly. Use {'{{variable}}'} for template variables.
                          </Box>
                        </InputGroup>
                      ) : (
                        <>
                          <InputGroup>
                            <InputLabel>Select Prompt</InputLabel>
                            <Select
                              value={formData.history?.summarizer?.llm_summarizer_prompt?.prompt_id || ''}
                              onChange={(e) => handleSummarizerPromptSelect(e.target.value)}
                              fullWidth
                              displayEmpty
                              MenuProps={{
                                style: { zIndex: 1500 },
                                PaperProps: { style: { zIndex: 1500, maxHeight: 300 } }
                              }}
                            >
                              <MenuItem value="">Select a prompt</MenuItem>
                              {prompts.map((prompt) => (
                                <MenuItem key={prompt.id} value={prompt.id}>
                                  <Box display="flex" alignItems="center" gap={1}>
                                    <span>{prompt.name}</span>
                                    {prompt.latest_label && (
                                      <Chip label={prompt.latest_label} size="small" variant="outlined" />
                                    )}
                                  </Box>
                                </MenuItem>
                              ))}
                            </Select>
                          </InputGroup>

                          {formData.history?.summarizer?.llm_summarizer_prompt?.prompt_id && (
                            <InputGroup>
                              <InputLabel>Prompt Version</InputLabel>
                              {loadingSummarizerPromptVersions ? (
                                <Box display="flex" alignItems="center" gap={1}>
                                  <CircularProgress size={16} />
                                  <span>Loading versions...</span>
                                </Box>
                              ) : (
                                <Select
                                  value={formData.history?.summarizer?.llm_summarizer_prompt?.version?.toString() || ''}
                                  onChange={(e) => {
                                    const versionNum = e.target.value ? parseInt(e.target.value, 10) : undefined;
                                    updateSummarizerPromptConfig('version', versionNum);
                                    updateSummarizerPromptConfig('label', undefined);
                                  }}
                                  fullWidth
                                  displayEmpty
                                  MenuProps={{
                                    style: { zIndex: 1500 },
                                    PaperProps: { style: { zIndex: 1500, maxHeight: 300 } }
                                  }}
                                >
                                  <MenuItem value="">Latest (default)</MenuItem>
                                  {[...summarizerPromptVersions]
                                    .sort((a, b) => b.version - a.version)
                                    .map((v) => (
                                      <MenuItem key={v.version} value={v.version.toString()}>
                                        <Box display="flex" alignItems="center" gap={1} width="100%">
                                          <span style={{ fontWeight: 500 }}>v{v.version}</span>
                                          {v.label && (
                                            <Chip label={v.label} size="small" color="primary" sx={{ ml: 1 }} />
                                          )}
                                          <span style={{ color: '#888', fontSize: '12px', marginLeft: 'auto' }}>
                                            {v.commit_message ? (v.commit_message.length > 40 ? v.commit_message.substring(0, 40) + '...' : v.commit_message) : ''}
                                          </span>
                                        </Box>
                                      </MenuItem>
                                    ))}
                                </Select>
                              )}
                            </InputGroup>
                          )}

                          {/* Preview of selected summarizer prompt */}
                          {formData.history?.summarizer?.llm_summarizer_prompt?.prompt_id && summarizerPromptVersions.length > 0 && (
                            <InputGroup>
                              <InputLabel>Preview</InputLabel>
                              <Box sx={{ 
                                border: '1px solid var(--border-color)', 
                                borderRadius: 1, 
                                p: 2, 
                                backgroundColor: 'rgba(0, 0, 0, 0.2)',
                                maxHeight: '150px',
                                overflow: 'auto'
                              }}>
                                <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: '12px', color: '#ccc' }}>
                                  {(() => {
                                    const selectedVersion = formData.history?.summarizer?.llm_summarizer_prompt?.version 
                                      ? summarizerPromptVersions.find(v => v.version === formData.history?.summarizer?.llm_summarizer_prompt?.version)
                                      : summarizerPromptVersions.reduce((latest, current) => 
                                          current.version > latest.version ? current : latest
                                        , summarizerPromptVersions[0]);
                                    return selectedVersion?.template || 'No preview available';
                                  })()}
                                </pre>
                              </Box>
                            </InputGroup>
                          )}
                        </>
                      )}
                    </Box>

                    {/* Summarizer Model Configuration */}
                    <Box sx={{ mt: 3 }}>
                      <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: 600 }}>
                        Summarizer Model
                      </Typography>

                      <InputGroup>
                        <InputLabel>Provider Type *</InputLabel>
                        <Select
                          value={formData.history?.summarizer?.llm_summarizer_model?.provider_type || ''}
                          onChange={(e) => {
                            updateSummarizerModelConfig('provider_type', e.target.value);
                            updateSummarizerModelConfig('model_id', '');
                          }}
                          fullWidth
                          displayEmpty
                          MenuProps={{
                            style: { zIndex: 1500 },
                            PaperProps: { style: { zIndex: 1500 } }
                          }}
                        >
                          <MenuItem value="" disabled>Select a provider</MenuItem>
                          {providerTypes.map((provider) => (
                            <MenuItem key={provider} value={provider}>
                              {provider}
                            </MenuItem>
                          ))}
                        </Select>
                      </InputGroup>

                      <InputGroup>
                        <InputLabel>Model ID *</InputLabel>
                        <Select
                          value={formData.history?.summarizer?.llm_summarizer_model?.model_id || ''}
                          onChange={(e) => updateSummarizerModelConfig('model_id', e.target.value)}
                          disabled={!formData.history?.summarizer?.llm_summarizer_model?.provider_type}
                          fullWidth
                          displayEmpty
                          MenuProps={{
                            style: { zIndex: 1500, maxHeight: 300 },
                            PaperProps: { style: { zIndex: 1500, maxHeight: 300 } }
                          }}
                        >
                          <MenuItem value="" disabled>
                            {formData.history?.summarizer?.llm_summarizer_model?.provider_type ? 'Select a model' : 'Select a provider first'}
                          </MenuItem>
                          {formData.history?.summarizer?.llm_summarizer_model?.provider_type && 
                           providerModels[formData.history.summarizer.llm_summarizer_model.provider_type as ProviderType]?.models?.map((modelName) => (
                            <MenuItem key={modelName} value={modelName}>
                              {modelName}
                            </MenuItem>
                          ))}
                        </Select>
                      </InputGroup>

                      <InputGroup>
                        <InputLabel>Model Parameters (Optional)</InputLabel>
                        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, p: 2, border: '1px solid rgba(255, 255, 255, 0.23)', borderRadius: 1 }}>
                          <Box sx={{ display: 'flex', gap: 2 }}>
                            <Box sx={{ flex: 1 }}>
                              <InputLabel>Temperature</InputLabel>
                              <Input
                                type="number"
                                value={summarizerModelParameters.temperature ?? ''}
                                onChange={(e) => setSummarizerModelParameters({
                                  ...summarizerModelParameters,
                                  temperature: e.target.value ? parseFloat(e.target.value) : undefined
                                })}
                                fullWidth
                                inputProps={{ min: 0, max: 2, step: 0.1 }}
                                helperText="Controls randomness (0.0-2.0)"
                              />
                            </Box>
                            <Box sx={{ flex: 1 }}>
                              <InputLabel>Top P</InputLabel>
                              <Input
                                type="number"
                                value={summarizerModelParameters.top_p ?? ''}
                                onChange={(e) => setSummarizerModelParameters({
                                  ...summarizerModelParameters,
                                  top_p: e.target.value ? parseFloat(e.target.value) : undefined
                                })}
                                fullWidth
                                inputProps={{ min: 0, max: 1, step: 0.1 }}
                                helperText="Nucleus sampling (0.0-1.0)"
                              />
                            </Box>
                          </Box>
                          <Box sx={{ display: 'flex', gap: 2 }}>
                            <Box sx={{ flex: 1 }}>
                              <InputLabel>Max Output Tokens</InputLabel>
                              <Input
                                type="number"
                                value={summarizerModelParameters.max_output_tokens ?? ''}
                                onChange={(e) => setSummarizerModelParameters({
                                  ...summarizerModelParameters,
                                  max_output_tokens: e.target.value ? parseInt(e.target.value, 10) : undefined
                                })}
                                fullWidth
                                inputProps={{ min: 1 }}
                                helperText="Maximum tokens in response"
                              />
                            </Box>
                            <Box sx={{ flex: 1 }}>
                              <InputLabel>Reasoning Effort</InputLabel>
                              <Select
                                value={summarizerReasoningConfig.effort || ''}
                                onChange={(e) => setSummarizerReasoningConfig({
                                  ...summarizerReasoningConfig,
                                  effort: e.target.value as 'low' | 'medium' | 'high' | undefined || undefined
                                })}
                                fullWidth
                                displayEmpty
                                MenuProps={{
                                  style: { zIndex: 1500 },
                                  PaperProps: { style: { zIndex: 1500 } }
                                }}
                              >
                                <MenuItem value="">Default</MenuItem>
                                <MenuItem value="low">Low</MenuItem>
                                <MenuItem value="medium">Medium</MenuItem>
                                <MenuItem value="high">High</MenuItem>
                              </Select>
                            </Box>
                          </Box>
                        </Box>
                      </InputGroup>
                    </Box>
                  </ConfigSection>
                )}
              </>
            )}
          </Box>
        </CustomTabPanel>

        {/* Tab 6: Versions */}
        <CustomTabPanel value={tabValue} index={6}>
          <Box>
            {isNew ? (
              <Box sx={{ textAlign: 'center', py: 6, color: 'var(--text-secondary)' }}>
                <Typography variant="body1" sx={{ mb: 1 }}>
                  Versions will be available after creating the agent
                </Typography>
              </Box>
            ) : (
              <>
                {(() => {
                  const formatDate = (dateString: string) => {
                    if (!dateString) return '';
                    return new Date(dateString).toLocaleDateString('en-US', {
                      year: 'numeric',
                      month: 'short',
                      day: 'numeric',
                      hour: '2-digit',
                      minute: '2-digit'
                    });
                  };

                  const columns: Column<AgentConfig>[] = [
                    {
                      key: 'version',
                      label: 'Version',
                      render: (value, item) => (
                        <Box display="flex" alignItems="center" gap={1}>
                          <Chip 
                            label={item.version === 0 ? '$LATEST' : `v${item.version}`}
                            size="small"
                            color={item.version === 0 ? 'primary' : 'default'}
                            variant={item.version === 0 ? 'filled' : 'outlined'}
                          />
                          {item.immutable && (
                            <Chip 
                              label="Immutable"
                              size="small"
                              variant="outlined"
                              sx={{ fontSize: '10px', height: '20px' }}
                            />
                          )}
                        </Box>
                      )
                    },
                    {
                      key: 'created_at',
                      label: 'Created',
                      render: (value, item) => formatDate(item.created_at)
                    },
                    {
                      key: 'updated_at',
                      label: 'Updated',
                      render: (value, item) => formatDate(item.updated_at)
                    }
                  ];

                  const handleViewVersion = async (versionItem: AgentConfig) => {
                    if (!configId && !agentName) return;
                    
                    if (versionItem.version === 0) {
                      // If clicking on version 0, reload the current config
                      await loadConfig();
                      setTabValue(0); // Switch to Agent Info tab
                    } else {
                      // For immutable versions, load that specific version
                      try {
                        setLoading(true);
                        setError(null);
                        // Get config by agent_id and version
                        let configData;
                        if (config?.agent_id && agentName) {
                          // Use name-based API with version parameter
                          const response = await api.get(`/agent-configs/by-name?name=${encodeURIComponent(agentName)}&version=${versionItem.version}`);
                          configData = response.data.data;
                        } else if (agentName) {
                          // Fallback to name-based API
                          const response = await api.get(`/agent-configs/by-name?name=${encodeURIComponent(agentName)}&version=${versionItem.version}`);
                          configData = response.data.data;
                        } else {
                          throw new Error('Cannot load version: agent name is required');
                        }
                        
                        setConfig(configData);
                        setConfigId(configData.id);
                        setFormData(configData.config || {});
                        setOriginalData(JSON.parse(JSON.stringify(configData.config || {})));
                        
                        // Parse model parameters
                        const params = configData.config?.model?.parameters || {};
                        setModelParameters({
                          temperature: params.temperature,
                          top_p: params.top_p,
                          max_output_tokens: params.max_output_tokens,
                          max_tool_calls: params.max_tool_calls,
                          parallel_tool_calls: params.parallel_tool_calls,
                          top_logprobs: params.top_logprobs,
                        });
                        setReasoningConfig(params.reasoning || {});
                        
                        // Set prompt mode
                        const hasRawPrompt = configData.config?.prompt?.raw_prompt !== undefined;
                        setUseRawPrompt(hasRawPrompt);
                        
                        // Set schema builder data
                        if (configData.config?.schema?.schema) {
                          setSchemaBuilderData(configData.config.schema.schema);
                        }
                        
                        // Load prompt versions if a prompt is selected
                        if (configData.config?.prompt?.prompt_id) {
                          loadPromptVersions(configData.config.prompt.prompt_id);
                        }

                        // Load summarizer prompt versions
                        const summarizerPrompt = configData.config?.history?.summarizer?.llm_summarizer_prompt;
                        if (summarizerPrompt) {
                          const hasSummarizerRawPrompt = summarizerPrompt.raw_prompt !== undefined;
                          setUseSummarizerRawPrompt(hasSummarizerRawPrompt);
                          if (summarizerPrompt.prompt_id) {
                            loadSummarizerPromptVersions(summarizerPrompt.prompt_id);
                          }
                        }

                        // Parse summarizer model parameters
                        const summarizerParams = configData.config?.history?.summarizer?.llm_summarizer_model?.parameters || {};
                        setSummarizerModelParameters({
                          temperature: summarizerParams.temperature,
                          top_p: summarizerParams.top_p,
                          max_output_tokens: summarizerParams.max_output_tokens,
                          max_tool_calls: summarizerParams.max_tool_calls,
                          parallel_tool_calls: summarizerParams.parallel_tool_calls,
                          top_logprobs: summarizerParams.top_logprobs,
                        });
                        setSummarizerReasoningConfig(summarizerParams.reasoning || {});
                        
                        setTabValue(0); // Switch to Agent Info tab
                      } catch (err: any) {
                        const errorMessage = err.response?.data?.message ||
                          err.response?.data?.errorDetails?.message ||
                          'Failed to load version';
                        setError(errorMessage);
                      } finally {
                        setLoading(false);
                      }
                    }
                  };

                  const actions: Action<AgentConfig>[] = [
                    {
                      label: 'View',
                      onClick: handleViewVersion,
                      icon: <VisibilityIcon />
                    }
                  ];

                  return (
                    <DataTable
                      data={versions}
                      columns={columns}
                      actions={actions}
                      loading={loadingVersions}
                      emptyState={{
                        icon: '📋',
                        title: 'No versions yet',
                        description: 'Versions will appear here after you create immutable versions from version 0.',
                        actionLabel: 'Refresh',
                        onAction: loadVersions
                      }}
                    />
                  );
                })()}
              </>
            )}
          </Box>
        </CustomTabPanel>

        {/* Tab 7: Alias */}
        <CustomTabPanel value={tabValue} index={7}>
          <Box>
            {isNew ? (
              <Box sx={{ textAlign: 'center', py: 6, color: 'var(--text-secondary)' }}>
                <Typography variant="body1" sx={{ mb: 1 }}>
                  Aliases will be available after creating the agent
                </Typography>
              </Box>
            ) : (
              <>
                <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="h6">Aliases</Typography>
                  <Button
                    variant="contained"
                    onClick={() => {
                      setEditingAlias(null);
                      setShowAliasForm(true);
                      setShowVersion2(false);
                      setAliasFormData({
                        name: '',
                        version1: versions.length > 0 ? versions[0].version : 0,
                        version2: undefined,
                        weight: undefined
                      });
                    }}
                    startIcon={<AddIcon />}
                  >
                    Create Alias
                  </Button>
                </Box>

                {(() => {
                  const formatDate = (dateString: string) => {
                    if (!dateString) return '';
                    return new Date(dateString).toLocaleDateString('en-US', {
                      year: 'numeric',
                      month: 'short',
                      day: 'numeric',
                      hour: '2-digit',
                      minute: '2-digit'
                    });
                  };

                  const handleEditAlias = (alias: AgentConfigAlias) => {
                    setEditingAlias(alias);
                    setShowAliasForm(true);
                    setShowVersion2(alias.version2 !== undefined);
                    setAliasFormData({
                      name: alias.name,
                      version1: alias.version1,
                      version2: alias.version2,
                      weight: alias.weight
                    });
                  };

                  const handleDeleteAlias = async (aliasId: string) => {
                    if (!window.confirm('Are you sure you want to delete this alias?')) {
                      return;
                    }

                    try {
                      await api.delete(`/agent-configs/aliases/${aliasId}`);
                      await loadAliases();
                    } catch (err: any) {
                      const errorMessage = err.response?.data?.message ||
                        err.response?.data?.errorDetails?.message ||
                        'Failed to delete alias';
                      setError(errorMessage);
                    }
                  };

                  const handleSaveAlias = async () => {
                    if (!aliasFormData.name.trim()) {
                      setError('Alias name is required');
                      return;
                    }

                    if (aliasFormData.version1 === 0) {
                      setError('Version 1 is required');
                      return;
                    }

                    if (showVersion2) {
                      if (aliasFormData.version2 === undefined) {
                        setError('Version 2 is required when enabled');
                        return;
                      }
                      if (aliasFormData.weight === undefined) {
                        setError('Weight is required when version 2 is set');
                        return;
                      }
                    }

                    try {
                      if (editingAlias) {
                        const updateReq: UpdateAliasRequest = {
                          name: aliasFormData.name,
                          version1: aliasFormData.version1,
                          version2: aliasFormData.version2,
                          weight: aliasFormData.weight
                        };
                        await api.put(`/agent-configs/aliases/${editingAlias.id}`, updateReq);
                      } else {
                        // Use config ID if available, otherwise fall back to name
                        if (configId) {
                          await api.post(`/agent-configs/${configId}/aliases`, aliasFormData);
                        } else if (agentName) {
                          await api.post(`/agent-configs/by-name/aliases?name=${encodeURIComponent(agentName)}`, aliasFormData);
                        } else {
                          throw new Error('Cannot create alias: config ID or name is required');
                        }
                      }
                      await loadAliases();
                      setEditingAlias(null);
                      setShowAliasForm(false);
                      setShowVersion2(false);
                      setAliasFormData({
                        name: '',
                        version1: 0,
                        version2: undefined,
                        weight: undefined
                      });
                    } catch (err: any) {
                      const errorMessage = err.response?.data?.message ||
                        err.response?.data?.errorDetails?.message ||
                        'Failed to save alias';
                      setError(errorMessage);
                    }
                  };

                  const columns: Column<AgentConfigAlias>[] = [
                    {
                      key: 'name',
                      label: 'Name',
                      render: (value, item) => item.name
                    },
                    {
                      key: 'version1',
                      label: 'Version 1',
                      render: (value, item) => (
                        <Chip 
                          label={item.version1 === 0 ? '$LATEST' : `v${item.version1}`}
                          size="small"
                          color={item.version1 === 0 ? 'primary' : 'default'}
                          variant={item.version1 === 0 ? 'filled' : 'outlined'}
                        />
                      )
                    },
                    {
                      key: 'version2',
                      label: 'Version 2',
                      render: (value, item) => item.version2 ? (
                        <Box display="flex" alignItems="center" gap={1}>
                          <Chip 
                            label={`v${item.version2}`}
                            size="small"
                            variant="outlined"
                          />
                          {item.weight !== undefined && (
                            <Chip 
                              label={`${item.weight}%`}
                              size="small"
                              variant="outlined"
                              sx={{ fontSize: '10px', height: '20px' }}
                            />
                          )}
                        </Box>
                      ) : (
                        <span style={{ color: 'var(--text-secondary)' }}>None</span>
                      )
                    },
                    {
                      key: 'created_at',
                      label: 'Created',
                      render: (value, item) => formatDate(item.created_at)
                    }
                  ];

                  const actions: Action<AgentConfigAlias>[] = [
                    {
                      label: 'Edit',
                      onClick: handleEditAlias,
                      icon: <EditIcon />
                    },
                    {
                      label: 'Delete',
                      onClick: (item) => handleDeleteAlias(item.id),
                      icon: <DeleteIcon />
                    }
                  ];

                  const handleCloseAliasDialog = () => {
                    setEditingAlias(null);
                    setShowAliasForm(false);
                    setShowVersion2(false);
                    setAliasFormData({
                      name: '',
                      version1: 0,
                      version2: undefined,
                      weight: undefined
                    });
                  };

                  return (
                    <>
                      <DataTable
                        data={aliases}
                        columns={columns}
                        actions={actions}
                        loading={loadingAliases}
                        emptyState={{
                          icon: '',
                          title: 'No aliases yet',
                          description: 'Create an alias to map to one or two versions of this agent.',
                          actionLabel: 'Create Alias',
                          onAction: () => {
                            setEditingAlias(null);
                            setShowAliasForm(true);
                            setShowVersion2(false);
                            setAliasFormData({
                              name: '',
                              version1: versions.length > 0 ? versions[0].version : 0,
                              version2: undefined,
                              weight: undefined
                            });
                          }
                        }}
                      />

                      <SlideDialog
                        open={showAliasForm}
                        onClose={handleCloseAliasDialog}
                        title={editingAlias ? 'Edit Alias' : 'Create Alias'}
                        maxWidth="600px"
                        actions={
                          <>
                            <Button
                              type="button"
                              onClick={handleCloseAliasDialog}
                            >
                              Cancel
                            </Button>
                            <Button
                              type="submit"
                              variant="contained"
                              form="alias-form"
                            >
                              {editingAlias ? 'Update' : 'Create'}
                            </Button>
                          </>
                        }
                      >
                        <form 
                          id="alias-form" 
                          onSubmit={(e) => {
                            e.preventDefault();
                            handleSaveAlias();
                          }}
                        >
                          <InputGroup>
                            <InputLabel>Alias Name *</InputLabel>
                            <Input
                              value={aliasFormData.name}
                              onChange={(e) => setAliasFormData({ ...aliasFormData, name: e.target.value })}
                              fullWidth
                              required
                            />
                          </InputGroup>

                          <InputGroup>
                            <InputLabel>Version 1 *</InputLabel>
                            <Select
                              value={aliasFormData.version1}
                              onChange={(e) => setAliasFormData({ ...aliasFormData, version1: parseInt(e.target.value as string) })}
                              fullWidth
                              required
                              MenuProps={{
                                style: { zIndex: 1500 },
                                PaperProps: {
                                  style: { zIndex: 1500 }
                                }
                              }}
                            >
                              {versions.map((v) => (
                                <MenuItem key={v.version} value={v.version}>
                                  {v.version === 0 ? '$LATEST' : `v${v.version}`}
                                </MenuItem>
                              ))}
                            </Select>
                          </InputGroup>

                          <InputGroup>
                            <FormControlLabel
                              control={
                                <Switch
                                  checked={showVersion2}
                                  onChange={(e) => {
                                    setShowVersion2(e.target.checked);
                                    if (!e.target.checked) {
                                      setAliasFormData({ ...aliasFormData, version2: undefined, weight: undefined });
                                    }
                                  }}
                                />
                              }
                              label="Add Version 2 (Optional)"
                            />
                          </InputGroup>

                          {showVersion2 && (
                            <>
                              <InputGroup>
                                <InputLabel>Version 2</InputLabel>
                                <Select
                                  value={aliasFormData.version2 !== undefined ? aliasFormData.version2 : ''}
                                  onChange={(e) => setAliasFormData({ 
                                    ...aliasFormData, 
                                    version2: e.target.value ? parseInt(e.target.value as string) : undefined 
                                  })}
                                  fullWidth
                                  MenuProps={{
                                    style: { zIndex: 1500 },
                                    PaperProps: {
                                      style: { zIndex: 1500 }
                                    }
                                  }}
                                >
                                  <MenuItem value="">Select version</MenuItem>
                                  {versions
                                    .filter(v => v.version !== aliasFormData.version1)
                                    .map((v) => (
                                      <MenuItem key={v.version} value={v.version}>
                                        v{v.version}
                                      </MenuItem>
                                    ))}
                                </Select>
                              </InputGroup>

                              <InputGroup>
                                <InputLabel>Weight (0-100) *</InputLabel>
                                <Input
                                  type="number"
                                  value={aliasFormData.weight || ''}
                                  onChange={(e) => setAliasFormData({ 
                                    ...aliasFormData, 
                                    weight: e.target.value ? parseFloat(e.target.value) : undefined 
                                  })}
                                  fullWidth
                                  inputProps={{ min: 0, max: 100, step: 0.1 }}
                                  helperText="Weight percentage for version 2 (0-100)"
                                  required
                                />
                              </InputGroup>
                            </>
                          )}
                        </form>
                      </SlideDialog>
                    </>
                  );
                })()}
              </>
            )}
          </Box>
        </CustomTabPanel>
      </PageContainer>
    </>
  );
};
