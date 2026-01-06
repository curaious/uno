import {styled, ListItem as _ListItem, ListItemProps} from "@mui/material";
import {NavLinkProps} from "react-router";

export const ListItem = styled(_ListItem)<ListItemProps & NavLinkProps>(({ theme }) => ({
  display: 'flex',
  gap: '4px',
  color: theme.palette.text.primary,
  alignItems: 'center',
  '&:hover': {
    backgroundColor: 'var(--border-color)',
  },
}));

export const ListItemIcon = styled('div')(({ theme }) => ({
  minWidth: 0
}));