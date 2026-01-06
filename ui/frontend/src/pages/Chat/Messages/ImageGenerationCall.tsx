import React, { useState } from 'react';
import {Accordion as MuiAccordion, AccordionDetails, Tabs, Tab, Typography, accordionSummaryClasses, AccordionProps} from "@mui/material";
import {Accordion, AccordionSummary} from "./Commons";
import PsychologyIcon from '@mui/icons-material/Psychology';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import BuildIcon from '@mui/icons-material/Build';
import {Prism as SyntaxHighlighter} from 'react-syntax-highlighter';
import {a11yDark} from 'react-syntax-highlighter/dist/esm/styles/prism';
import { FunctionCallMessage, ImageGenerationCallMessage } from "@curaious/uno-converse";

interface IProps {
  message: ImageGenerationCallMessage;
}

export const ImageGenerationCallRenderer: React.FC<IProps> = props => {
  const {message} = props;

  const [value, setValue] = useState(0);

  const handleChange = (event: React.SyntheticEvent, newValue: number) => {
    setValue(newValue);
  };

  if (message.type !== "image_generation_call") {
    return null;
  }

  return (
    <img src={`data:image/${message.output_format};base64,${message.result}`} />
  );
}