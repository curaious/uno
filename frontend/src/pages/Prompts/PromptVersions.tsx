import React, {useCallback, useEffect, useState} from 'react';
import {useNavigate, useParams} from 'react-router';
import {api} from '../../api';
import {CreatePromptVersionRequest, PromptVersion, UpdatePromptVersionLabelRequest} from '../../components/Chat/types';
import {Action, Column, DataTable} from '../../components/DataTable/DataTable';
import {Box, Chip, IconButton, MenuItem, Typography} from '@mui/material';
import {Button} from "../../components/shared/Buttons";
import {PageContainer, PageHeader, PageTitle} from "../../components/shared/Page";
import Close from "@mui/icons-material/Close";
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import FactoryIcon from '@mui/icons-material/Factory';
import StarIcon from '@mui/icons-material/Star';
import LabelOffIcon from '@mui/icons-material/LabelOff';
import DeleteIcon from '@mui/icons-material/Delete';
import DescriptionIcon from '@mui/icons-material/Description';
import {Input, InputGroup, InputLabel, Select} from "../../components/shared/Input";
import {SlideDialog} from "../../components/shared/Dialog";
import Editor from '@monaco-editor/react';
import styles from './PromptVersions.module.css';

export const PromptVersions: React.FC = props => {
  const {name} = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [versions, setVersions] = useState<PromptVersion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showSidebar, setShowSidebar] = useState(false);
  const [formData, setFormData] = useState<CreatePromptVersionRequest>({
    template: '',
    commit_message: '',
    label: undefined
  });
  const [formErrors, setFormErrors] = useState<{ [key: string]: string }>({});
  const [detectedVariables, setDetectedVariables] = useState<string[]>([]);

  const extractVariables = (template: string): string[] => {
    const regex = /\{\{([^}]+)\}\}/g;
    const variables: string[] = [];
    let match;

    while ((match = regex.exec(template)) !== null) {
      const variable = match[1].trim();
      if (variable && !variables.includes(variable)) {
        variables.push(variable);
      }
    }

    return variables;
  };

  const loadVersions = useCallback(async () => {
    if (!name) return;

    try {
      setLoading(true);
      setError(null);
      const response = await api.get(`/prompts/${encodeURIComponent(name)}/versions`);
      setVersions(response.data.data || []);
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to load prompt versions';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  }, [name]);

  // Load versions on component mount
  useEffect(() => {
    if (name) {
      loadVersions();
    }
  }, [name, loadVersions]);

  // Detect variables in template
  useEffect(() => {
    if (formData.template) {
      const variables = extractVariables(formData.template);
      setDetectedVariables(variables);
    } else {
      setDetectedVariables([]);
    }
  }, [formData.template]);

  const handleCreate = () => {
    // Find the latest version to pre-populate the form
    const latestVersion = versions.length > 0 
      ? versions.reduce((latest, current) => 
          current.version > latest.version ? current : latest
        )
      : null;
    
    setFormData({
      template: latestVersion?.template || '',
      commit_message: '',
      label: undefined
    });
    setFormErrors({});
    setShowSidebar(true);
  };

  const handleSetLabel = async (version: PromptVersion, label: string) => {
    if (!name) return;

    try {
      const request: UpdatePromptVersionLabelRequest = {label};
      await api.patch(`/prompts/${encodeURIComponent(name)}/versions/${version.version}/label`, request);
      await loadVersions();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to update label';
      setError(errorMessage);
    }
  };

  const handleRemoveLabel = async (version: PromptVersion) => {
    if (!name) return;

    try {
      const request: UpdatePromptVersionLabelRequest = {label: undefined};
      await api.patch(`/prompts/${encodeURIComponent(name)}/versions/${version.version}/label`, request);
      await loadVersions();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to remove label';
      setError(errorMessage);
    }
  };

  const handleDelete = async (version: PromptVersion) => {
    if (!name) return;

    if (!window.confirm(`Are you sure you want to delete version ${version.version}?`)) {
      return;
    }

    try {
      await api.delete(`/prompts/${encodeURIComponent(name)}/versions/${version.version}`);
      await loadVersions();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to delete version';
      setError(errorMessage);
    }
  };

  const validateForm = (): boolean => {
    const errors: { [key: string]: string } = {};

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

    if (!name || !validateForm()) {
      return;
    }

    try {
      await api.post(`/prompts/${encodeURIComponent(name)}/versions`, formData);
      setShowSidebar(false);
      await loadVersions();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to create prompt version';
      setError(errorMessage);
    }
  };

  const handleCancel = () => {
    setShowSidebar(false);
    setFormData({template: '', commit_message: '', label: undefined});
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

    return <Chip label={label} variant="outlined" size="small"/>
  };

  const columns: Column<PromptVersion>[] = [
    {
      key: 'version',
      label: 'Version',
      render: (version) => (
        <Box display="flex" flexDirection="column" gap="4px">
          <div>{version.version}</div>
          <div>
            {version.label && getLabelBadge(version.label)}
          </div>
        </Box>
      )
    },
    {
      key: 'commit_message',
      label: 'Commit Message',
      render: (version) => version.commit_message || ''
    },
    {
      key: 'template',
      label: 'Template Preview',
      render: (version) => (
        <div>
          {version.template && version.template.length > 100
            ? `${version.template.substring(0, 100)}...`
            : version.template || ''
          }
        </div>
      )
    },
    {
      key: 'created_at',
      label: 'Created',
      render: (version) => formatDate(version.created_at)
    }
  ];

  const actions: Action<PromptVersion>[] = [
    {
      label: 'Set Production',
      onClick: (version) => handleSetLabel(version, 'production'),
      icon: <FactoryIcon />
    },
    {
      label: 'Set Latest',
      onClick: (version) => handleSetLabel(version, 'latest'),
      icon: <StarIcon />
    },
    {
      label: 'Remove Label',
      onClick: (version) => handleRemoveLabel(version),
      icon: <LabelOffIcon />
    },
    {
      label: 'Delete',
      onClick: handleDelete,
      icon: <DeleteIcon />
    }
  ];

  if (!name) {
    return <div>Invalid prompt name</div>;
  }

  return (
    <PageContainer>
      <div className={styles.backButton} onClick={() => navigate('/agent-framework/prompts')}>
        <ArrowBackIcon sx={{ fontSize: 18 }} />
        Back to Prompts
      </div>
      <PageHeader>
        <Box display="flex" flexDirection="column" justifyContent="flex-start">
          <div>

          </div>
          <PageTitle>{name}</PageTitle>
        </Box>
        <Button
          variant="contained"
          color="primary"
          onClick={handleCreate}
        >
          Create Version
        </Button>
      </PageHeader>

      {error && (
        <Box style={{padding: '20px'}}>
          <Typography color="error">{error}</Typography>
        </Box>
      )}

      <DataTable
        data={versions}
        columns={columns}
        actions={actions}
        loading={loading}
        emptyState={{
          icon: <DescriptionIcon sx={{ fontSize: 48, color: 'var(--text-secondary)' }} />,
          title: 'No versions yet',
          description: 'No versions found. Create the first version to get started.',
          actionLabel: 'Create Version',
          onAction: handleCreate
        }}
      />

      <SlideDialog
        open={showSidebar}
        onClose={handleCancel}
        title="Create New Version"
        maxWidth="90vw"
        width="90vw"
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
              form="prompt-version-form"
            >
              Create Version
            </Button>
          </>
        }
      >
        <form id="prompt-version-form" onSubmit={handleSubmit} style={{ height: '100%', display: 'flex' }}>
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', p: 2 }}>
            <InputGroup>
              <label htmlFor="template">Template</label>
              <Box sx={{ 
                border: formErrors.template ? '1px solid #d32f2f' : '1px solid #ccc', 
                borderRadius: 1, 
                flex: 1,
                minHeight: 'calc(100vh - 200px)'
              }}>
                <Editor
                  height="calc(100vh - 200px)"
                  defaultLanguage="markdown"
                  value={formData.template}
                  onChange={(value) => setFormData({...formData, template: value || ''})}
                  options={{
                    minimap: { enabled: false },
                    scrollBeyondLastLine: false,
                    wordWrap: 'on',
                    automaticLayout: true,
                    fontSize: 14,
                    lineNumbers: 'on',
                    folding: true,
                    renderWhitespace: 'selection',
                    selectOnLineNumbers: true,
                    cursorStyle: 'line',
                    theme: 'vs-dark'
                  }}
                  theme="vs-dark"
                />
              </Box>
              {formErrors.template && (
                <Typography color="error" variant="caption" sx={{ mt: 0.5, display: 'block' }}>
                  {formErrors.template}
                </Typography>
              )}
              {detectedVariables.length > 0 && (
                <Box sx={{ mt: 1 }}>
                  <Typography variant="body2" color="text.secondary">
                    <strong>Detected variables:</strong> {detectedVariables.join(', ')}
                  </Typography>
                </Box>
              )}
            </InputGroup>
          </Box>

          <Box sx={{ width: '350px', p: 2, borderLeft: '1px solid #e0e0e0', display: 'flex', flexDirection: 'column' }}>
            <InputGroup>
              <InputLabel>Commit Message</InputLabel>
              <Input
                id="commit_message"
                type="text"
                value={formData.commit_message}
                onChange={(e) => setFormData({...formData, commit_message: e.target.value})}
                error={!!formErrors.commit_message}
                placeholder="Describe the changes in this version"
                helperText={formErrors.commit_message}
                fullWidth
              />
              {formErrors.commit_message && (
                <span>{formErrors.commit_message}</span>
              )}
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
          </Box>
        </form>
      </SlideDialog>
    </PageContainer>
  );
};
