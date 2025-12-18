import React, {ReactNode} from 'react';
import {styled, Box, IconButton} from "@mui/material";
import Close from '@mui/icons-material/Close';

export const DialogContent = styled('div')(() => ({
  backgroundColor: 'var(--background-elevated)',
  display: 'flex',
  flexDirection: 'column',
  overflow: 'hidden',
  padding: '0 16px',
}));

export const DialogHeader = styled('div')(() => ({
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
  padding: '0 16px',
  borderBottom: '1px solid var(--border-color)',
  backgroundColor: 'var(--background-elevated)',
}));

// Backdrop for slide dialog
const SlideDialogBackdrop = styled(Box)<{ open: boolean }>(({ open }) => ({
  position: 'fixed',
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  backgroundColor: 'rgba(0, 0, 0, 0.5)',
  zIndex: 1400,
  opacity: open ? 1 : 0,
  visibility: open ? 'visible' : 'hidden',
  transition: 'opacity 0.3s ease, visibility 0.3s ease',
  pointerEvents: open ? 'auto' : 'none',
}));

// Slide dialog container
const SlideDialogContainer = styled(Box)<{ open: boolean; width?: string }>(({ open, width = '500px' }) => ({
  position: 'fixed',
  top: 0,
  right: 0,
  bottom: 0,
  width: width,
  maxWidth: '90vw',
  backgroundColor: 'var(--background-elevated)',
  zIndex: 1401,
  display: 'flex',
  flexDirection: 'column',
  boxShadow: '-2px 0 8px rgba(0, 0, 0, 0.15)',
  transform: open ? 'translateX(0)' : 'translateX(100%)',
  transition: 'transform 0.3s ease-in-out',
  // Ensure Select/Menu dropdowns appear above the dialog
  '& .MuiPopover-root': {
    zIndex: '1500 !important',
  },
  '& .MuiMenu-root': {
    zIndex: '1500 !important',
  },
}));

// Slide dialog header
const SlideDialogHeader = styled(Box)(() => ({
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
  padding: '16px 24px',
  borderBottom: '1px solid var(--border-color)',
  minHeight: '64px',
  '& h2': {
    margin: 0,
    fontSize: '1.25rem',
    fontWeight: 600,
    color: 'var(--text-primary)',
  },
}));

// Slide dialog content
const SlideDialogContent = styled(Box)(() => ({
  flex: 1,
  overflowY: 'auto',
  padding: '24px',
  backgroundColor: 'var(--background-elevated)',
}));

// Slide dialog actions
const SlideDialogActions = styled(Box)(() => ({
  display: 'flex',
  justifyContent: 'flex-end',
  gap: '8px',
  padding: '16px 24px',
  borderTop: '1px solid var(--border-color)',
  backgroundColor: 'var(--background-elevated)',
}));

export interface SlideDialogProps {
  open: boolean;
  onClose: () => void;
  title: string | ReactNode;
  children: ReactNode;
  actions?: ReactNode;
  width?: string;
  maxWidth?: string;
}

export const SlideDialog: React.FC<SlideDialogProps> = ({
  open,
  onClose,
  title,
  children,
  actions,
  width = '500px',
  maxWidth,
}) => {
  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return (
    <>
      <SlideDialogBackdrop open={open} onClick={handleBackdropClick} />
      <SlideDialogContainer open={open} width={maxWidth || width}>
        <SlideDialogHeader>
          <Box component="h2">{title}</Box>
          <IconButton onClick={onClose} size="small" aria-label="close">
            <Close />
          </IconButton>
        </SlideDialogHeader>
        <SlideDialogContent>{children}</SlideDialogContent>
        {actions && <SlideDialogActions>{actions}</SlideDialogActions>}
      </SlideDialogContainer>
    </>
  );
};