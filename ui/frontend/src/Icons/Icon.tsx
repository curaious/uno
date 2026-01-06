import React from 'react';
import {styled} from "@mui/material";
import classNames from "classnames";

export const IconRoot = styled('div')(({ theme }) => ({
  marginRight: theme.spacing(0.5),
  color: theme.palette.text.secondary,

  height: 24,
  width: 24,
  fontSize: 16,
  '& > svg': {
    fontSize: 18,
  },

  '&.large': {
    height: 32,
    width: 32,
    fontSize: 20,
  },

  '&.small': {
    height: 16,
    width: 16,
    fontSize: 13,
    '& > svg': {
      fontSize: 16,
    },
  }
}));

export enum IconSize {
  Small = 'small',
  Medium = 'medium',
  Large = 'large',
}

export const Icon: React.FC<{ children: React.ReactNode; size?: IconSize }> = (props) => {
  const { size } = props;

  return <IconRoot className={classNames(size)}>
    {props.children}
  </IconRoot>
};

export const ProviderIcon: React.FC = props => {
  return <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960" fill="#e3e3e3"><path d="M440-183v-274L200-596v274l240 139Zm80 0 240-139v-274L520-457v274Zm-40-343 237-137-237-137-237 137 237 137ZM160-252q-19-11-29.5-29T120-321v-318q0-22 10.5-40t29.5-29l280-161q19-11 40-11t40 11l280 161q19 11 29.5 29t10.5 40v318q0 22-10.5 40T800-252L520-91q-19 11-40 11t-40-11L160-252Zm320-228Z"/></svg>
}