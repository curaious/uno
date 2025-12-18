import {styled} from "@mui/material";

export const PageContainer = styled('div')(({theme}) => ({
  display: 'flex',
  flexDirection: 'column',
  height: '100%',
  backgroundColor: 'oklch(21% .006 285.885)',
  border: '1px solid oklch(27.4% .006 286.033)',
  borderRadius: 6,
  color: theme.palette.text.primary,
  padding: '16px',
  boxSizing: 'border-box',
  flex: 1
}));

export const PageHeader = styled('div')(({theme}) => ({
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  marginBottom: '12px',
  padding: '0px 0px 8px 0',

  '&>div:first-child': {
    flex: 1
  }
}));

export const PageTitle = styled('h1')(({theme}) => ({
  fontSize: '22px',
  fontWeight: 500,
  margin: '0',
  color: 'var(--text-default)'
}));

export const PageSubtitle = styled('p')(({theme}) => ({
  fontSize: '16px',
  color: 'var(--text-secondary)',
  margin: 0
}))