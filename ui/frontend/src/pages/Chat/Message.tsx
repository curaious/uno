import React from 'react';
import classnames from 'classnames';
import styles from './Message.module.css';
import "highlight.js/styles/a11y-dark.css";
import {
  Accordion as MuiAccordion,
  AccordionProps,
  AccordionSummary as MuiAccordionSummary,
  accordionSummaryClasses,
  AccordionSummaryProps
} from "@mui/material";
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import styled from '@emotion/styled';
import {InputMessageRenderer} from "./Messages/InputMessage";
import {ReasoningMessageRenderer} from "./Messages/ReasoningMessage";
import {FunctionCallMessageRenderer} from "./Messages/FunctionCallMessage";
import {FunctionCallOutputMessageRenderer} from "./Messages/FunctionCallOutputMessage";
import { MessageType, MessageUnion, Role } from "@praveen001/uno-converse";
import {ImageGenerationCallRenderer} from "./Messages/ImageGenerationCall";

const Accordion = styled((props: AccordionProps) => <MuiAccordion {...props} />)(() => ({
  background: 'transparent',
  outline: 'none',
  border: 'none',
  boxShadow: 'none',
  padding: 0,
}));

const AccordionSummary = styled((props: AccordionSummaryProps) => (
  <MuiAccordionSummary
    expandIcon={<ExpandMoreIcon sx={{ fontSize: '0.9rem' }} />}
    {...props}
  />
))(({ theme }) => ({
  backgroundColor: 'rgba(0, 0, 0, .03)',
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

interface IOwnProps {
  message: MessageUnion;
}

export const MessageRenderer: React.FC<IOwnProps> = props => {
  const {message} = props;

  const getContent = (message: MessageUnion) => {
    if (message.type === "function_call") {
      return <FunctionCallMessageRenderer message={message} />
    }

    if (message.type === "function_call_output") {
      return <FunctionCallOutputMessageRenderer message={message} />
    }

    if (message.type === "reasoning") {
      return <ReasoningMessageRenderer message={message} />
    }

    if (message.type === MessageType.Message) {
      return <InputMessageRenderer message={message} />;
    }

    if (message.type === MessageType.ImageGenerationCall) {
      return <ImageGenerationCallRenderer message={message} />;
    }
  }

  let role: Role = Role.Assistant
  if (message.type === "message") {
    role = message.role!;
  }

  return <div className={classnames(styles.message, {
    [styles.right]: role === "user"
  })}>
    <div className={classnames({
      [styles.userMessage]: role === "user",
      [styles.assistantMessage]: role === "assistant"
    })}>
      {<div key={message.id}>{getContent(message)}</div>}
    </div>
  </div>
}