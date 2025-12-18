import React, {useEffect, useState} from 'react';
import {api} from '../../api';
import {
  CreateModelRequest,
  ModelWithProvider,
  UpdateModelRequest,
  ProviderType,
  ProviderModelsResponse,
  ProviderModelsData,
  ModelParameters,
  ReasoningConfig
} from '../../components/Chat/types';
import {Action, Column, DataTable} from '../../components/DataTable/DataTable';
import {PageContainer, PageHeader, PageSubtitle, PageTitle} from "../../components/shared/Page";
import {Button} from "../../components/shared/Buttons";
import {Box, IconButton, MenuItem, FormControlLabel, Switch} from '@mui/material';
import Close from '@mui/icons-material/Close';
import Edit from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import {Input, InputGroup, InputLabel, Select} from '../../components/shared/Input';
import {SlideDialog} from "../../components/shared/Dialog";
import {OpenAIIcon} from '../../Icons/OpenAI';
import {AnthropicIcon} from '../../Icons/Anthropic';
import {GeminiIcon} from '../../Icons/Gemini';
import {XAIIcon} from '../../Icons/XAI';

export const Models: React.FC = props => {
  const [models, setModels] = useState<ModelWithProvider[]>([]);
  const [providerModels, setProviderModels] = useState<ProviderModelsResponse['providers']>({});
  const [providerTypes, setProviderTypes] = useState<ProviderType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showDialog, setShowDialog] = useState(false);
  const [editingModel, setEditingModel] = useState<ModelWithProvider | null>(null);
  const [formData, setFormData] = useState<CreateModelRequest>({
    provider_type: "OpenAI",
    name: '',
    model_id: '',
    parameters: {},
  });
  const [modelParameters, setModelParameters] = useState<ModelParameters>({});
  const [reasoningConfig, setReasoningConfig] = useState<ReasoningConfig>({});
  const [formErrors, setFormErrors] = useState<{ [key: string]: string }>({});

  // Load models and provider models on component mount
  useEffect(() => {
    loadModels();
    loadProviderModels();
  }, []);

  // Debug: Log when providerModels or formData.provider_type changes
  useEffect(() => {
    if (formData.provider_type) {
      console.log('Provider type:', formData.provider_type);
      console.log('Provider models:', providerModels);
      console.log('Models for provider:', providerModels[formData.provider_type]);
    }
  }, [formData.provider_type, providerModels]);

  const loadModels = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await api.get('/models');
      setModels(response.data.data || []);
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to load models';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const loadProviderModels = async () => {
    try {
      const response = await api.get('/providers/models');
      const data = response.data.data;
      if (data && data.providers) {
        setProviderModels(data.providers);
        const types = Object.keys(data.providers) as ProviderType[];
        setProviderTypes(types);
      }
    } catch (err: any) {
      console.error('Failed to load provider models:', err);
      // Fallback to default provider types
      setProviderTypes(['OpenAI', 'Anthropic', 'Gemini', 'xAI']);
    }
  };

  const handleCreate = () => {
    setEditingModel(null);
    setFormData({
      provider_type: providerTypes[0] || 'OpenAI',
      name: '',
      model_id: '',
      parameters: {},
    });
    setModelParameters({});
    setReasoningConfig({});
    setFormErrors({});
    setShowDialog(true);
  };

  const handleEdit = (model: ModelWithProvider) => {
    setEditingModel(model);
    setFormData({
      provider_type: model.provider_type,
      name: model.name,
      model_id: model.model_id,
      parameters: model.parameters || {},
    });
    // Convert parameters map to ModelParameters interface
    const params = model.parameters || {};
    const standardParams: ModelParameters = {
      temperature: params.temperature as number | undefined,
      top_p: params.top_p as number | undefined,
      max_output_tokens: params.max_output_tokens as number | undefined,
      max_tool_calls: params.max_tool_calls as number | undefined,
      parallel_tool_calls: params.parallel_tool_calls as boolean | undefined,
      top_logprobs: params.top_logprobs as number | undefined,
    };
    setModelParameters(standardParams);
    
    // Extract reasoning config
    const reasoning = (params.reasoning as ReasoningConfig) || {};
    setReasoningConfig(reasoning);
    setFormErrors({});
    setShowDialog(true);
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this model?')) {
      return;
    }

    try {
      await api.delete(`/models/${id}`);
      await loadModels();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to delete model';
      setError(errorMessage);
    }
  };

  const validateForm = (): boolean => {
    const errors: { [key: string]: string } = {};

    if (!formData.name.trim()) {
      errors.name = 'Name is required';
    }

    if (!formData.model_id.trim()) {
      errors.model_id = 'Model ID is required';
    }

    if (!formData.provider_type) {
      errors.provider_type = 'Provider type is required';
    }

    // Convert model parameters to parameters map
    const params: any = {};
    if (modelParameters.temperature !== undefined) params.temperature = modelParameters.temperature;
    if (modelParameters.top_p !== undefined) params.top_p = modelParameters.top_p;
    if (modelParameters.max_output_tokens !== undefined) params.max_output_tokens = modelParameters.max_output_tokens;
    if (modelParameters.max_tool_calls !== undefined) params.max_tool_calls = modelParameters.max_tool_calls;
    if (modelParameters.parallel_tool_calls !== undefined) params.parallel_tool_calls = modelParameters.parallel_tool_calls;
    if (modelParameters.top_logprobs !== undefined) params.top_logprobs = modelParameters.top_logprobs;
    
    // Add reasoning config if any field is set
    if (reasoningConfig.effort || reasoningConfig.budget_tokens !== undefined) {
      params.reasoning = reasoningConfig;
    }
    
    formData.parameters = params;

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    try {
      if (editingModel) {
        // Update existing model
        const updateData: UpdateModelRequest = {
          provider_type: formData.provider_type,
          name: formData.name,
          model_id: formData.model_id,
          parameters: formData.parameters,
        };
        await api.put(`/models/${editingModel.id}`, updateData);
      } else {
        // Create new model
        await api.post('/models', formData);
      }

      setShowDialog(false);
      await loadModels();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to save model';
      setError(errorMessage);
    }
  };

  const handleCloseDialog = () => {
    setShowDialog(false);
    setEditingModel(null);
    setFormData({
      provider_type: '' as any,
      name: '',
      model_id: '',
      parameters: {},
    });
    setModelParameters({});
    setReasoningConfig({});
    setFormErrors({});
  };

  // Provider icons mapping
  const providerIcons: Record<ProviderType, React.ReactElement> = {
    OpenAI: <OpenAIIcon />,
    Anthropic: <AnthropicIcon />,
    Gemini: <GeminiIcon />,
    xAI: <XAIIcon />,
  };

  const getProviderIcon = (providerType: ProviderType) => {
    return providerIcons[providerType] || null;
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

  const getProviderName = (providerType: ProviderType) => {
    return providerType.charAt(0).toUpperCase() + providerType.slice(1);
  };

  // Table configuration
  const columns: Column<ModelWithProvider>[] = [
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
      key: 'provider_type',
      label: 'Provider',
      render: (value, item) => (
        <Box display="flex" alignItems="center" gap={1}>
          {getProviderIcon(item.provider_type)}
          <span>{getProviderName(item.provider_type)}</span>
        </Box>
      )
    },
    {
      key: 'model_id',
      label: 'Model ID',
      render: (value, item) => (
        <div>
          {item.model_id}
        </div>
      )
    },
    {
      key: 'parameters',
      label: 'Parameters',
      render: (value, item) => (
        <div>
          <code style={{fontSize: '0.775rem'}}>
            {Object.keys(item.parameters || {}).length > 0 
              ? JSON.stringify(item.parameters).substring(0, 100) + (JSON.stringify(item.parameters).length > 100 ? '...' : '')
              : '-'}
          </code>
        </div>
      )
    },
    {
      key: 'created_at',
      label: 'Created',
      render: (value, item) => (
        <div>
          {formatDate(item.created_at)}
        </div>
      )
    }
  ];

  const actions: Action<ModelWithProvider>[] = [
    {
      label: 'Edit',
      icon: <Edit />,
      onClick: handleEdit,
      title: 'Edit'
    },
    {
      label: 'Delete',
      icon: <DeleteIcon />,
      onClick: (item) => handleDelete(item.id),
      className: 'deleteButton',
      title: 'Delete'
    }
  ];

  return (
    <PageContainer>
      <PageHeader>
        <div>
          <PageTitle>Models</PageTitle>
          <PageSubtitle>Manage your AI models and their parameters</PageSubtitle>
        </div>
        <Button color="primary" variant="contained" onClick={handleCreate}>
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path
              d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
          </svg>
          Create Model
        </Button>
      </PageHeader>

      <Box display="flex" flexDirection="column" flex={1}>
        <DataTable
          data={models}
          columns={columns}
          actions={actions}
          loading={loading}
          error={error}
          onRetry={loadModels}
          emptyState={{
            icon: '🤖',
            title: 'No Models yet',
            description: 'Create your first model to get started',
            actionLabel: 'Create Model',
            onAction: handleCreate
          }}
        />
      </Box>

      <SlideDialog
        open={showDialog}
        onClose={handleCloseDialog}
        title={editingModel ? 'Edit Model' : 'Create Model'}
        actions={
          <>
            <Button
              color="info"
              type="button"
              onClick={handleCloseDialog}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              color="primary"
              variant="contained"
              form="model-form"
            >
              {editingModel ? 'Update' : 'Create'}
            </Button>
          </>
        }
      >
        <form id="model-form" onSubmit={handleSubmit}>
          <InputGroup>
            <InputLabel>Model Name *</InputLabel>
            <Input
              type="text"
              id="name"
              value={formData.name}
              onChange={(e) => setFormData({...formData, name: e.target.value})}
              error={!!formErrors.name}
              placeholder="e.g., GPT-4 Production"
              helperText={formErrors.name}
              fullWidth
            />
          </InputGroup>

          <InputGroup>
            <InputLabel>Provider Type *</InputLabel>
            <Select
              id="provider_type"
              value={formData.provider_type}
              onChange={(e) => {
                const newProvider = e.target.value as ProviderType;
                setFormData({
                  ...formData,
                  provider_type: newProvider,
                  model_id: '' // Reset model_id when provider changes
                });
              }}
              fullWidth
              error={!!formErrors.provider_type}
              MenuProps={{
                style: { zIndex: 1500 },
                PaperProps: {
                  style: { zIndex: 1500 }
                }
              }}
            >
              {providerTypes.length === 0 ? (
                <MenuItem disabled>Loading providers...</MenuItem>
              ) : (
                providerTypes.map((provider) => (
                  <MenuItem key={provider} value={provider}>
                    {provider}
                  </MenuItem>
                ))
              )}
            </Select>
            {formErrors.provider_type && (
              <p style={{color: 'red', fontSize: '0.875rem', marginTop: '4px'}}>{formErrors.provider_type}</p>
            )}
          </InputGroup>

          <InputGroup>
            <InputLabel>Model ID *</InputLabel>
            <Select
              id="model_id"
              value={formData.model_id}
              onChange={(e) => setFormData({...formData, model_id: e.target.value})}
              disabled={!formData.provider_type}
              fullWidth
              error={!!formErrors.model_id}
              displayEmpty
              MenuProps={{
                style: { zIndex: 1500, maxHeight: 300 },
                PaperProps: {
                  style: { zIndex: 1500, maxHeight: 300 }
                }
              }}
            >
              <MenuItem value="" disabled>
                {formData.provider_type
                  ? 'Select a model'
                  : 'Select a provider first'}
              </MenuItem>
              {formData.provider_type && providerModels[formData.provider_type]?.models && providerModels[formData.provider_type].models.length > 0 ? (
                providerModels[formData.provider_type].models.map((modelName) => (
                  <MenuItem key={modelName} value={modelName}>
                    {modelName}
                  </MenuItem>
                ))
              ) : formData.provider_type ? (
                <MenuItem disabled>
                  {Object.keys(providerModels).length === 0 
                    ? 'Loading models...' 
                    : providerModels[formData.provider_type] 
                      ? `No models available for ${formData.provider_type}` 
                      : `Provider ${formData.provider_type} not found`}
                </MenuItem>
              ) : null}
            </Select>
            {formErrors.model_id && (
              <p style={{color: 'red', fontSize: '0.875rem', marginTop: '4px'}}>{formErrors.model_id}</p>
            )}
            {formData.provider_type && (
              <p style={{color: '#999', fontSize: '0.875rem', marginTop: '4px'}}>
                Select a model from the available options for {formData.provider_type}
              </p>
            )}
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
                    id="reasoning_effort"
                    value={reasoningConfig.effort || ''}
                    onChange={(e) => setReasoningConfig({
                      ...reasoningConfig,
                      effort: e.target.value as 'low' | 'medium' | 'high' | undefined || undefined
                    })}
                    fullWidth
                    displayEmpty
                    MenuProps={{
                      style: { zIndex: 1500 },
                      PaperProps: {
                        style: { zIndex: 1500 }
                      }
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
        </form>
      </SlideDialog>
    </PageContainer>
  );
};

