import React, { useState, useCallback } from 'react';
import { Box, Typography } from '@mui/material';
import { useProjectContext } from '../../contexts/ProjectContext';
import { PageContainer, PageHeader, PageSubtitle, PageTitle } from '../shared/Page';
import { Button } from '../shared/Buttons';
import { SlideDialog } from '../shared/Dialog';
import { Input, InputGroup, InputLabel } from '../shared/Input';
import AddIcon from '@mui/icons-material/Add';

const getErrorMessage = (error: any, fallback: string) => (
  error?.response?.data?.message ||
  error?.response?.data?.errorDetails?.error ||
  error?.message ||
  fallback
);

export const NoProjects: React.FC = () => {
  const { createProject } = useProjectContext();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [projectName, setProjectName] = useState('');
  const [defaultKey, setDefaultKey] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const closeDialog = useCallback(() => {
    if (submitting) {
      return;
    }
    setDialogOpen(false);
    setFormError(null);
    setProjectName('');
    setDefaultKey('');
  }, [submitting]);

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
      await createProject(trimmedName, trimmedKey);
      closeDialog();
    } catch (requestError) {
      setFormError(getErrorMessage(requestError, 'Failed to create project'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <PageContainer>
      <PageHeader>
        <div>
          <PageTitle>Create Your First Project</PageTitle>
          <PageSubtitle>Projects help you organize conversations and resources</PageSubtitle>
        </div>
        <Button 
          variant="contained" 
          color="primary" 
          startIcon={<AddIcon />} 
          onClick={() => setDialogOpen(true)}
        >
          Create Project
        </Button>
      </PageHeader>

      <Box 
        display="flex" 
        flexDirection="column" 
        alignItems="center" 
        justifyContent="center" 
        flex={1}
        sx={{ 
          textAlign: 'center',
          padding: 4,
          gap: 2
        }}
      >
        <Typography variant="h2" sx={{ color: 'var(--text-secondary)', mb: 2 }}>
          No projects yet
        </Typography>
        <Typography variant="body1" sx={{ color: 'var(--text-secondary)', mb: 3, maxWidth: 500 }}>
          Create your first project to get started with the Agent Framework. Projects help you organize your agents, prompts, and conversations.
        </Typography>
        <Button 
          variant="contained" 
          color="primary" 
          size="large"
          startIcon={<AddIcon />} 
          onClick={() => setDialogOpen(true)}
        >
          Create Your First Project
        </Button>
      </Box>

      <SlideDialog
        open={dialogOpen}
        onClose={closeDialog}
        title="Create Project"
        actions={
          <>
            <Button color="inherit" onClick={closeDialog} disabled={submitting}>
              Cancel
            </Button>
            <Button variant="contained" color="primary" type="submit" form="project-form" disabled={submitting}>
              {submitting ? 'Creating...' : 'Create'}
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
              placeholder="e.g., sk-uno-..."
              fullWidth
            />
          </InputGroup>
        </form>
      </SlideDialog>
    </PageContainer>
  );
};


