import React, {useCallback, useEffect, useMemo, useState} from 'react';
import {
  Box,
  Chip,
  IconButton
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';

import {useProjectContext, Project} from '../../contexts/ProjectContext';
import {PageContainer, PageHeader, PageSubtitle, PageTitle} from '../../components/shared/Page';
import {Button} from '../../components/shared/Buttons';
import {Action, Column, DataTable} from '../../components/DataTable/DataTable';
import {SlideDialog} from '../../components/shared/Dialog';
import {Input, InputGroup, InputLabel} from '../../components/shared/Input';

const getErrorMessage = (error: any, fallback: string) => (
  error?.response?.data?.message ||
  error?.response?.data?.errorDetails?.error ||
  error?.message ||
  fallback
);

const formatDate = (date: string) => {
  if (!date) {
    return '-';
  }
  const parsed = new Date(date);
  if (Number.isNaN(parsed.getTime())) {
    return date;
  }
  return parsed.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
};

type DialogMode = 'create' | 'edit';

export const ProjectsPage: React.FC = () => {
  const {
    projects,
    selectedProjectId,
    selectProject,
    createProject,
    updateProject,
    deleteProject,
    refreshProjects,
    loading,
    error
  } = useProjectContext();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [dialogMode, setDialogMode] = useState<DialogMode>('create');
  const [activeProject, setActiveProject] = useState<Project | null>(null);
  const [projectName, setProjectName] = useState('');
  const [defaultKey, setDefaultKey] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    void refreshProjects();
  }, [refreshProjects]);

  const closeDialog = useCallback((force = false) => {
    if (!force && submitting) {
      return;
    }
    setDialogOpen(false);
    setFormError(null);
    setActiveProject(null);
    setProjectName('');
    setDefaultKey('');
  }, [submitting]);

  const openCreateDialog = useCallback(() => {
    setDialogMode('create');
    setActiveProject(null);
    setProjectName('');
    setDefaultKey('');
    setFormError(null);
    setDialogOpen(true);
  }, []);

  const openEditDialog = useCallback((project: Project) => {
    setDialogMode('edit');
    setActiveProject(project);
    setProjectName(project.name);
    setDefaultKey(project.default_key || '');
    setFormError(null);
    setDialogOpen(true);
  }, []);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    const trimmedName = projectName.trim();
    const trimmedKey = defaultKey.trim() || null;

    if (!trimmedName) {
      setFormError('Name is required');
      return;
    }

    setSubmitting(true);
    try {
      if (dialogMode === 'create') {
        await createProject(trimmedName, trimmedKey);
      } else if (activeProject) {
        const updatePayload: {name?: string; default_key?: string | null} = {};
        if (trimmedName !== activeProject.name) {
          updatePayload.name = trimmedName;
        }
        if (trimmedKey !== (activeProject.default_key || null)) {
          updatePayload.default_key = trimmedKey;
        }
        if (Object.keys(updatePayload).length > 0) {
          await updateProject(activeProject.id, updatePayload);
        }
      }
      closeDialog(true);
    } catch (requestError) {
      setFormError(getErrorMessage(requestError, 'Failed to save project'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = useCallback(async (project: Project) => {
    const confirmed = window.confirm(`Delete project "${project.name}"? This action cannot be undone.`);
    if (!confirmed) {
      return;
    }

    try {
      await deleteProject(project.id);
    } catch (requestError) {
      window.alert(getErrorMessage(requestError, 'Failed to delete project'));
    }
  }, [deleteProject]);

  const columns: Column<Project>[] = useMemo(() => [
    {
      key: 'name',
      label: 'Name',
      render: (_, project) => (
        <Box display="flex" alignItems="center" gap={1}>
          <span>{project.name}</span>
          {project.id === selectedProjectId && (
            <Chip size="small" color="success" label="Active" />
          )}
        </Box>
      )
    },
    {
      key: 'default_key',
      label: 'Default Key',
      render: (_, project) => (
        project.default_key ? (
          <Chip size="small" label="Set" color="info" />
        ) : (
          <span style={{ color: '#666', fontSize: '0.875rem' }}>Not set</span>
        )
      )
    },
    {
      key: 'created_at',
      label: 'Created',
      render: (_, project) => formatDate(project.created_at)
    },
    {
      key: 'updated_at',
      label: 'Updated',
      render: (_, project) => formatDate(project.updated_at)
    }
  ], [selectedProjectId]);

  const actions: Action<Project>[] = useMemo(() => [
    {
      label: 'Edit',
      icon: <EditIcon fontSize="small" />,
      onClick: openEditDialog,
      title: 'Rename project'
    },
    {
      label: 'Delete',
      icon: <DeleteIcon fontSize="small" />,
      onClick: handleDelete,
      className: 'deleteButton',
      title: 'Delete project'
    }
  ], [handleDelete, openEditDialog]);

  return (
    <PageContainer>
      <PageHeader>
        <div>
          <PageTitle>Projects</PageTitle>
          <PageSubtitle>Organize conversations and resources by project</PageSubtitle>
        </div>
        <Button variant="contained" color="primary" startIcon={<AddIcon />} onClick={openCreateDialog}>
          New Project
        </Button>
      </PageHeader>

      <Box display="flex" flexDirection="column" flex={1}>
        <DataTable
          data={projects}
          columns={columns}
          actions={actions}
          loading={loading}
          error={error}
          onRetry={refreshProjects}
          emptyState={{
            icon: '🗂️',
            title: 'No projects yet',
            description: 'Create your first project to get started',
            actionLabel: 'Create Project',
            onAction: openCreateDialog
          }}
        />
      </Box>

      <SlideDialog
        open={dialogOpen}
        onClose={() => closeDialog()}
        title={dialogMode === 'create' ? 'Create Project' : 'Edit Project'}
        actions={
          <>
            <Button color="inherit" onClick={() => closeDialog()} disabled={submitting}>
              Cancel
            </Button>
            <Button variant="contained" color="primary" type="submit" form="project-form" disabled={submitting}>
              {submitting ? 'Saving...' : 'Save'}
            </Button>
          </>
        }
      >
        <form id="project-form" onSubmit={handleSubmit}>
          <InputGroup>
            <InputLabel>Project Name *</InputLabel>
            <Input
              id="project-name-input"
              size="small"
              value={projectName}
              onChange={(event) => {
                setProjectName(event.target.value);
                setFormError(null);
              }}
              placeholder="e.g., Growth Experiments"
              error={!!formError}
              helperText={formError}
              fullWidth
            />
          </InputGroup>
          <InputGroup>
            <InputLabel>Default Virtual Key</InputLabel>
            <Input
              id="default-key-input"
              size="small"
              value={defaultKey}
              onChange={(event) => {
                setDefaultKey(event.target.value);
                setFormError(null);
              }}
              placeholder="e.g., sk-amg-..."
              fullWidth
            />
          </InputGroup>
        </form>
      </SlideDialog>
    </PageContainer>
  );
};

