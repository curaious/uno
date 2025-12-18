import React, { useEffect, useState } from 'react';
import {Accordion as MuiAccordion, AccordionDetails, AccordionSummary as MuiAccordionSummary, AccordionSummaryProps, Box, CircularProgress, IconButton, Popover, Tooltip, Typography, accordionSummaryClasses, AccordionProps} from "@mui/material";
import {Prism as SyntaxHighlighter} from 'react-syntax-highlighter';
import {a11yDark} from 'react-syntax-highlighter/dist/esm/styles/prism';
import {FunctionCallOutputMessage} from "../../../lib/converse/types/types";
import ReactDOM from "react-dom";

interface IProps {
  message: FunctionCallOutputMessage;
}

export const FunctionCallOutputMessageRenderer: React.FC<IProps> = props => {
  const {message} = props;

  const [container, setContainer] = useState<HTMLElement | null>(null);

  useEffect(() => {
    // Run after mount/commit, when the DOM is ready
    const el = document.getElementById(message.id); // see #2 below about which id to use
    setContainer(el);
  }, [message.id]);

  if (message.type !== "function_call_output") {
    return null;
  }

  if (!container) {
    return null;
  }

  let output = ""
  if (typeof message.output === "string") {
    output = message.output
  } else if (Array.isArray(message.output) && message.output.length > 0) {
    output += message.output[0].text
  }

  try {
    const formattedJSON = JSON.stringify(JSON.parse(output as any), null, 2);
    return ReactDOM.createPortal(
        <SyntaxHighlighter language="json" style={a11yDark} customStyle={{fontSize: '12px'}}>
          {formattedJSON}
        </SyntaxHighlighter>
    , document.getElementById(message.id)!)
  } catch (e) {
      return ReactDOM.createPortal(
          <SyntaxHighlighter language="json" style={a11yDark} customStyle={{fontSize: '12px'}}>
            {output}
          </SyntaxHighlighter>
        , document.getElementById(message.id)!)
  }
}