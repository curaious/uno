import styled from "@emotion/styled";
import {
  Accordion as MuiAccordion,
  AccordionProps,
  AccordionSummary as MuiAccordionSummary, accordionSummaryClasses,
  AccordionSummaryProps
} from "@mui/material";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import React from "react";

export const Accordion = styled((props: AccordionProps) => <MuiAccordion {...props} />)(() => ({
  background: 'transparent',
  outline: 'none',
  border: 'none',
  boxShadow: 'none',
  padding: 0,
}));

export const AccordionSummary = styled((props: AccordionSummaryProps) => (
  <MuiAccordionSummary
    expandIcon={<ExpandMoreIcon sx={{ fontSize: '0.9rem' }} />}
    {...props}
  />
))(({ theme }) => ({
  backgroundColor: 'transparent',
  justifyContent: 'flex-start',
  gap: 4,
  margin: 0, padding: 0,
  [`& .${accordionSummaryClasses.content}`]: {
    margin: 0, padding: 0, opacity: 0.6, flex: 0,
    display: 'flex',
    alignItems: 'center',
    gap: 4,
    textTransform: 'capitalize',
  },
  [`&.${accordionSummaryClasses.expanded}`]: {
    minHeight: '48px',
  }
}));