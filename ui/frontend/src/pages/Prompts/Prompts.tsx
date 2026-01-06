import React, {useEffect, useState} from 'react';
import {useNavigate} from 'react-router';
import {api} from '../../api';
import {CreatePromptRequest, PromptWithLatestVersion} from '../../components/Chat/types';
import {Action, Column, DataTable} from '../../components/DataTable/DataTable';
import {PageContainer, PageHeader, PageSubtitle, PageTitle} from "../../components/shared/Page";
import {Button} from '../../components/shared/Buttons';
import {Box, Chip, IconButton, MenuItem, Typography} from "@mui/material";
import Close from "@mui/icons-material/Close";
import DeleteIcon from '@mui/icons-material/Delete';
import Visibility from '@mui/icons-material/Visibility';
import {Input, InputGroup, InputLabel, Select} from '../../components/shared/Input';
import {SlideDialog} from "../../components/shared/Dialog";

export const Prompts: React.FC = props => {
  const navigate = useNavigate();
  const [prompts, setPrompts] = useState<PromptWithLatestVersion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showSidebar, setShowSidebar] = useState(false);
  const [formData, setFormData] = useState<CreatePromptRequest>({
    name: '',
    template: '',
    commit_message: '',
    label: undefined
  });
  const [formErrors, setFormErrors] = useState<{ [key: string]: string }>({});

  // Load prompts on component mount
  useEffect(() => {
    loadPrompts();
  }, []);

  const loadPrompts = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await api.get('/prompts');
      setPrompts(response.data.data || []);
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to load prompts';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setFormData({name: '', template: '', commit_message: '', label: undefined});
    setFormErrors({});
    setShowSidebar(true);
  };

  const handleViewVersions = (prompt: PromptWithLatestVersion) => {
    navigate(`/prompts/${prompt.name}/versions`);
  };

  const handleDelete = async (prompt: PromptWithLatestVersion) => {
    if (!window.confirm(`Are you sure you want to delete the prompt "${prompt.name}" and all its versions?`)) {
      return;
    }

    try {
      await api.delete(`/prompts?name=${encodeURIComponent(prompt.name)}`);
      await loadPrompts();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to delete prompt';
      setError(errorMessage);
    }
  };

  const validateForm = (): boolean => {
    const errors: { [key: string]: string } = {};

    if (!formData.name.trim()) {
      errors.name = 'Name is required';
    } else if (formData.name.length > 255) {
      errors.name = 'Name must be less than 255 characters';
    }

    if (!formData.template.trim()) {
      errors.template = 'Template is required';
    }

    if (!formData.commit_message.trim()) {
      errors.commit_message = 'Commit message is required';
    } else if (formData.commit_message.length > 500) {
      errors.commit_message = 'Commit message must be less than 500 characters';
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
      await api.post('/prompts', formData);
      setShowSidebar(false);
      await loadPrompts();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to create prompt';
      setError(errorMessage);
    }
  };

  const handleCancel = () => {
    setShowSidebar(false);
    setFormData({name: '', template: '', commit_message: '', label: undefined});
    setFormErrors({});
  };

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

  const getLabelBadge = (label?: string) => {
    if (!label) return null;

    return <Chip label={label} variant="outlined" size="small" />
  };

  const columns: Column<PromptWithLatestVersion>[] = [
    {
      key: 'name',
      label: 'Name',
      render: (prompt) => (
        <Box display="flex" flexDirection="column" gap="4px">
          <div>{prompt.name}</div>
          <div>
          {prompt.latest_label && getLabelBadge(prompt.latest_label)}
          </div>
        </Box>
      )
    },
    {
      key: 'latest_version',
      label: 'Latest Version',
      render: (prompt) => prompt.latest_version || 'No versions'
    },
    {
      key: 'latest_commit_message',
      label: 'Latest Commit',
      render: (prompt) => prompt.latest_commit_message || '-'
    },
    {
      key: 'created_at',
      label: 'Created',
      render: (prompt) => formatDate(prompt.created_at)
    }
  ];

  const actions: Action<PromptWithLatestVersion>[] = [
    {
      label: 'View Versions',
      onClick: handleViewVersions,
      icon: <Visibility />
    },
    {
      label: 'Delete',
      onClick: handleDelete,
      icon: <DeleteIcon />
    }
  ];

  return (
    <PageContainer>
      <PageHeader>
        <div>
          <PageTitle>Prompts</PageTitle>
          <PageSubtitle>Manage your prompts</PageSubtitle>
        </div>
        <Button
          variant="contained"
          color="primary"
          onClick={handleCreate}
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path
              d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
          </svg>
          Create Prompt
        </Button>
      </PageHeader>

      {error && (
        <Typography color="error">
          {error}
        </Typography>
      )}

      <Box display="flex" flexDirection="column" flex={1}>
        <DataTable
          data={prompts}
          columns={columns}
          actions={actions}
          loading={loading}
          emptyState={{
            icon: '🔗',
            title: 'No prompts yet',
            description: 'No prompts found. Create your first prompt to get started.',
            actionLabel: 'Create Prompt',
            onAction: handleCreate
          }}
        />
      </Box>

      <SlideDialog
        open={showSidebar}
        onClose={handleCancel}
        title="Create New Prompt"
        actions={
          <>
            <Button
              color="info"
              type="button"
              onClick={handleCancel}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              color="primary"
              variant="contained"
              form="prompt-form"
            >
              Create Prompt
            </Button>
          </>
        }
      >
        <form id="prompt-form" onSubmit={handleSubmit}>
          <InputGroup>
            <InputLabel>Name</InputLabel>
            <Input
              id="name"
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({...formData, name: e.target.value})}
              error={!!formErrors.name}
              placeholder="Enter prompt name"
              helperText={formErrors.name}
              fullWidth
            />
          </InputGroup>

          <InputGroup>
            <InputLabel>Template</InputLabel>
            <Input
              id="template"
              value={formData.template}
              onChange={(e) => setFormData({...formData, template: e.target.value})}
              error={!!formErrors.template}
              placeholder="Enter your prompt template with {{variables}}"
              helperText={formErrors.template}
              rows={6}
              multiline
              fullWidth
            />
          </InputGroup>

          <InputGroup>
            <InputLabel>Commit Message</InputLabel>
            <Input
              id="commit_message"
              type="text"
              value={formData.commit_message}
              onChange={(e) => setFormData({...formData, commit_message: e.target.value})}
              placeholder="Describe this prompt"
              error={!!formErrors.commit_message}
              helperText={formErrors.commit_message}
              fullWidth
            />
          </InputGroup>

          <InputGroup>
            <InputLabel>Label (Optional)</InputLabel>
            <Select
              id="label"
              value={formData.label || ''}
              onChange={(e) => setFormData({
                ...formData,
                label: e.target.value ? e.target.value : undefined
              })}
              fullWidth
              MenuProps={{
                style: { zIndex: 1500 },
                PaperProps: {
                  style: { zIndex: 1500 }
                }
              }}
            >
              <MenuItem value="">No label</MenuItem>
              <MenuItem value="production">Production</MenuItem>
              <MenuItem value="latest">Latest</MenuItem>
            </Select>
          </InputGroup>
        </form>
      </SlideDialog>
    </PageContainer>
  );
};
