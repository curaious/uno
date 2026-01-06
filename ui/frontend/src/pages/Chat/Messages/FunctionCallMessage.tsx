import React, { useState } from 'react';
import {Accordion as MuiAccordion, AccordionDetails, Tabs, Tab, Typography, accordionSummaryClasses, AccordionProps} from "@mui/material";
import {Accordion, AccordionSummary} from "./Commons";
import PsychologyIcon from '@mui/icons-material/Psychology';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import BuildIcon from '@mui/icons-material/Build';
import {Prism as SyntaxHighlighter} from 'react-syntax-highlighter';
import {a11yDark} from 'react-syntax-highlighter/dist/esm/styles/prism';
import { FunctionCallMessage } from "@praveen001/uno-converse";

interface IProps {
  message: FunctionCallMessage;
}

export const FunctionCallMessageRenderer: React.FC<IProps> = props => {
  const {message} = props;

  const [value, setValue] = useState(0);

  const handleChange = (event: React.SyntheticEvent, newValue: number) => {
    setValue(newValue);
  };

  if (message.type !== "function_call") {
    return null;
  }

  return (
    <Accordion
      TransitionProps={{timeout: 100}}
    >
      <AccordionSummary expandIcon={<ExpandMoreIcon />}><BuildIcon fontSize='small' style={{ fontSize: 14 }} />
        <span style={{ whiteSpace: 'nowrap' }}>
          {message.name.replaceAll("_", " ")}
        </span>
      </AccordionSummary>

      <AccordionDetails>
        <Tabs value={value} onChange={handleChange}>
          <Tab label="Request" value={0} />
          <Tab label="Response" value={1} />
        </Tabs>

        {value === 0 && (
          <SyntaxHighlighter language="json" style={a11yDark} customStyle={{fontSize: '12px'}}>
            {JSON.stringify(message.arguments, null, 2)}
          </SyntaxHighlighter>
        )}

        <div id={message.id} style={value === 0 ? { position: 'absolute', pointerEvents: 'none', visibility :'hidden' } : { } }></div>

      </AccordionDetails>
    </Accordion>
  );
}