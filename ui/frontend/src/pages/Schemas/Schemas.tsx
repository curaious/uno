import React, { useEffect, useState } from 'react';
import { api } from '../../api';
import { Schema, CreateSchemaRequest, JSONSchemaDefinition } from '../../components/Chat/types';
import { Action, Column, DataTable } from '../../components/DataTable/DataTable';
import { PageContainer, PageHeader, PageSubtitle, PageTitle } from "../../components/shared/Page";
import { Button } from '../../components/shared/Buttons';
import { Box, Chip, Typography } from "@mui/material";
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import { SlideDialog } from "../../components/shared/Dialog";
import { SchemaBuilder } from './SchemaBuilder';

const defaultSchema: JSONSchemaDefinition = {
  type: 'object',
  properties: {},
  required: []
};

export const Schemas: React.FC = () => {
  const [schemas, setSchemas] = useState<Schema[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showBuilder, setShowBuilder] = useState(false);
  const [editingSchema, setEditingSchema] = useState<Schema | null>(null);
  const [formData, setFormData] = useState<CreateSchemaRequest>({
    name: '',
    description: '',
    schema: defaultSchema,
    source_type: 'manual'
  });

  useEffect(() => {
    loadSchemas();
  }, []);

  const loadSchemas = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await api.get('/schemas');
      setSchemas(response.data.data || []);
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to load schemas';
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditingSchema(null);
    setFormData({
      name: '',
      description: '',
      schema: defaultSchema,
      source_type: 'manual'
    });
    setShowBuilder(true);
  };

  const handleEdit = (schema: Schema) => {
    setEditingSchema(schema);
    setFormData({
      name: schema.name,
      description: schema.description || '',
      schema: schema.schema,
      source_type: schema.source_type,
      source_content: schema.source_content
    });
    setShowBuilder(true);
  };

  const handleDuplicate = (schema: Schema) => {
    setEditingSchema(null);
    setFormData({
      name: `${schema.name}-copy`,
      description: schema.description || '',
      schema: schema.schema,
      source_type: 'manual'
    });
    setShowBuilder(true);
  };

  const handleDelete = async (schema: Schema) => {
    if (!window.confirm(`Are you sure you want to delete the schema "${schema.name}"?`)) {
      return;
    }

    try {
      await api.delete(`/schemas/${schema.id}`);
      await loadSchemas();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        'Failed to delete schema';
      setError(errorMessage);
    }
  };

  const handleSubmit = async (data: CreateSchemaRequest) => {
    try {
      if (editingSchema) {
        await api.put(`/schemas/${editingSchema.id}`, data);
      } else {
        await api.post('/schemas', data);
      }
      setShowBuilder(false);
      await loadSchemas();
    } catch (err: any) {
      const errorMessage = err.response?.data?.message ||
        err.response?.data?.errorDetails?.message ||
        `Failed to ${editingSchema ? 'update' : 'create'} schema`;
      throw new Error(errorMessage);
    }
  };

  const handleCancel = () => {
    setShowBuilder(false);
    setEditingSchema(null);
    setFormData({
      name: '',
      description: '',
      schema: defaultSchema,
      source_type: 'manual'
    });
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

  const getSourceTypeBadge = (sourceType: string) => {
    const colors: Record<string, 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'info'> = {
      'manual': 'default',
      'go_struct': 'info',
      'typescript': 'primary'
    };
    const labels: Record<string, string> = {
      'manual': 'Manual',
      'go_struct': 'Go Struct',
      'typescript': 'TypeScript'
    };
    return <Chip label={labels[sourceType] || sourceType} color={colors[sourceType] || 'default'} size="small" variant="outlined" />;
  };

  const getPropertyCount = (schema: JSONSchemaDefinition) => {
    if (schema.type === 'object' && schema.properties) {
      return Object.keys(schema.properties).length;
    }
    return 0;
  };

  const columns: Column<Schema>[] = [
    {
      key: 'name',
      label: 'Name',
      render: (schema) => (
        <Box display="flex" flexDirection="column" gap="4px">
          <Typography variant="body2" fontWeight={500}>{schema.name}</Typography>
          {schema.description && (
            <Typography variant="caption" color="text.secondary">{schema.description}</Typography>
          )}
        </Box>
      )
    },
    {
      key: 'source_type',
      label: 'Source',
      render: (schema) => getSourceTypeBadge(schema.source_type)
    },
    {
      key: 'schema',
      label: 'Properties',
      render: (schema) => (
        <Chip 
          label={`${getPropertyCount(schema.schema)} properties`} 
          size="small" 
          variant="outlined"
          sx={{ fontFamily: 'monospace', fontSize: '11px' }}
        />
      )
    },
    {
      key: 'created_at',
      label: 'Created',
      render: (schema) => formatDate(schema.created_at)
    }
  ];

  const actions: Action<Schema>[] = [
    {
      label: 'Edit',
      onClick: handleEdit,
      icon: <EditIcon />
    },
    {
      label: 'Duplicate',
      onClick: handleDuplicate,
      icon: <ContentCopyIcon />
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
          <PageTitle>JSON Schemas</PageTitle>
          <PageSubtitle>Define and manage JSON schemas for structured outputs</PageSubtitle>
        </div>
        <Button
          variant="contained"
          color="primary"
          onClick={handleCreate}
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2Z"/>
          </svg>
          Create Schema
        </Button>
      </PageHeader>

      {error && (
        <Typography color="error" sx={{ mb: 2 }}>
          {error}
        </Typography>
      )}

      <Box display="flex" flexDirection="column" flex={1}>
        <DataTable
          data={schemas}
          columns={columns}
          actions={actions}
          loading={loading}
          emptyState={{
            icon: '📋',
            title: 'No schemas yet',
            description: 'Create your first JSON schema to define structured outputs for your agents.',
            actionLabel: 'Create Schema',
            onAction: handleCreate
          }}
        />
      </Box>

      <SlideDialog
        open={showBuilder}
        onClose={handleCancel}
        title={editingSchema ? `Edit Schema: ${editingSchema.name}` : 'Create New Schema'}
        maxWidth="lg"
        actions={<></>}
      >
        <SchemaBuilder
          key={editingSchema?.id || 'new'}
          initialData={formData}
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          isEditing={!!editingSchema}
        />
      </SlideDialog>
    </PageContainer>
  );
};

