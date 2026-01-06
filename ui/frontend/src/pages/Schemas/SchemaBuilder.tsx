import React, { useState, useCallback } from 'react';
import {
  Box,
  Typography,
  IconButton,
  Collapse,
  Tooltip,
  MenuItem,
  Checkbox,
  FormControlLabel,
  styled,
  alpha
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import CodeIcon from '@mui/icons-material/Code';
import { Button } from '../../components/shared/Buttons';
import { Input, InputGroup, InputLabel, Select } from '../../components/shared/Input';
import { CreateSchemaRequest, JSONSchemaProperty, JSONSchemaDefinition, JSONSchemaType } from '../../components/Chat/types';

const BuilderContainer = styled(Box)(({ theme }) => ({
  display: 'flex',
  flexDirection: 'column',
  gap: theme.spacing(3),
  minHeight: '60vh',
}));

const PropertyCard = styled(Box)(({ theme }) => ({
  border: `1px solid ${theme.palette.divider}`,
  borderRadius: theme.shape.borderRadius,
  background: alpha(theme.palette.background.paper, 0.5),
  overflow: 'hidden',
}));

const PropertyHeader = styled(Box)(({ theme }) => ({
  display: 'flex',
  alignItems: 'center',
  gap: theme.spacing(1),
  padding: theme.spacing(1, 1.5),
  background: alpha(theme.palette.primary.main, 0.05),
  borderBottom: `1px solid ${theme.palette.divider}`,
  cursor: 'pointer',
  '&:hover': {
    background: alpha(theme.palette.primary.main, 0.1),
  },
}));

const PropertyContent = styled(Box)(({ theme }) => ({
  padding: theme.spacing(2),
  display: 'flex',
  flexDirection: 'column',
  gap: theme.spacing(2),
}));

const NestedProperties = styled(Box)(({ theme }) => ({
  marginLeft: theme.spacing(3),
  paddingLeft: theme.spacing(2),
  borderLeft: `2px solid ${theme.palette.divider}`,
  display: 'flex',
  flexDirection: 'column',
  gap: theme.spacing(1.5),
}));

const TypeBadge = styled(Box)<{ typeName: JSONSchemaType }>(({ theme, typeName }) => {
  const colors: Record<JSONSchemaType, string> = {
    string: '#4caf50',
    number: '#2196f3',
    integer: '#3f51b5',
    boolean: '#ff9800',
    object: '#9c27b0',
    array: '#00bcd4',
    null: '#607d8b',
  };
  return {
    padding: '2px 8px',
    borderRadius: 4,
    fontSize: 11,
    fontWeight: 600,
    fontFamily: 'monospace',
    background: alpha(colors[typeName] || '#666', 0.2),
    color: colors[typeName] || '#666',
    textTransform: 'uppercase',
  };
});

const CodePreview = styled(Box)(({ theme }) => ({
  background: '#1a1a2e',
  borderRadius: theme.shape.borderRadius,
  padding: theme.spacing(2),
  fontFamily: 'JetBrains Mono, Fira Code, Monaco, monospace',
  fontSize: 12,
  lineHeight: 1.6,
  overflow: 'auto',
  maxHeight: 400,
  color: '#e0e0e0',
  '& .key': { color: '#82aaff' },
  '& .string': { color: '#c3e88d' },
  '& .number': { color: '#f78c6c' },
  '& .boolean': { color: '#ff9cac' },
  '& .null': { color: '#89ddff' },
  '& .bracket': { color: '#89ddff' },
}));

const ActionBar = styled(Box)(({ theme }) => ({
  display: 'flex',
  justifyContent: 'flex-end',
  gap: theme.spacing(1),
  paddingTop: theme.spacing(2),
  borderTop: `1px solid ${theme.palette.divider}`,
}));

const schemaTypes: JSONSchemaType[] = ['string', 'number', 'integer', 'boolean', 'object', 'array'];

// Recursively ensures all object types have additionalProperties: false
const ensureStrictObjects = (obj: any): any => {
  if (!obj || typeof obj !== 'object') return obj;

  const result = { ...obj };

  // If this is an object type, add additionalProperties: false
  if (result.type === 'object') {
    result.additionalProperties = false;
  }

  // Process nested properties
  if (result.properties) {
    const newProps: Record<string, any> = {};
    for (const [key, value] of Object.entries(result.properties)) {
      newProps[key] = ensureStrictObjects(value);
    }
    result.properties = newProps;
  }

  // Process array items
  if (result.items) {
    result.items = ensureStrictObjects(result.items);
  }

  return result;
};

const stringFormats = [
  { value: '', label: 'None' },
  { value: 'date-time', label: 'Date-Time' },
  { value: 'date', label: 'Date' },
  { value: 'time', label: 'Time' },
  { value: 'email', label: 'Email' },
  { value: 'uri', label: 'URI' },
  { value: 'uuid', label: 'UUID' },
  { value: 'hostname', label: 'Hostname' },
  { value: 'ipv4', label: 'IPv4' },
  { value: 'ipv6', label: 'IPv6' },
];

interface PropertyEditorProps {
  name: string;
  property: JSONSchemaProperty;
  isRequired: boolean;
  onUpdate: (name: string, property: JSONSchemaProperty, newName?: string) => void;
  onDelete: (name: string) => void;
  onToggleRequired: (name: string) => void;
  depth?: number;
}

const PropertyEditor: React.FC<PropertyEditorProps> = ({
  name,
  property,
  isRequired,
  onUpdate,
  onDelete,
  onToggleRequired,
  depth = 0,
}) => {
  const [expanded, setExpanded] = useState(true);
  const [localName, setLocalName] = useState(name);
  const type = (property.type as JSONSchemaType) || 'string';

  const handleNameChange = (newName: string) => {
    setLocalName(newName);
  };

  const handleNameBlur = () => {
    if (localName !== name && localName.trim()) {
      onUpdate(name, property, localName.trim());
    }
  };

  const handleTypeChange = (newType: JSONSchemaType) => {
    const updated: JSONSchemaProperty = { ...property, type: newType };
    
    // Reset type-specific fields
    if (newType === 'object') {
      updated.properties = updated.properties || {};
      updated.required = updated.required || [];
      delete updated.items;
      delete updated.enum;
      delete updated.format;
      delete updated.minimum;
      delete updated.maximum;
      delete updated.minLength;
      delete updated.maxLength;
      delete updated.pattern;
    } else if (newType === 'array') {
      updated.items = updated.items || { type: 'string' };
      delete updated.properties;
      delete updated.required;
      delete updated.enum;
      delete updated.format;
      delete updated.minimum;
      delete updated.maximum;
      delete updated.minLength;
      delete updated.maxLength;
      delete updated.pattern;
    } else {
      delete updated.properties;
      delete updated.required;
      delete updated.items;
    }
    
    onUpdate(name, updated);
  };

  const handleFieldChange = (field: keyof JSONSchemaProperty, value: any) => {
    onUpdate(name, { ...property, [field]: value || undefined });
  };

  const addNestedProperty = () => {
    const props = property.properties || {};
    let newKey = 'newProperty';
    let counter = 1;
    while (props[newKey]) {
      newKey = `newProperty${counter++}`;
    }
    onUpdate(name, {
      ...property,
      properties: { ...props, [newKey]: { type: 'string' } },
    });
  };

  const updateNestedProperty = (propName: string, prop: JSONSchemaProperty, newName?: string) => {
    const props = { ...property.properties };
    if (newName && newName !== propName) {
      delete props[propName];
      props[newName] = prop;
      // Update required array if needed
      const req = (property.required || []).map(r => r === propName ? newName : r);
      onUpdate(name, { ...property, properties: props, required: req });
    } else {
      props[propName] = prop;
      onUpdate(name, { ...property, properties: props });
    }
  };

  const deleteNestedProperty = (propName: string) => {
    const props = { ...property.properties };
    delete props[propName];
    const req = (property.required || []).filter(r => r !== propName);
    onUpdate(name, { ...property, properties: props, required: req });
  };

  const toggleNestedRequired = (propName: string) => {
    const req = property.required || [];
    if (req.includes(propName)) {
      onUpdate(name, { ...property, required: req.filter(r => r !== propName) });
    } else {
      onUpdate(name, { ...property, required: [...req, propName] });
    }
  };

  const updateArrayItems = (items: JSONSchemaProperty) => {
    onUpdate(name, { ...property, items });
  };

  return (
    <PropertyCard>
      <PropertyHeader onClick={() => setExpanded(!expanded)}>
        <IconButton size="small" onClick={(e) => { e.stopPropagation(); setExpanded(!expanded); }}>
          {expanded ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
        </IconButton>
        <Typography fontWeight={500} flex={1} fontFamily="monospace" fontSize={13}>
          {name}
        </Typography>
        <TypeBadge typeName={type}>{type}</TypeBadge>
        {isRequired && (
          <Tooltip title="Required">
            <Typography color="error" fontWeight={600} fontSize={12}>*</Typography>
          </Tooltip>
        )}
        <Tooltip title="Delete property">
          <IconButton size="small" color="error" onClick={(e) => { e.stopPropagation(); onDelete(name); }}>
            <DeleteIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </PropertyHeader>

      <Collapse in={expanded}>
        <PropertyContent>
          <Box display="flex" gap={2}>
            <InputGroup sx={{ flex: 1 }}>
              <InputLabel>Property Name</InputLabel>
              <Input
                size="small"
                value={localName}
                onChange={(e) => handleNameChange(e.target.value)}
                onBlur={handleNameBlur}
                placeholder="Property name"
                fullWidth
              />
            </InputGroup>
            <InputGroup sx={{ width: 140 }}>
              <InputLabel>Type</InputLabel>
              <Select
                size="small"
                value={type}
                onChange={(e) => handleTypeChange(e.target.value as JSONSchemaType)}
                fullWidth
                MenuProps={{
                  style: { zIndex: 1500 },
                  PaperProps: { style: { zIndex: 1500 } }
                }}
              >
                {schemaTypes.map(t => (
                  <MenuItem key={t} value={t}>{t}</MenuItem>
                ))}
              </Select>
            </InputGroup>
          </Box>

          <InputGroup>
            <InputLabel>Description</InputLabel>
            <Input
              size="small"
              value={property.description || ''}
              onChange={(e) => handleFieldChange('description', e.target.value)}
              placeholder="Describe this property"
              fullWidth
            />
          </InputGroup>

          <FormControlLabel
            control={
              <Checkbox
                checked={isRequired}
                onChange={() => onToggleRequired(name)}
                size="small"
              />
            }
            label={<Typography variant="body2">Required</Typography>}
          />

          {/* Type-specific fields */}
          {type === 'string' && (
            <Box display="flex" gap={2} flexWrap="wrap">
              <InputGroup sx={{ minWidth: 150 }}>
                <InputLabel>Format</InputLabel>
                <Select
                  size="small"
                  value={property.format || ''}
                  onChange={(e) => handleFieldChange('format', e.target.value)}
                  fullWidth
                  MenuProps={{
                    style: { zIndex: 1500 },
                    PaperProps: { style: { zIndex: 1500 } }
                  }}
                >
                  {stringFormats.map(f => (
                    <MenuItem key={f.value} value={f.value}>{f.label}</MenuItem>
                  ))}
                </Select>
              </InputGroup>
              <InputGroup sx={{ minWidth: 100 }}>
                <InputLabel>Min Length</InputLabel>
                <Input
                  size="small"
                  type="number"
                  value={property.minLength ?? ''}
                  onChange={(e) => handleFieldChange('minLength', e.target.value ? parseInt(e.target.value) : undefined)}
                  fullWidth
                />
              </InputGroup>
              <InputGroup sx={{ minWidth: 100 }}>
                <InputLabel>Max Length</InputLabel>
                <Input
                  size="small"
                  type="number"
                  value={property.maxLength ?? ''}
                  onChange={(e) => handleFieldChange('maxLength', e.target.value ? parseInt(e.target.value) : undefined)}
                  fullWidth
                />
              </InputGroup>
              <InputGroup sx={{ flex: 1, minWidth: 200 }}>
                <InputLabel>Pattern (Regex)</InputLabel>
                <Input
                  size="small"
                  value={property.pattern || ''}
                  onChange={(e) => handleFieldChange('pattern', e.target.value)}
                  placeholder="^[a-zA-Z]+$"
                  fullWidth
                />
              </InputGroup>
            </Box>
          )}

          {(type === 'number' || type === 'integer') && (
            <Box display="flex" gap={2}>
              <InputGroup sx={{ flex: 1 }}>
                <InputLabel>Minimum</InputLabel>
                <Input
                  size="small"
                  type="number"
                  value={property.minimum ?? ''}
                  onChange={(e) => handleFieldChange('minimum', e.target.value ? parseFloat(e.target.value) : undefined)}
                  fullWidth
                />
              </InputGroup>
              <InputGroup sx={{ flex: 1 }}>
                <InputLabel>Maximum</InputLabel>
                <Input
                  size="small"
                  type="number"
                  value={property.maximum ?? ''}
                  onChange={(e) => handleFieldChange('maximum', e.target.value ? parseFloat(e.target.value) : undefined)}
                  fullWidth
                />
              </InputGroup>
            </Box>
          )}

          {type === 'object' && (
            <Box>
              <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
                <Typography variant="subtitle2" color="text.secondary">
                  Nested Properties
                </Typography>
                <Button size="small" onClick={addNestedProperty} startIcon={<AddIcon />}>
                  Add Property
                </Button>
              </Box>
              {property.properties && Object.keys(property.properties).length > 0 && (
                <NestedProperties>
                  {Object.entries(property.properties).map(([key, value]) => (
                    <PropertyEditor
                      key={key}
                      name={key}
                      property={value}
                      isRequired={(property.required || []).includes(key)}
                      onUpdate={updateNestedProperty}
                      onDelete={deleteNestedProperty}
                      onToggleRequired={toggleNestedRequired}
                      depth={depth + 1}
                    />
                  ))}
                </NestedProperties>
              )}
            </Box>
          )}

          {type === 'array' && (
            <Box>
              <Typography variant="subtitle2" color="text.secondary" mb={1}>
                Array Items Schema
              </Typography>
              <Box sx={{ pl: 2, borderLeft: '2px solid', borderColor: 'divider' }}>
                <InputGroup>
                  <InputLabel>Items Type</InputLabel>
                  <Select
                    size="small"
                    value={(property.items?.type as string) || 'string'}
                    onChange={(e) => {
                      const newType = e.target.value as JSONSchemaType;
                      updateArrayItems({
                        type: newType,
                        ...(newType === 'object' ? { properties: {}, required: [] } : {}),
                      });
                    }}
                    fullWidth
                    MenuProps={{
                      style: { zIndex: 1500 },
                      PaperProps: { style: { zIndex: 1500 } }
                    }}
                  >
                    {schemaTypes.map(t => (
                      <MenuItem key={t} value={t}>{t}</MenuItem>
                    ))}
                  </Select>
                </InputGroup>
                {property.items?.type === 'object' && (
                  <Box mt={2}>
                    <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
                      <Typography variant="subtitle2" color="text.secondary">
                        Item Properties
                      </Typography>
                      <Button
                        size="small"
                        onClick={() => {
                          const props = property.items?.properties || {};
                          let newKey = 'newProperty';
                          let counter = 1;
                          while (props[newKey]) {
                            newKey = `newProperty${counter++}`;
                          }
                          updateArrayItems({
                            ...property.items!,
                            properties: { ...props, [newKey]: { type: 'string' } },
                          });
                        }}
                        startIcon={<AddIcon />}
                      >
                        Add Property
                      </Button>
                    </Box>
                    {property.items?.properties && Object.keys(property.items.properties).length > 0 && (
                      <NestedProperties>
                        {Object.entries(property.items.properties).map(([key, value]) => (
                          <PropertyEditor
                            key={key}
                            name={key}
                            property={value}
                            isRequired={(property.items?.required || []).includes(key)}
                            onUpdate={(pName, prop, newName) => {
                              const props = { ...property.items?.properties };
                              if (newName && newName !== pName) {
                                delete props[pName];
                                props[newName] = prop;
                                const req = (property.items?.required || []).map(r => r === pName ? newName : r);
                                updateArrayItems({ ...property.items!, properties: props, required: req });
                              } else {
                                props[pName] = prop;
                                updateArrayItems({ ...property.items!, properties: props });
                              }
                            }}
                            onDelete={(pName) => {
                              const props = { ...property.items?.properties };
                              delete props[pName];
                              const req = (property.items?.required || []).filter(r => r !== pName);
                              updateArrayItems({ ...property.items!, properties: props, required: req });
                            }}
                            onToggleRequired={(pName) => {
                              const req = property.items?.required || [];
                              if (req.includes(pName)) {
                                updateArrayItems({ ...property.items!, required: req.filter(r => r !== pName) });
                              } else {
                                updateArrayItems({ ...property.items!, required: [...req, pName] });
                              }
                            }}
                            depth={depth + 1}
                          />
                        ))}
                      </NestedProperties>
                    )}
                  </Box>
                )}
              </Box>
            </Box>
          )}
        </PropertyContent>
      </Collapse>
    </PropertyCard>
  );
};

interface SchemaBuilderProps {
  initialData: CreateSchemaRequest;
  onSubmit: (data: CreateSchemaRequest) => Promise<void>;
  onCancel: () => void;
  isEditing: boolean;
}

export const SchemaBuilder: React.FC<SchemaBuilderProps> = ({
  initialData,
  onSubmit,
  onCancel,
  isEditing,
}) => {
  const [name, setName] = useState(initialData.name);
  const [description, setDescription] = useState(initialData.description || '');
  const [schema, setSchema] = useState<JSONSchemaDefinition>(initialData.schema);
  const [showPreview, setShowPreview] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const addProperty = useCallback(() => {
    const props = schema.properties || {};
    let newKey = 'newProperty';
    let counter = 1;
    while (props[newKey]) {
      newKey = `newProperty${counter++}`;
    }
    setSchema({
      ...schema,
      properties: { ...props, [newKey]: { type: 'string' } },
    });
  }, [schema]);

  const updateProperty = useCallback((propName: string, property: JSONSchemaProperty, newName?: string) => {
    const props = { ...schema.properties };
    if (newName && newName !== propName) {
      delete props[propName];
      props[newName] = property;
      // Update required array if needed
      const req = (schema.required || []).map(r => r === propName ? newName : r);
      setSchema({ ...schema, properties: props, required: req });
    } else {
      props[propName] = property;
      setSchema({ ...schema, properties: props });
    }
  }, [schema]);

  const deleteProperty = useCallback((propName: string) => {
    const props = { ...schema.properties };
    delete props[propName];
    const req = (schema.required || []).filter(r => r !== propName);
    setSchema({ ...schema, properties: props, required: req });
  }, [schema]);

  const toggleRequired = useCallback((propName: string) => {
    const req = schema.required || [];
    if (req.includes(propName)) {
      setSchema({ ...schema, required: req.filter(r => r !== propName) });
    } else {
      setSchema({ ...schema, required: [...req, propName] });
    }
  }, [schema]);

  const handleSubmit = async () => {
    if (!name.trim()) {
      setError('Schema name is required');
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      // Ensure all object types have additionalProperties: false for strict schema validation
      const strictSchema = ensureStrictObjects(schema) as JSONSchemaDefinition;
      
      await onSubmit({
        name: name.trim(),
        description: description.trim() || undefined,
        schema: strictSchema,
        source_type: 'manual',
      });
    } catch (err: any) {
      setError(err.message || 'Failed to save schema');
    } finally {
      setSubmitting(false);
    }
  };

  // Get the strict schema for display and copying
  const strictSchema = ensureStrictObjects(schema);

  const copyToClipboard = () => {
    navigator.clipboard.writeText(JSON.stringify(strictSchema, null, 2));
  };

  const formatJsonForDisplay = (obj: any): string => {
    return JSON.stringify(obj, null, 2);
  };

  return (
    <BuilderContainer>
      <Box display="flex" gap={2}>
        <InputGroup sx={{ flex: 1 }}>
          <InputLabel>Schema Name *</InputLabel>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., UserProfile, OrderResponse"
            error={!!error && !name.trim()}
            helperText={error && !name.trim() ? error : undefined}
            fullWidth
          />
        </InputGroup>
      </Box>

      <InputGroup>
        <InputLabel>Description</InputLabel>
        <Input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Describe what this schema is used for"
          multiline
          rows={2}
          fullWidth
        />
      </InputGroup>

      <Box>
        <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
          <Typography variant="h6" fontSize={16} fontWeight={600}>
            Schema Properties
          </Typography>
          <Box display="flex" gap={1}>
            <Button
              size="small"
              variant={showPreview ? 'contained' : 'outlined'}
              onClick={() => setShowPreview(!showPreview)}
              startIcon={<CodeIcon />}
            >
              {showPreview ? 'Hide JSON' : 'Show JSON'}
            </Button>
            <Button
              size="small"
              onClick={addProperty}
              startIcon={<AddIcon />}
              variant="contained"
            >
              Add Property
            </Button>
          </Box>
        </Box>

        {showPreview && (
          <Box mb={2}>
            <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
              <Typography variant="subtitle2" color="text.secondary">
                JSON Schema Preview
              </Typography>
              <Tooltip title="Copy to clipboard">
                <IconButton size="small" onClick={copyToClipboard}>
                  <ContentCopyIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Box>
            <CodePreview>
              <pre style={{ margin: 0 }}>{formatJsonForDisplay(strictSchema)}</pre>
            </CodePreview>
          </Box>
        )}

        <Box display="flex" flexDirection="column" gap={1.5}>
          {schema.properties && Object.keys(schema.properties).length > 0 ? (
            Object.entries(schema.properties).map(([key, value]) => (
              <PropertyEditor
                key={key}
                name={key}
                property={value}
                isRequired={(schema.required || []).includes(key)}
                onUpdate={updateProperty}
                onDelete={deleteProperty}
                onToggleRequired={toggleRequired}
              />
            ))
          ) : (
            <Box
              sx={{
                p: 4,
                textAlign: 'center',
                border: '2px dashed',
                borderColor: 'divider',
                borderRadius: 1,
              }}
            >
              <Typography color="text.secondary" mb={2}>
                No properties defined yet. Add your first property to start building the schema.
              </Typography>
              <Button onClick={addProperty} startIcon={<AddIcon />} variant="outlined">
                Add First Property
              </Button>
            </Box>
          )}
        </Box>
      </Box>

      {error && (
        <Typography color="error" variant="body2">
          {error}
        </Typography>
      )}

      <ActionBar>
        <Button onClick={onCancel} color="inherit" disabled={submitting}>
          Cancel
        </Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          color="primary"
          disabled={submitting}
        >
          {submitting ? 'Saving...' : isEditing ? 'Update Schema' : 'Create Schema'}
        </Button>
      </ActionBar>
    </BuilderContainer>
  );
};

