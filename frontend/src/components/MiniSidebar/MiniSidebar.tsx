import React from 'react';
import {styled, Tooltip} from '@mui/material';
import {useNavigate} from 'react-router';
import ApiIcon from '@mui/icons-material/Api';
import {useAppContext, AppType} from '../../contexts/AppContext';

// Custom Agent Framework Icon - Robot/Agent with Framework Structure
const AgentFrameworkIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.5"
    strokeLinecap="round"
    strokeLinejoin="round"
    xmlns="http://www.w3.org/2000/svg"
    style={{ width: '24px', height: '24px' }}
  >
    {/* Robot head */}
    <rect x="8" y="6" width="8" height="8" rx="1.5" fill="currentColor" opacity="0.2" />
    <rect x="8" y="6" width="8" height="8" rx="1.5" />
    {/* Eyes */}
    <circle cx="10.5" cy="9" r="1" fill="currentColor" />
    <circle cx="13.5" cy="9" r="1" fill="currentColor" />
    {/* Antenna/connection point */}
    <line x1="12" y1="6" x2="12" y2="4" />
    <circle cx="12" cy="3.5" r="1" fill="currentColor" />
    {/* Body/chest */}
    <rect x="9" y="14" width="6" height="4" rx="0.5" fill="currentColor" opacity="0.2" />
    <rect x="9" y="14" width="6" height="4" rx="0.5" />
    {/* Framework connections - nodes around the robot */}
    <circle cx="5" cy="10" r="1" fill="currentColor" />
    <circle cx="19" cy="10" r="1" fill="currentColor" />
    <circle cx="5" cy="16" r="1" fill="currentColor" />
    <circle cx="19" cy="16" r="1" fill="currentColor" />
    {/* Connection lines to framework nodes */}
    <line x1="8" y1="10" x2="6" y2="10" strokeWidth="1" />
    <line x1="16" y1="10" x2="18" y2="10" strokeWidth="1" />
    <line x1="9" y1="16" x2="6" y2="16" strokeWidth="1" />
    <line x1="15" y1="16" x2="18" y2="16" strokeWidth="1" />
  </svg>
);

const MiniSidebarContainer = styled('div')(() => ({
  width: 48,
  minWidth: 48,
  height: '100vh',
  background: 'oklch(21% .006 285.885)',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  paddingTop: 16,
  gap: 8,
  borderRight: '1px solid rgba(255, 255, 255, 0.1)',
}));

const AppIconButton = styled('button')<{ active: boolean }>(({ theme, active }) => ({
  width: 40,
  height: 40,
  borderRadius: theme.shape.borderRadius,
  border: 'none',
  background: active ? theme.palette.action.active : 'transparent',
  color: active ? '#fff' : theme.palette.text.secondary,
  cursor: 'pointer',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  transition: 'all 0.2s ease',
  '&:hover': {
    background: active ? theme.palette.action.active : theme.palette.action.hover,
    color: '#fff',
  },
  '& svg': {
    fontSize: 24,
  },
}));

export const MiniSidebar: React.FC = () => {
  const { selectedApp, setSelectedApp } = useAppContext();
  const navigate = useNavigate();

  const handleAppSwitch = (app: AppType) => {
    setSelectedApp(app);
    // Navigate to default route for the selected app
    if (app === 'llm-gateway') {
      navigate('/gateway/providers');
    } else {
      navigate('/agent-framework/projects');
    }
  };

  return (
    <MiniSidebarContainer>
      <Tooltip title="LLM Gateway" placement="right" arrow>
        <AppIconButton
          active={selectedApp === 'llm-gateway'}
          onClick={() => handleAppSwitch('llm-gateway')}
          aria-label="LLM Gateway"
        >
          <ApiIcon />
        </AppIconButton>
      </Tooltip>
      <Tooltip title="Agent Framework" placement="right" arrow>
        <AppIconButton
          active={selectedApp === 'agent-framework'}
          onClick={() => handleAppSwitch('agent-framework')}
          aria-label="Agent Framework"
        >
          <AgentFrameworkIcon />
        </AppIconButton>
      </Tooltip>
    </MiniSidebarContainer>
  );
};

