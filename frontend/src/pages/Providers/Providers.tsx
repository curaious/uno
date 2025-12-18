import React, {useEffect, useState} from 'react';
import {api} from '../../api';
import {
  APIKey,
  CreateAPIKeyRequest,
  UpdateAPIKeyRequest,
  ProviderType,
  ProviderConfig,
  CreateProviderConfigRequest,
  UpdateProviderConfigRequest
} from '../../components/Chat/types';
import {Action, Column, DataTable} from '../../components/DataTable/DataTable';
import {PageContainer, PageHeader, PageSubtitle, PageTitle} from "../../components/shared/Page";
import {Button} from "../../components/shared/Buttons";
import {Box, IconButton, MenuItem, Chip, Switch, Tabs, Tab, styled, Divider, Typography} from '@mui/material';
import Close from '@mui/icons-material/Close';
import Edit from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import Add from '@mui/icons-material/Add';
import {Input, InputGroup, InputLabel, Select} from '../../components/shared/Input';
import {SlideDialog} from "../../components/shared/Dialog";
import {OpenAIIcon} from '../../Icons/OpenAI';
import {AnthropicIcon} from '../../Icons/Anthropic';
import {GeminiIcon} from '../../Icons/Gemini';
import {XAIIcon} from '../../Icons/XAI';

const StyledTab = styled(Tab)(() => ({
  justifyContent: 'flex-start',
  alignItems: 'center',
  gap: '4px',
  padding: '10px 8px',
  minHeight: 'auto',
  textTransform: 'none',
}))

export const Providers: React.FC = props => {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [providerConfigs, setProviderConfigs] = useState<Record<ProviderType, ProviderConfig | null>>({
    OpenAI: null,
    Anthropic: null,
    Gemini: null,
    xAI: null,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showApiKeyDialog, setShowApiKeyDialog] = useState(false);
  const [showConfigDialog, setShowConfigDialog] = useState(false);
  const [editingApiKey, setEditingApiKey] = useState<APIKey | null>(null);
  const [editingProviderType, setEditingProviderType] = useState<ProviderType | null>(null);
  const [apiKeyFormData, setApiKeyFormData] = useState<CreateAPIKeyRequest>({
    provider_type: 'OpenAI',
    name: '',
    api_key: '',
    enabled: true,
    is_default: false,
  });
  const [configFormData, setConfigFormData] = useState<CreateProviderConfigRequest>({
    provider_type: 'OpenAI',
    base_url: '',
    custom_headers: {}
  });
  const [formErrors, setFormErrors] = useState<{ [key: string]: string }>({});
  const [headerKey, setHeaderKey] = useState('');
  const [headerValue, setHeaderValue] = useState('');
  const [selectedProvider, setSelectedProvider] = useState<ProviderType>('OpenAI');

  // Load data on component mount
  useEffect(() => {
    loadAPIKeys();
    loadProviderConfigs();
  }, []);

  // Provider types in order
  const providerTypes: ProviderType[] = ['OpenAI', 'Anthropic', 'Gemini', 'xAI'];

  // Provider icons mapping
  const providerIcons: Record<ProviderType, React.ReactElement> = {
    OpenAI: <OpenAIIcon />,
    Anthropic: <AnthropicIcon />,
    Gemini: <GeminiIcon />,
    xAI: <XAIIcon />,
  };

  const loadAPIKeys = async (providerType?: string) => {
    try {
      setLoading(true);
      setError(null);
      const params = providerType ? { provider: providerType } : {};
      const response = await api.get('/api-keys', { params });
      setApiKeys(response.data.data || []);
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to load API keys';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const loadProviderConfigs = async () => {
    try {
      const response = await api.get('/provider-configs');
      const configs = response.data.data || [];
      const configMap: Record<ProviderType, ProviderConfig | null> = {
        OpenAI: null,
        Anthropic: null,
        Gemini: null,
        xAI: null,
      };
      configs.forEach((config: ProviderConfig) => {
        configMap[config.provider_type] = config;
      });
      setProviderConfigs(configMap);
    } catch (err: any) {
      console.error('Failed to load provider configs:', err);
    }
  };

  const handleCreateApiKey = () => {
    setEditingApiKey(null);
    setApiKeyFormData({
      provider_type: selectedProvider,
      name: '',
      api_key: '',
      enabled: true,
      is_default: false,
    });
    setFormErrors({});
    setShowApiKeyDialog(true);
  };

  const handleEditApiKey = (apiKey: APIKey) => {
    setEditingApiKey(apiKey);
    setApiKeyFormData({
      provider_type: apiKey.provider_type,
      name: apiKey.name,
      api_key: apiKey.api_key,
      enabled: apiKey.enabled,
      is_default: apiKey.is_default,
    });
    setFormErrors({});
    setShowApiKeyDialog(true);
  };

  const handleEditConfig = (providerType: ProviderType) => {
    setEditingProviderType(providerType);
    const config = providerConfigs[providerType];
    setConfigFormData({
      provider_type: providerType,
      base_url: config?.base_url || '',
      custom_headers: config?.custom_headers || {}
    });
    setHeaderKey('');
    setHeaderValue('');
    setShowConfigDialog(true);
  };

  const handleDeleteApiKey = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this API key?')) {
      return;
    }

    try {
      await api.delete(`/api-keys/${id}`);
      await loadAPIKeys();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to delete API key';
      setError(errorMessage);
    }
  };

  const validateApiKeyForm = (): boolean => {
    const errors: { [key: string]: string } = {};

    if (!apiKeyFormData.name.trim()) {
      errors.name = 'Name is required';
    }

    if (!apiKeyFormData.api_key.trim()) {
      errors.api_key = 'API key is required';
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleApiKeySubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateApiKeyForm()) {
      return;
    }

    try {
      if (editingApiKey) {
        const updateData: UpdateAPIKeyRequest = {
          name: apiKeyFormData.name,
          api_key: apiKeyFormData.api_key,
          enabled: apiKeyFormData.enabled,
          is_default: apiKeyFormData.is_default,
        };
        await api.put(`/api-keys/${editingApiKey.id}`, updateData);
      } else {
        await api.post('/api-keys', apiKeyFormData);
      }

      setShowApiKeyDialog(false);
      await loadAPIKeys();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to save API key';
      setError(errorMessage);
    }
  };

  const handleConfigSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!editingProviderType) return;

    try {
      const createData: CreateProviderConfigRequest = {
        provider_type: editingProviderType,
        base_url: configFormData.base_url?.trim() || undefined,
        custom_headers: configFormData.custom_headers && Object.keys(configFormData.custom_headers).length > 0 ? configFormData.custom_headers : undefined
      };
      await api.put(`/provider-configs/${editingProviderType}`, createData);

      setShowConfigDialog(false);
      await loadProviderConfigs();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to save provider config';
      setError(errorMessage);
    }
  };

  const handleCloseApiKeyDialog = () => {
    setShowApiKeyDialog(false);
    setEditingApiKey(null);
    setApiKeyFormData({
      provider_type: 'OpenAI',
      name: '',
      api_key: '',
      enabled: true,
      is_default: false,
    });
    setFormErrors({});
  };

  const handleCloseConfigDialog = () => {
    setShowConfigDialog(false);
    setEditingProviderType(null);
    setConfigFormData({
      provider_type: 'OpenAI',
      base_url: '',
      custom_headers: {}
    });
    setHeaderKey('');
    setHeaderValue('');
  };

  const addHeader = () => {
    if (headerKey.trim() && headerValue.trim()) {
      setConfigFormData({
        ...configFormData,
        custom_headers: {
          ...configFormData.custom_headers,
          [headerKey.trim()]: headerValue.trim()
        }
      });
      setHeaderKey('');
      setHeaderValue('');
    }
  };

  const removeHeader = (key: string) => {
    const newHeaders = {...configFormData.custom_headers};
    delete newHeaders[key];
    setConfigFormData({
      ...configFormData,
      custom_headers: newHeaders
    });
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

  const maskApiKey = (apiKey: string) => {
    if (apiKey.length <= 8) return '••••••••';
    return apiKey.substring(0, 4) + '••••••••' + apiKey.substring(apiKey.length - 4);
  };

  const toggleEnabled = async (apiKey: APIKey) => {
    try {
      const updateData: UpdateAPIKeyRequest = {
        enabled: !apiKey.enabled
      };
      await api.put(`/api-keys/${apiKey.id}`, updateData);
      await loadAPIKeys();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to update API key';
      setError(errorMessage);
    }
  };

  // Filter API keys by selected provider
  const filteredAPIKeys = apiKeys.filter(key => key.provider_type === selectedProvider);

  // Filter provider configs by selected provider
  const filteredConfigs = [{ id: selectedProvider, provider_type: selectedProvider, config: providerConfigs[selectedProvider] }];

  // Provider Config columns
  const configColumns: Column<{ id: string; provider_type: ProviderType; config: ProviderConfig | null }>[] = [
    {
      key: 'provider_type',
      label: 'Provider',
      render: (value, item) => (
        <Box display="flex" alignItems="center" gap={1}>
          {providerIcons[item.provider_type]}
          <span>{item.provider_type}</span>
        </Box>
      )
    },
    {
      key: 'config',
      label: 'Base URL',
      render: (value, item) => (
        <div>
          {item.config?.base_url ? (
            <code style={{ fontSize: '0.875rem' }}>{item.config.base_url}</code>
          ) : (
            <span style={{ color: '#999' }}>Not set</span>
          )}
        </div>
      )
    },
    {
      key: 'config',
      label: 'Custom Headers',
      render: (value, item) => (
        <div>
          {item.config?.custom_headers && Object.keys(item.config.custom_headers).length > 0 ? (
            <span>{Object.keys(item.config.custom_headers).length} header{Object.keys(item.config.custom_headers).length !== 1 ? 's' : ''}</span>
          ) : (
            <span style={{ color: '#999' }}>None</span>
          )}
        </div>
      )
    },
    {
      key: 'provider_type',
      label: 'Actions',
      render: (value, item) => (
        <Button
          type="button"
          size="small"
          onClick={() => handleEditConfig(item.provider_type)}
        >
          <Edit fontSize="small" style={{ marginRight: '4px' }} />
          Edit
        </Button>
      )
    }
  ];

  // API Key columns
  const apiKeyColumns: Column<APIKey>[] = [
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
      key: 'api_key',
      label: 'API Key',
      render: (value, item) => (
        <div>
          <code>{maskApiKey(item.api_key)}</code>
        </div>
      )
    },
    {
      key: 'enabled',
      label: 'Enabled',
      render: (value, item) => (
        <Switch
          checked={item.enabled}
          onChange={() => toggleEnabled(item)}
          size="small"
        />
      )
    },
    {
      key: 'is_default',
      label: 'Default',
      render: (value, item) => (
        <div>
          {item.is_default && (
            <Chip
              label="Default"
              size="small"
              color="primary"
            />
          )}
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

  const apiKeyActions: Action<APIKey>[] = [
    {
      label: 'Edit',
      icon: (
        <Edit />
      ),
      onClick: handleEditApiKey,
      title: 'Edit'
    },
    {
      label: 'Delete',
      icon: (
        <DeleteIcon />
      ),
      onClick: (item) => handleDeleteApiKey(item.id),
      className: 'deleteButton',
      title: 'Delete'
    }
  ];

  return (
    <PageContainer>
      <PageHeader>
        <div>
          <PageTitle>Providers</PageTitle>
          <PageSubtitle>Manage provider configurations and API keys</PageSubtitle>
        </div>
        <Button color="primary" variant="contained" onClick={handleCreateApiKey}>
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path
              d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
          </svg>
          Add API Key
        </Button>
      </PageHeader>

      <Box display="flex" flex={1} gap={2} sx={{ mt: 2 }}>
        {/* Vertical Tabs */}
        <Box
          sx={{
            borderRight: 1,
            borderColor: 'divider',
            minWidth: 200,
            maxWidth: 200,
          }}
        >
          <Tabs
            orientation="vertical"
            value={selectedProvider}
            onChange={(e, newValue) => setSelectedProvider(newValue)}
            sx={{
              borderRight: 1,
              borderColor: 'divider',
              '& .MuiTabs-indicator': {
                left: 0,
                right: 'auto',
              },
            }}
          >
            {providerTypes.map((provider) => (
              <StyledTab
                key={provider}
                label={provider}
                value={provider}
                icon={providerIcons[provider]}
                iconPosition="start"
              />
            ))}
          </Tabs>
        </Box>

        {/* Content Area */}
        <Box flex={1} display="flex" flexDirection="column" gap={3}>
          {/* Provider Configs Section */}
          <Box>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
              <Typography variant="h6">Provider Configurations</Typography>
            </Box>
            <DataTable
              data={filteredConfigs}
              columns={configColumns}
              loading={loading}
              error={error}
              onRetry={() => loadProviderConfigs()}
              emptyState={{
                icon: '⚙️',
                title: 'No Provider Configurations',
                description: 'Provider configurations will appear here',
                actionLabel: '',
                onAction: () => {}
              }}
            />
          </Box>

          <Divider />

          {/* API Keys Section */}
          <Box>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
              <Typography variant="h6">API Keys</Typography>
            </Box>
            <DataTable
              data={filteredAPIKeys}
              columns={apiKeyColumns}
              actions={apiKeyActions}
              loading={loading}
              error={error}
              onRetry={() => loadAPIKeys()}
              emptyState={{
                icon: '🔑',
                title: 'No API Keys yet',
                description: `No API keys for ${selectedProvider} yet`,
                actionLabel: 'Add API Key',
                onAction: handleCreateApiKey
              }}
            />
          </Box>
        </Box>
      </Box>

      {/* API Key Dialog */}
      <SlideDialog
        open={showApiKeyDialog}
        onClose={handleCloseApiKeyDialog}
        title={editingApiKey ? 'Edit API Key' : 'Add API Key'}
        actions={
          <>
            <Button
              color="info"
              type="button"
              onClick={handleCloseApiKeyDialog}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              color="primary"
              variant="contained"
              form="api-key-form"
            >
              {editingApiKey ? 'Update' : 'Create'}
            </Button>
          </>
        }
      >
        <form id="api-key-form" onSubmit={handleApiKeySubmit}>
          <InputGroup>
            <InputLabel>Provider Type *</InputLabel>
            <Select
              id="provider_type"
              value={apiKeyFormData.provider_type}
              onChange={(e) => setApiKeyFormData({...apiKeyFormData, provider_type: e.target.value as ProviderType})}
              disabled={!!editingApiKey}
              fullWidth
              MenuProps={{
                style: { zIndex: 1500 },
                PaperProps: {
                  style: { zIndex: 1500 }
                }
              }}
            >
              <MenuItem value="OpenAI">OpenAI</MenuItem>
              <MenuItem value="Anthropic">Anthropic</MenuItem>
              <MenuItem value="Gemini">Gemini</MenuItem>
              <MenuItem value="xAI">xAI</MenuItem>
            </Select>
            {editingApiKey && (
              <p>Provider type cannot be changed after creation</p>
            )}
          </InputGroup>

          <InputGroup>
            <InputLabel>Name *</InputLabel>
            <Input
              type="text"
              id="name"
              value={apiKeyFormData.name}
              onChange={(e) => setApiKeyFormData({...apiKeyFormData, name: e.target.value})}
              error={!!formErrors.name}
              placeholder="e.g., Production Key, Development Key"
              helperText={formErrors.name}
              fullWidth
            />
          </InputGroup>

          <InputGroup>
            <InputLabel>API Key *</InputLabel>
            <Input
              type="password"
              id="api_key"
              value={apiKeyFormData.api_key}
              onChange={(e) => setApiKeyFormData({...apiKeyFormData, api_key: e.target.value})}
              error={!!formErrors.api_key}
              placeholder="Enter your API key"
              helperText={formErrors.api_key}
              fullWidth
            />
          </InputGroup>

          <InputGroup>
            <Box display="flex" alignItems="center" gap={2}>
              <Box display="flex" alignItems="center" gap={1}>
                <Switch
                  checked={apiKeyFormData.enabled}
                  onChange={(e) => setApiKeyFormData({...apiKeyFormData, enabled: e.target.checked})}
                />
                <InputLabel style={{ margin: 0, cursor: 'pointer' }}>
                  Enabled
                </InputLabel>
              </Box>
              <Box display="flex" alignItems="center" gap={1}>
                <input
                  type="checkbox"
                  id="is_default"
                  checked={apiKeyFormData.is_default}
                  onChange={(e) => setApiKeyFormData({...apiKeyFormData, is_default: e.target.checked})}
                />
                <InputLabel htmlFor="is_default" style={{ margin: 0, cursor: 'pointer' }}>
                  Set as default
                </InputLabel>
              </Box>
            </Box>
          </InputGroup>
        </form>
      </SlideDialog>

      {/* Provider Config Dialog */}
      <SlideDialog
        open={showConfigDialog}
        onClose={handleCloseConfigDialog}
        title={`Edit ${editingProviderType} Configuration`}
        actions={
          <>
            <Button
              color="info"
              type="button"
              onClick={handleCloseConfigDialog}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              color="primary"
              variant="contained"
              form="config-form"
            >
              Save
            </Button>
          </>
        }
      >
        <form id="config-form" onSubmit={handleConfigSubmit}>
          <InputGroup>
            <InputLabel>Base URL (Optional)</InputLabel>
            <Input
              type="text"
              id="base_url"
              value={configFormData.base_url}
              onChange={(e) => setConfigFormData({...configFormData, base_url: e.target.value})}
              placeholder="e.g., https://api.openai.com/v1"
              helperText="Custom base URL for the provider API"
              fullWidth
            />
          </InputGroup>

          <InputGroup>
            <InputLabel>Custom Headers (Optional)</InputLabel>
            <div>
              <Box display="flex" alignItems="flex-start" gap="8px" mb={1}>
                <Input
                  type="text"
                  value={headerKey}
                  onChange={(e) => setHeaderKey(e.target.value)}
                  placeholder="Header name"
                  fullWidth
                />
                <Input
                  type="text"
                  value={headerValue}
                  onChange={(e) => setHeaderValue(e.target.value)}
                  placeholder="Header value"
                  fullWidth
                />
                <Button
                  type="button"
                  onClick={addHeader}
                  size="small"
                  variant="contained"
                >
                  Add
                </Button>
              </Box>

              {configFormData.custom_headers && Object.keys(configFormData.custom_headers).length > 0 && (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  {Object.entries(configFormData.custom_headers).map(([key, value]) => (
                    <Box
                      key={key}
                      display="flex"
                      alignItems="center"
                      gap={1}
                      p={1.5}
                      sx={{
                        border: '1px solid rgba(255, 255, 255, 0.23)',
                        borderRadius: '4px',
                        backgroundColor: 'var(--background-elevated)',
                      }}
                    >
                      <Box
                        component="span"
                        sx={{
                          fontWeight: 600,
                          minWidth: '120px',
                          fontSize: '0.875rem',
                          color: 'var(--text-default)',
                        }}
                      >
                        {key}:
                      </Box>
                      <Box
                        component="span"
                        sx={{
                          flex: 1,
                          fontSize: '0.875rem',
                          color: 'var(--text-secondary)',
                          wordBreak: 'break-word',
                        }}
                      >
                        {value}
                      </Box>
                      <IconButton
                        size="small"
                        onClick={() => removeHeader(key)}
                        sx={{
                          color: 'error.main',
                          '&:hover': {
                            backgroundColor: 'rgba(211, 47, 47, 0.1)',
                          },
                        }}
                      >
                        <Close fontSize="small" />
                      </IconButton>
                    </Box>
                  ))}
                </Box>
              )}
            </div>
          </InputGroup>
        </form>
      </SlideDialog>
    </PageContainer>
  );
};
