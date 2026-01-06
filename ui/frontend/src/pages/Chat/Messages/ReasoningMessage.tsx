import React from 'react';
import production from 'react/jsx-runtime';
import remarkParse from 'remark-parse'
import remarkRehype from 'remark-rehype'
import rehypeHighlight from 'rehype-highlight';
import rehypeReact from "rehype-react";
import {unified} from 'unified';
import remarkGfm from 'remark-gfm';
import { AccordionDetails} from "@mui/material";
import {Accordion, AccordionSummary} from "./Commons";
import PsychologyIcon from '@mui/icons-material/Psychology';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { ReasoningMessage } from "@curaious/uno-converse";

interface IProps {
  message: ReasoningMessage;
}

export const ReasoningMessageRenderer: React.FC<IProps> = props => {
  const {message} = props;

  if (message.type !== "reasoning") {
    return null;
  }

  const last = true;

  const text = unified()
    .use(remarkParse)
    .use(remarkGfm)
    .use(remarkRehype as any)
    .use(rehypeHighlight as any)
    .use(rehypeReact, {
      ...production as any, components: {}
    }).processSync(message.summary?.map(s => s.text).join("")).result;

    return (
      <Accordion
        TransitionProps={{timeout: 100}}
        style={{ background: 'transparent', outline: 'none', border: 'none', boxShadow: 'none', padding: 0}}
        // expanded={last || toolCallsOpen.includes(content.reasoning?.id!)}
        // onChange={(_, expanded) => onAccordionToggle(expanded, content.reasoning?.id!)}
      >
        <AccordionSummary expandIcon={<ExpandMoreIcon />}><PsychologyIcon fontSize='small' />{last ? 'Thinking' : 'Thoughts'}</AccordionSummary>
        <AccordionDetails style={{ fontSize: 14, padding: 0, opacity: 0.6 }}>{text}</AccordionDetails>
      </Accordion>
    );
}