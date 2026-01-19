import React, { useState } from 'react';
import {NavLink, useNavigate} from "react-router";
import {List as _List, styled, ListItemText as _ListItemText, Divider, MenuItem, Box} from "@mui/material";
import PsychologyIcon from '@mui/icons-material/Psychology';
import ConstructionIcon from '@mui/icons-material/Construction';
import ChatIcon from '@mui/icons-material/Chat';
import TextSnippetIcon from '@mui/icons-material/TextSnippet';
import StorageIcon from '@mui/icons-material/Storage';
import TimelineIcon from '@mui/icons-material/Timeline';
import ForumIcon from '@mui/icons-material/Forum';
import AddIcon from '@mui/icons-material/Add';
import ListIcon from '@mui/icons-material/List';
import {ProviderIcon} from "../../Icons/ProviderIcon";
import {Icon, IconSize} from "../../Icons/Icon";
import KeyIcon from '@mui/icons-material/Key';
import DataObjectIcon from '@mui/icons-material/DataObject';
import {useAppContext} from "../../contexts/AppContext";
import {useProjectContext} from "../../contexts/ProjectContext";
import {Select} from "../shared/Input";
import {SlideDialog} from "../shared/Dialog";
import {Input, InputGroup, InputLabel} from "../shared/Input";
import {Button} from "../shared/Buttons";
import TuneIcon from '@mui/icons-material/Tune';
import SmartToyIcon from '@mui/icons-material/SmartToy';


const SidebarDiv = styled('div')(() => ({
  width: 250,
  minWidth: 220,
  background: '#000',
  paddingTop: 8,
  display: 'flex',
  flexDirection: 'column',
}));

const ProjectSelectorContainer = styled(Box)(({ theme }) => ({
  padding: theme.spacing(1.5),
  paddingBottom: theme.spacing(1),
  borderBottom: `1px solid ${theme.palette.divider}`,
}));

const ProjectSelect = styled(Select)(({ theme }) => ({
  width: '100%',
  fontSize: '13px',
  '& .MuiSelect-select': {
    padding: '6px 12px',
    fontSize: '13px',
    display: 'flex',
    alignItems: 'center',
    lineHeight: '1.5',
  },
  '& .MuiInputBase-input': {
    display: 'flex',
    alignItems: 'center',
    padding: '6px 12px !important',
  },
  '& .MuiPaper-root': {
    '& .MuiMenuItem-root': {
      fontSize: '13px',
      minHeight: '30px',
      padding: '4px 12px',
      '& .MuiSvgIcon-root': {
        fontSize: '16px',
      },
    },
  },
}));

const StyledMenuItem = styled(MenuItem)(({ theme }) => ({
  fontSize: '13px !important',
  minHeight: '30px !important',
  padding: '4px 12px !important',
  '&:hover': {
    backgroundColor: theme.palette.action.hover,
  },
  '& .MuiSvgIcon-root': {
    fontSize: '16px',
    marginRight: theme.spacing(1),
  },
}));

const CreateProjectMenuItem = styled(StyledMenuItem)(({ theme }) => ({
  borderTop: `1px solid ${theme.palette.divider}`,
  marginTop: theme.spacing(0.5),
  paddingTop: theme.spacing(1),
}));

const ViewAllProjectsMenuItem = styled(StyledMenuItem)(({ theme }) => ({}));

const List = styled(_List)(({ theme }) => ({
  display: 'flex',
  flexDirection: 'column',
  gap: 6,
  paddingLeft: theme.spacing(1.5),
  paddingRight: theme.spacing(1.5),
  paddingTop: theme.spacing(1.5),
  width: '100%',
  '& > a': {
    margin: '0px',
    width: '100%',
    borderRadius: theme.shape.borderRadius,
    padding: '0px 8px',
    height: 30,
    display: 'flex',
    gap: 4,
    alignItems: 'center',
    textDecoration: 'none',
  },
  '& > a.active': {
    border: `0.5px solid ${theme.palette.divider}`,
    background: theme.palette.action.active,
    '& svg': {
      color: '#fff',
    },
    '& span': {
      color: '#fff'
    }
  },
  '& > a:hover': {
    background: theme.palette.action.hover,
  }
}));

const ListItemText = styled(_ListItemText)(({ theme }) => ({
  color: theme.palette.text.secondary,
  margin: 0,
  '& > span': {
    fontSize: 13,
  },
}));

const getErrorMessage = (error: any, fallback: string) => (
  error?.response?.data?.message ||
  error?.response?.data?.errorDetails?.error ||
  error?.message ||
  fallback
);

export const Sidebar: React.FC = props => {
  const { selectedApp } = useAppContext();
  const { projects, selectedProjectId, selectProject, createProject } = useProjectContext();
  const navigate = useNavigate();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [projectName, setProjectName] = useState('');
  const [defaultKey, setDefaultKey] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleCreateProjectClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    // Open dialog after a small delay to allow menu to close
    setTimeout(() => {
      setDialogOpen(true);
    }, 100);
  };

  const handleViewAllProjectsClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    setTimeout(() => {
      navigate('/agent-framework/projects');
    }, 100);
  };

  const closeDialog = () => {
    if (submitting) return;
    setDialogOpen(false);
    setFormError(null);
    setProjectName('');
    setDefaultKey('');
  };

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

  // LLM Gateway routes - only Providers and Virtual Keys
  const llmGatewayRoutes = (
    <List>
      <NavLink className={({ isActive }) => isActive ? 'active': ''}  to="/gateway/providers" end >
        <Icon size={IconSize.Small}><ProviderIcon /></Icon>
        <ListItemText primary="Providers" />
      </NavLink>
      <NavLink className={({ isActive }) => isActive ? 'active': ''}  to="/gateway/virtual-keys" end >
        <Icon size={IconSize.Small}><KeyIcon fontSize="small" /></Icon>
        <ListItemText primary="Virtual Keys" />
      </NavLink>
      <NavLink className={({ isActive }) => isActive ? 'active': ''}  to="/gateway/traces" end >
        <Icon size={IconSize.Small}><TimelineIcon fontSize="small" /></Icon>
        <ListItemText primary="Traces" />
      </NavLink>
    </List>
  );

  // Agent Framework routes - all other items
  const agentFrameworkRoutes = (
    <>
      <ProjectSelectorContainer>
        <ProjectSelect
          value={selectedProjectId || ''}
          onChange={(e) => {
            const projectId = e.target.value as string;
            if (projectId && projectId !== 'view-all') {
              selectProject(projectId);
            }
          }}
          displayEmpty
          MenuProps={{
            style: { zIndex: 1500 },
            PaperProps: {
              style: { 
                zIndex: 1500, 
                maxHeight: 300
              }
            }
          }}
        >
          <StyledMenuItem value="" disabled>
            Select Project
          </StyledMenuItem>
          {projects.map((project) => (
            <StyledMenuItem key={project.id} value={project.id}>
              {project.name}
            </StyledMenuItem>
          ))}
          <Divider sx={{ my: 0.5 }} />
          <ViewAllProjectsMenuItem 
            value="view-all"
            onClick={handleViewAllProjectsClick}
          >
            <ListIcon fontSize="small" />
            View All Projects
          </ViewAllProjectsMenuItem>
          <CreateProjectMenuItem 
            value=""
            onClick={handleCreateProjectClick}
            data-create-project="true"
          >
            <AddIcon fontSize="small" />
            Create New Project
          </CreateProjectMenuItem>
        </ProjectSelect>
      </ProjectSelectorContainer>
      <List>
        <NavLink className={({ isActive }) => isActive ? 'active': ''}  to="/agent-framework/prompts" end >
          <Icon size={IconSize.Small}><TextSnippetIcon fontSize="small" /></Icon>
          <ListItemText primary="Prompts" />
        </NavLink>

        <Divider />

        <NavLink className={({ isActive }) => isActive ? 'active': ''}  to="/agent-framework/agents" end >
          <Icon size={IconSize.Small}><PsychologyIcon fontSize="small" /></Icon>
          <ListItemText primary="Agents" />
        </NavLink>

        <NavLink className={({ isActive }) => isActive ? 'active': ''}  to="/agent-framework/builder" end >
          <Icon size={IconSize.Small}><ConstructionIcon fontSize="small" /></Icon>
          <ListItemText primary="Workflows" />
        </NavLink>

        <NavLink className={({ isActive }) => isActive ? 'active': ''}  to="/agent-framework/chat" end >
          <Icon size={IconSize.Small}><ChatIcon fontSize="small" /></Icon>
          <ListItemText primary="Chat" />
        </NavLink>

        <Divider />

        <NavLink className={({ isActive }) => isActive ? 'active': ''}  to="/agent-framework/traces" end >
          <Icon size={IconSize.Small}><TimelineIcon fontSize="small" /></Icon>
          <ListItemText primary="Traces" />
        </NavLink>

        <NavLink className={({ isActive }) => isActive ? 'active': ''}  to="/agent-framework/conversation-traces" end >
          <Icon size={IconSize.Small}><ForumIcon fontSize="small" /></Icon>
          <ListItemText primary="Conversation Traces" />
        </NavLink>
      </List>
    </>
  );

  return (
    <>
      <SidebarDiv>
        {selectedApp === 'llm-gateway' ? llmGatewayRoutes : agentFrameworkRoutes}
      </SidebarDiv>
      
      {selectedApp === 'agent-framework' && (
        <SlideDialog
          open={dialogOpen}
          onClose={closeDialog}
          title="Create Project"
          actions={
            <>
              <Button color="inherit" onClick={closeDialog} disabled={submitting}>
                Cancel
              </Button>
              <Button variant="contained" color="primary" type="submit" form="create-project-form" disabled={submitting}>
                {submitting ? 'Creating...' : 'Create'}
              </Button>
            </>
          }
        >
          <form id="create-project-form" onSubmit={handleSubmit}>
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
      )}
    </>
  );
}