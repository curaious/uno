import React, {useEffect, useState} from 'react';
import {api} from '../../api';
import {VirtualKey, CreateVirtualKeyRequest, UpdateVirtualKeyRequest, ProviderType, ProviderModelsData} from '../../components/Chat/types';
import {Action, Column, DataTable} from '../../components/DataTable/DataTable';
import {PageContainer, PageHeader, PageSubtitle, PageTitle} from "../../components/shared/Page";
import {Button} from "../../components/shared/Buttons";
import {Box, IconButton, Chip, MenuItem} from '@mui/material';
import Edit from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import {Input, InputGroup, InputLabel, Select} from '../../components/shared/Input';
import {SlideDialog} from "../../components/shared/Dialog";

export const VirtualKeys: React.FC = props => {
  const [virtualKeys, setVirtualKeys] = useState<VirtualKey[]>([]);
  const [providerModels, setProviderModels] = useState<{ [key: string]: ProviderModelsData }>({});
  const [providerTypes, setProviderTypes] = useState<ProviderType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showDialog, setShowDialog] = useState(false);
  const [editingVirtualKey, setEditingVirtualKey] = useState<VirtualKey | null>(null);
  const [formData, setFormData] = useState<CreateVirtualKeyRequest>({
    name: '',
    providers: [],
    model_ids: []
  });
  const [formErrors, setFormErrors] = useState<{ [key: string]: string }>({});

  // Load virtual keys and provider models on component mount
  useEffect(() => {
    loadVirtualKeys();
    loadProviderModels();
  }, []);

  const loadVirtualKeys = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await api.get('/virtual-keys');
      setVirtualKeys(response.data.data || []);
      return response.data.data || [];
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to load virtual keys';
      setError(errorMessage);
      return [];
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
        // Extract provider types from the response keys
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
    setEditingVirtualKey(null);
    setFormData({
      name: '',
      providers: [],
      model_ids: []
    });
    setFormErrors({});
    setShowDialog(true);
  };

  const handleEdit = (virtualKey: VirtualKey) => {
    setEditingVirtualKey(virtualKey);
    setFormData({
      name: virtualKey.name,
      providers: virtualKey.providers,
      model_ids: virtualKey.model_ids
    });
    setFormErrors({});
    setShowDialog(true);
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this virtual key?')) {
      return;
    }

    try {
      await api.delete(`/virtual-keys/${id}`);
      await loadVirtualKeys();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to delete virtual key';
      setError(errorMessage);
    }
  };

  const validateForm = (): boolean => {
    const errors: { [key: string]: string } = {};

    if (!formData.name.trim()) {
      errors.name = 'Name is required';
    }

    if (formData.providers.length === 0) {
      errors.providers = 'At least one provider is required';
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
      if (editingVirtualKey) {
        // Update existing virtual key
        const updateData: UpdateVirtualKeyRequest = {
          name: formData.name,
          providers: formData.providers,
          model_ids: formData.model_ids && formData.model_ids.length > 0 ? formData.model_ids : undefined
        };
        await api.put(`/virtual-keys/${editingVirtualKey.id}`, updateData);
      } else {
        // Create new virtual key
        const createData: CreateVirtualKeyRequest = {
          name: formData.name,
          providers: formData.providers,
          model_ids: formData.model_ids && formData.model_ids.length > 0 ? formData.model_ids : undefined
        };
        const response = await api.post('/virtual-keys', createData);
        const createdKey: VirtualKey = response.data.data;
        
        setShowDialog(false);
        await loadVirtualKeys();
        
        // Show the newly created key's secret key
        alert(`Virtual key created!\n\nSecret key: ${createdKey.secret_key}\n\nMake sure to save this securely - you won't be able to view it again.`);
        return;
      }

      setShowDialog(false);
      await loadVirtualKeys();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to save virtual key';
      setError(errorMessage);
    }
  };

  const handleCloseDialog = () => {
    setShowDialog(false);
    setEditingVirtualKey(null);
    setFormData({
      name: '',
      providers: [],
      model_ids: []
    });
    setFormErrors({});
  };

  const handleProviderChange = (event: any) => {
    const value = event.target.value;
    setFormData({
      ...formData,
      providers: typeof value === 'string' ? value.split(',') : value
    });
  };

  const handleModelChange = (event: any) => {
    const value = event.target.value;
    setFormData({
      ...formData,
      model_ids: typeof value === 'string' ? value.split(',') : value
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

  const maskSecretKey = (secretKey: string) => {
    if (secretKey.length <= 12) return '••••••••';
    // Show first 8 chars (sk-amg-) and last 4 chars
    return secretKey.substring(0, 8) + '••••••••' + secretKey.substring(secretKey.length - 4);
  };

  // Get available models from provider models data based on selected providers
  const getAvailableModels = (): Array<{ id: string; name: string; provider_type: ProviderType }> => {
    if (formData.providers.length === 0) {
      return [];
    }
    
    const availableModels: Array<{ id: string; name: string; provider_type: ProviderType }> = [];
    formData.providers.forEach(provider => {
      const providerData = providerModels[provider];
      if (providerData && providerData.models && Array.isArray(providerData.models)) {
        providerData.models.forEach((modelName: string) => {
          // Use model name as ID since backend expects model names, not UUIDs
          availableModels.push({
            id: modelName,
            name: modelName,
            provider_type: provider
          });
        });
      }
    });
    
    return availableModels;
  };

  const availableModels = getAvailableModels();

  // Table configuration
  const columns: Column<VirtualKey>[] = [
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
      key: 'secret_key',
      label: 'Secret Key',
      render: (value, item) => (
        <Box display="flex" alignItems="center" gap={1}>
          <code style={{ fontSize: '0.875rem', wordBreak: 'break-all' }}>{item.secret_key}</code>
          <Button
            type="button"
            size="small"
            onClick={() => {
              navigator.clipboard.writeText(item.secret_key);
              alert('Secret key copied to clipboard!');
            }}
          >
            Copy
          </Button>
        </Box>
      )
    },
    {
      key: 'providers',
      label: 'Providers',
      render: (value, item) => (
        <Box display="flex" gap={0.5} flexWrap="wrap">
          {item.providers.map((provider) => (
            <Chip
              key={provider}
              label={provider}
              size="small"
              color="primary"
              variant="outlined"
            />
          ))}
        </Box>
      )
    },
    {
      key: 'model_ids',
      label: 'Models',
      render: (value, item) => (
        <div>
          {item.model_ids && item.model_ids.length > 0 ? (
            <span>{item.model_ids.length} model{item.model_ids.length !== 1 ? 's' : ''}</span>
          ) : (
            <span style={{ color: '#999' }}>All models</span>
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

  const actions: Action<VirtualKey>[] = [
    {
      label: 'Edit',
      icon: (
        <Edit />
      ),
      onClick: handleEdit,
      title: 'Edit'
    },
    {
      label: 'Delete',
      icon: (
        <DeleteIcon />
      ),
      onClick: (item) => handleDelete(item.id),
      className: 'deleteButton',
      title: 'Delete'
    }
  ];

  return (
    <PageContainer>
      <PageHeader>
        <div>
          <PageTitle>Virtual Keys</PageTitle>
          <PageSubtitle>Manage virtual keys for provider and model access control</PageSubtitle>
        </div>
        <Button color="primary" variant="contained" onClick={handleCreate}>
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path
              d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
          </svg>
          Add Virtual Key
        </Button>
      </PageHeader>

      <Box display="flex" flexDirection="column" flex={1} mb={2}>
        <DataTable
          data={virtualKeys}
          columns={columns}
          actions={actions}
          loading={loading}
          error={error}
          onRetry={() => loadVirtualKeys()}
          emptyState={{
            icon: '🔑',
            title: 'No Virtual Keys yet',
            description: 'Add your first virtual key to get started',
            actionLabel: 'Add Virtual Key',
            onAction: handleCreate
          }}
        />
      </Box>

      <SlideDialog
        open={showDialog}
        onClose={handleCloseDialog}
        title={editingVirtualKey ? 'Edit Virtual Key' : 'Add Virtual Key'}
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
              form="virtual-key-form"
            >
              {editingVirtualKey ? 'Update' : 'Create'}
            </Button>
          </>
        }
      >
        <form id="virtual-key-form" onSubmit={handleSubmit}>
          {editingVirtualKey && (
            <InputGroup>
              <InputLabel>Secret Key</InputLabel>
              <Box display="flex" alignItems="center" gap={1}>
                <code style={{ flex: 1, padding: '8px', background: '#1a1a1a', borderRadius: '4px', fontSize: '0.875rem', wordBreak: 'break-all' }}>
                  {editingVirtualKey.secret_key}
                </code>
                <Button
                  type="button"
                  size="small"
                  onClick={() => {
                    navigator.clipboard.writeText(editingVirtualKey.secret_key);
                    alert('Secret key copied to clipboard!');
                  }}
                >
                  Copy
                </Button>
              </Box>
            </InputGroup>
          )}
          <InputGroup>
            <InputLabel>Name *</InputLabel>
            <Input
              type="text"
              id="name"
              value={formData.name}
              onChange={(e) => setFormData({...formData, name: e.target.value})}
              error={!!formErrors.name}
              placeholder="e.g., Production Access, Development Access"
              helperText={formErrors.name}
              fullWidth
            />
          </InputGroup>

          <InputGroup>
            <InputLabel>Providers *</InputLabel>
            <Select
              multiple
              value={formData.providers}
              onChange={handleProviderChange}
              fullWidth
              renderValue={(selected) => (
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {(selected as string[]).map((value) => (
                    <Chip key={value} label={value} size="small" />
                  ))}
                </Box>
              )}
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
            {formErrors.providers && (
              <p style={{color: 'red', fontSize: '0.875rem', marginTop: '4px'}}>{formErrors.providers}</p>
            )}
          </InputGroup>

          <InputGroup>
            <InputLabel>Models (Optional)</InputLabel>
            <Select
              multiple
              value={formData.model_ids || []}
              onChange={handleModelChange}
              fullWidth
              disabled={formData.providers.length === 0}
              renderValue={(selected) => {
                if ((selected as string[]).length === 0) {
                  return <span style={{ color: '#999' }}>All models</span>;
                }
                return (
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                    {(selected as string[]).map((modelId) => {
                      const model = availableModels.find(m => m.id === modelId);
                      return (
                        <Chip 
                          key={modelId} 
                          label={model ? `${model.name} (${model.provider_type})` : modelId} 
                          size="small" 
                        />
                      );
                    })}
                  </Box>
                );
              }}
              MenuProps={{
                style: { zIndex: 1500, maxHeight: 300 },
                PaperProps: {
                  style: { zIndex: 1500, maxHeight: 300 }
                }
              }}
            >
              {availableModels.length === 0 ? (
                <MenuItem disabled>
                  {formData.providers.length === 0
                    ? 'Select providers first to see available models'
                    : 'No models available for selected providers'}
                </MenuItem>
              ) : (
                availableModels.map((model) => (
                  <MenuItem key={model.id} value={model.id}>
                    {model.name} ({model.provider_type})
                  </MenuItem>
                ))
              )}
            </Select>
            <p style={{color: '#999', fontSize: '0.875rem', marginTop: '4px'}}>
              {formData.model_ids && formData.model_ids.length > 0
                ? `${formData.model_ids.length} model${formData.model_ids.length !== 1 ? 's' : ''} selected`
                : 'Leave empty to allow all models'}
            </p>
          </InputGroup>
        </form>
      </SlideDialog>
    </PageContainer>
  );
};

