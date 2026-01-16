import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router';
import { api } from '../../api';
import { AgentConfigSummary } from './types';
import { Action, Column, DataTable } from '../../components/DataTable/DataTable';
import { PageContainer, PageHeader, PageSubtitle, PageTitle } from '../../components/shared/Page';
import { Button } from '../../components/shared/Buttons';
import { Box, Chip } from '@mui/material';
import Edit from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import SmartToyIcon from '@mui/icons-material/SmartToy';

export const AgentBuilder: React.FC = () => {
  const navigate = useNavigate();
  const [configs, setConfigs] = useState<AgentConfigSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadConfigs();
  }, []);

  const loadConfigs = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await api.get('/agent-configs');
      setConfigs(response.data.data || []);
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to load agent configs';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    // Navigate to detail page with "new" flag
    navigate('/agent-framework/agent-builder/new');
  };

  const handleEdit = (config: AgentConfigSummary) => {
    navigate(`/agent-framework/agent-builder/${config.id}`);
  };

  const handleDelete = async (name: string) => {
    if (!window.confirm(`Are you sure you want to delete agent config "${name}" and all its versions?`)) {
      return;
    }

    try {
      await api.delete(`/agent-configs/by-name?name=${encodeURIComponent(name)}`);
      await loadConfigs();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to delete agent config';
      setError(errorMessage);
    }
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

  const columns: Column<AgentConfigSummary>[] = [
    {
      key: 'name',
      label: 'Name',
      render: (value, item) => (
        <Box display="flex" alignItems="center" gap={1}>
          <span style={{ fontWeight: 500 }}>{item.name}</span>
        </Box>
      )
    },
    {
      key: 'latest_version',
      label: 'Version',
      render: (value, item) => (
        <Chip 
          label={`v${item.latest_version}`} 
          size="small" 
          variant="outlined"
          sx={{ borderColor: 'var(--border-color)' }}
        />
      )
    },
    {
      key: 'updated_at',
      label: 'Last Updated',
      render: (value, item) => (
        <span style={{ color: 'var(--text-secondary)' }}>
          {formatDate(item.updated_at)}
        </span>
      )
    },
    {
      key: 'created_at',
      label: 'Created',
      render: (value, item) => (
        <span style={{ color: 'var(--text-secondary)' }}>
          {formatDate(item.created_at)}
        </span>
      )
    }
  ];

  const actions: Action<AgentConfigSummary>[] = [
    {
      icon: <Edit />,
      label: 'Edit',
      onClick: handleEdit,
      title: 'Edit agent config'
    },
    {
      icon: <DeleteIcon />,
      label: 'Delete',
      onClick: (item) => handleDelete(item.name),
      title: 'Delete agent config'
    }
  ];

  return (
    <PageContainer>
      <PageHeader>
        <div>
          <PageTitle>Agent Builder</PageTitle>
          <PageSubtitle>Build and configure AI agents with models, prompts, schemas, and MCP servers</PageSubtitle>
        </div>
        <Button variant="contained" color="primary" onClick={handleCreate}>
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
          </svg>
          Create Agent
        </Button>
      </PageHeader>

      <Box display="flex" flexDirection="column" flex={1}>
        <DataTable
          data={configs}
          columns={columns}
          actions={actions}
          loading={loading}
          error={error}
          onRetry={loadConfigs}
          emptyState={{
            icon: null,
            title: 'No Agents yet',
            description: 'Create your first agent configuration to get started',
            actionLabel: 'Create Agent',
            onAction: handleCreate
          }}
        />
      </Box>
    </PageContainer>
  );
};

