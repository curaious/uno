import {
  styled,
  TextField as _Input,
  InputLabel as _InputLabel,
  Select as _Select,
  SelectProps,
  BaseSelectProps
} from "@mui/material";
import {jsx} from "@emotion/react";
import IntrinsicElements = jsx.JSX.IntrinsicElements;

export const Input = styled(_Input)(() => ({
  width: '100%',
  borderRadius: '8px',
  color: 'var(--text-default)',
  transition: 'border-color 0.2s ease',
  boxSizing: 'border-box',
  minHeight: 36,
  '& input': {
    fontSize: '13px',
    padding: '6px 12px',
  },
  '&:focus': {
    outline: 'none',
    borderColor: '#10a37f',
    boxShadow: '0 0 0 2px rgba(16, 163, 127, 0.1)'
  },
  '&::placeholder': {
    color: 'var(--text-secondary)'
  },
  '&.inputError': {
    borderColor: '#c53030'
  },
  '&.inputError:focus': {
    borderColor: '#c53030',
    boxShadow: '0 0 0 3px rgba(197, 48, 48, 0.1)',
  }
}));

export const InputGroup = styled('div')(() => ({
  marginBottom: '24px;'
}));

export const InputLabel = styled(_InputLabel)(() => ({
  color: '#fff',
  fontSize: 13,
  fontWeight: 500,
}));

export const Select = styled(_Select)<SelectProps<any>>(() => ({
  '& > .MuiInputBase-input': {
    minHeight: 36,
    borderRadius: '8px',
    color: 'var(--text-default)',
    transition: 'border-color 0.2s ease',
    boxSizing: 'border-box',
    padding: '6px 12px',
    fontSize: '13px',
  },
  '& input': {
    fontSize: '13px',
    padding: '6px 12px',
    backgroundColor: 'var(--input-field-default)',
  },
  '&:focus': {
    outline: 'none',
    borderColor: '#10a37f',
    boxShadow: '0 0 0 2px rgba(16, 163, 127, 0.1)'
  },
  '&::placeholder': {
    color: 'var(--text-secondary)'
  },
  '&.inputError': {
    borderColor: '#c53030'
  },
  '&.inputError:focus': {
    borderColor: '#c53030',
    boxShadow: '0 0 0 3px rgba(197, 48, 48, 0.1)',
  }
}));