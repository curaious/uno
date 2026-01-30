import React, {useEffect, useState} from 'react';
import {useNavigate} from 'react-router';
import {api} from '../../api';
import {CreateMCPServerRequest, MCPServer, UpdateMCPServerRequest} from '../../components/Chat/types';
import {Action, Column, DataTable} from '../../components/DataTable/DataTable';
import {PageContainer, PageHeader, PageSubtitle, PageTitle} from "../../components/shared/Page";
import {Button} from '../../components/shared/Buttons';
import {Input, InputGroup, InputLabel} from '../../components/shared/Input';
import {Box, IconButton} from "@mui/material";
import Close from "@mui/icons-material/Close";
import Edit from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import Visibility from '@mui/icons-material/Visibility';
import {SlideDialog} from "../../components/shared/Dialog";

export const MCPServers: React.FC = props => {
  const navigate = useNavigate();
  const [servers, setServers] = useState<MCPServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showSidebar, setShowSidebar] = useState(false);
  const [editingServer, setEditingServer] = useState<MCPServer | null>(null);
  const [formData, setFormData] = useState<CreateMCPServerRequest>({
    name: '',
    endpoint: '',
    headers: {}
  });
  const [formErrors, setFormErrors] = useState<{ [key: string]: string }>({});
  const [headerKey, setHeaderKey] = useState('');
  const [headerValue, setHeaderValue] = useState('');

  // Load servers on component mount
  useEffect(() => {
    loadServers();
  }, []);

  const loadServers = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await api.get('/mcp-servers');
      setServers(response.data.data || []);
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to load MCP servers';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditingServer(null);
    setFormData({name: '', endpoint: '', headers: {}});
    setFormErrors({});
    setHeaderKey('');
    setHeaderValue('');
    setShowSidebar(true);
  };

  const handleEdit = (server: MCPServer) => {
    setEditingServer(server);
    setFormData({
      name: server.name,
      endpoint: server.endpoint,
      headers: {...server.headers}
    });
    setFormErrors({});
    setHeaderKey('');
    setHeaderValue('');
    setShowSidebar(true);
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this MCP server?')) {
      return;
    }

    try {
      await api.delete(`/mcp-servers/${id}`);
      await loadServers();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to delete MCP server';
      setError(errorMessage);
    }
  };

  const handleInspect = (server: MCPServer) => {
    navigate(`/mcp-servers/${server.id}/inspect`);
  };

  const validateForm = (): boolean => {
    const errors: { [key: string]: string } = {};

    if (!formData.name.trim()) {
      errors.name = 'Name is required';
    }

    if (!formData.endpoint.trim()) {
      errors.endpoint = 'Endpoint is required';
    } else {
      try {
        new URL(formData.endpoint);
      } catch {
        errors.endpoint = 'Please enter a valid URL';
      }
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
      if (editingServer) {
        // Update existing server
        const updateData: UpdateMCPServerRequest = {
          name: formData.name,
          endpoint: formData.endpoint,
          headers: formData.headers
        };
        await api.put(`/mcp-servers/${editingServer.id}`, updateData);
      } else {
        // Create new server
        await api.post('/mcp-servers', formData);
      }

      setShowSidebar(false);
      await loadServers();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to save MCP server';
      setError(errorMessage);
    }
  };

  const handleCloseSidebar = () => {
    setShowSidebar(false);
    setEditingServer(null);
    setFormData({name: '', endpoint: '', headers: {}});
    setFormErrors({});
    setHeaderKey('');
    setHeaderValue('');
  };

  const addHeader = () => {
    if (headerKey.trim() && headerValue.trim()) {
      setFormData({
        ...formData,
        headers: {
          ...formData.headers,
          [headerKey.trim()]: headerValue.trim()
        }
      });
      setHeaderKey('');
      setHeaderValue('');
    }
  };

  const removeHeader = (key: string) => {
    const newHeaders = {...formData.headers};
    delete newHeaders[key];
    setFormData({
      ...formData,
      headers: newHeaders
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

  const formatHeaders = (headers: { [key: string]: string }) => {
    const entries = Object.entries(headers);
    if (entries.length === 0) return 'No headers';
    if (entries.length === 1) return `${entries[0][0]}: ${entries[0][1]}`;
    return `${entries.length} headers`;
  };

  // Table configuration
  const columns: Column<MCPServer>[] = [
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
      key: 'endpoint',
      label: 'Endpoint',
      render: (value, item) => (
        <div>
          <p>{item.endpoint}</p>
        </div>
      )
    },
    {
      key: 'headers',
      label: 'Headers',
      render: (value, item) => (
        <div>
          <span>{formatHeaders(item.headers)}</span>
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

  const actions: Action<MCPServer>[] = [
    {
      label: 'Inspect',
      icon: (
        <Visibility/>
      ),
      onClick: handleInspect,
      title: 'Inspect MCP Server'
    },
    {
      label: 'Edit',
      icon: (
        <Edit/>
      ),
      onClick: handleEdit,
      title: 'Edit'
    },
    {
      label: 'Delete',
      icon: (
        <DeleteIcon/>
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
          <PageTitle>MCP Servers</PageTitle>
          <PageSubtitle>Manage your Model Context Protocol servers</PageSubtitle>
        </div>
        <Button variant="contained" color="primary" onClick={handleCreate}>
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path
              d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
          </svg>
          Create Server
        </Button>
      </PageHeader>

      <Box display="flex" flexDirection="column" flex={1}>
        <DataTable
          data={servers}
          columns={columns}
          actions={actions}
          loading={loading}
          error={error}
          onRetry={loadServers}
          emptyState={{
            icon: '🔗',
            title: 'No MCP servers yet',
            description: 'Create your first MCP server to get started',
            actionLabel: 'Create Server',
            onAction: handleCreate
          }}
        />
      </Box>

      <SlideDialog
        open={showSidebar}
        onClose={handleCloseSidebar}
        title={editingServer ? 'Edit MCP Server' : 'Create MCP Server'}
        actions={
          <>
            <Button
              color="info"
              type="button"
              onClick={handleCloseSidebar}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              color="primary"
              variant="contained"
              form="mcp-server-form"
            >
              {editingServer ? 'Update' : 'Create'}
            </Button>
          </>
        }
      >
        <form id="mcp-server-form" onSubmit={handleSubmit}>
          <InputGroup>
            <InputLabel>Server Name *</InputLabel>
            <Input
              type="text"
              id="name"
              value={formData.name}
              onChange={(e) => setFormData({...formData, name: e.target.value})}
              placeholder="e.g., Production Planner MCP"
              helperText={formErrors.name}
              error={!!formErrors.name}
              fullWidth
            />
          </InputGroup>

          <InputGroup>
            <InputLabel>Endpoint URL *</InputLabel>
            <Input
              type="url"
              id="endpoint"
              value={formData.endpoint}
              onChange={(e) => setFormData({...formData, endpoint: e.target.value})}
              placeholder="http://localhost:9001/mcp/sse"
              helperText={formErrors.endpoint}
              error={!!formErrors.endpoint}
              fullWidth
            />
          </InputGroup>

          <InputGroup>
            <InputLabel>Headers (Optional)</InputLabel>
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

              {Object.keys(formData.headers).length > 0 && (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  {Object.entries(formData.headers).map(([key, value]) => (
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