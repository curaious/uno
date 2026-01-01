import React, { useState } from 'react';
import {styled, Tooltip, Popover, Box, Typography, Divider} from '@mui/material';
import {useNavigate} from 'react-router';
import ApiIcon from '@mui/icons-material/Api';
import LogoutIcon from '@mui/icons-material/Logout';
import {useAppContext, AppType} from '../../contexts/AppContext';
import {useAuth} from '../../contexts/AuthContext';

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

const Spacer = styled('div')({
  flex: 1,
});

const UserAvatar = styled('button')(({ theme }) => ({
  width: 36,
  height: 36,
  borderRadius: '50%',
  border: '2px solid rgba(255, 255, 255, 0.15)',
  background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
  color: '#fff',
  cursor: 'pointer',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  fontSize: 14,
  fontWeight: 600,
  textTransform: 'uppercase',
  transition: 'all 0.2s ease',
  marginBottom: 12,
  '&:hover': {
    border: '2px solid rgba(255, 255, 255, 0.4)',
    transform: 'scale(1.05)',
  },
}));

const UserPopover = styled(Popover)(({ theme }) => ({
  '& .MuiPaper-root': {
    background: 'oklch(18% .006 285.885)',
    border: '1px solid rgba(255, 255, 255, 0.1)',
    borderRadius: 12,
    padding: 0,
    minWidth: 220,
    boxShadow: '0 8px 32px rgba(0, 0, 0, 0.4)',
  },
}));

const UserInfo = styled(Box)(({ theme }) => ({
  padding: '16px',
  display: 'flex',
  flexDirection: 'column',
  gap: 4,
}));

const UserName = styled(Typography)({
  color: '#fff',
  fontSize: 14,
  fontWeight: 600,
  lineHeight: 1.3,
});

const UserEmail = styled(Typography)({
  color: 'rgba(255, 255, 255, 0.5)',
  fontSize: 12,
  lineHeight: 1.3,
});

const UserRole = styled(Box)({
  marginTop: 4,
  display: 'inline-flex',
  alignItems: 'center',
  background: 'rgba(102, 126, 234, 0.15)',
  color: '#667eea',
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.5px',
  padding: '3px 8px',
  borderRadius: 4,
  width: 'fit-content',
});

const LogoutButton = styled('button')(({ theme }) => ({
  width: '100%',
  padding: '12px 16px',
  border: 'none',
  background: 'transparent',
  color: 'rgba(255, 255, 255, 0.7)',
  cursor: 'pointer',
  display: 'flex',
  alignItems: 'center',
  gap: 10,
  fontSize: 13,
  transition: 'all 0.2s ease',
  '&:hover': {
    background: 'rgba(239, 68, 68, 0.1)',
    color: '#ef4444',
  },
  '& svg': {
    fontSize: 18,
  },
}));

export const MiniSidebar: React.FC = () => {
  const { selectedApp, setSelectedApp } = useAppContext();
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null);

  const handleAppSwitch = (app: AppType) => {
    setSelectedApp(app);
    // Navigate to default route for the selected app
    if (app === 'llm-gateway') {
      navigate('/gateway/providers');
    } else {
      navigate('/agent-framework/projects');
    }
  };

  const handleUserClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handlePopoverClose = () => {
    setAnchorEl(null);
  };

  const handleLogout = () => {
    handlePopoverClose();
    logout();
  };

  const getInitials = (name: string) => {
    return name
      .split(' ')
      .map(part => part[0])
      .join('')
      .slice(0, 2);
  };

  const open = Boolean(anchorEl);

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
      
      <Spacer />
      
      {user && (
        <>
          <Tooltip title={user.name} placement="right" arrow>
            <UserAvatar onClick={handleUserClick} aria-label="User menu">
              {getInitials(user.name)}
            </UserAvatar>
          </Tooltip>
          <UserPopover
            open={open}
            anchorEl={anchorEl}
            onClose={handlePopoverClose}
            anchorOrigin={{
              vertical: 'bottom',
              horizontal: 'right',
            }}
            transformOrigin={{
              vertical: 'bottom',
              horizontal: 'left',
            }}
          >
            <UserInfo>
              <UserName>{user.name}</UserName>
              <UserEmail>{user.email}</UserEmail>
              {user.role && <UserRole>{user.role}</UserRole>}
            </UserInfo>
            <Divider sx={{ borderColor: 'rgba(255, 255, 255, 0.08)' }} />
            <LogoutButton onClick={handleLogout}>
              <LogoutIcon />
              Sign out
            </LogoutButton>
          </UserPopover>
        </>
      )}
    </MiniSidebarContainer>
  );
};

