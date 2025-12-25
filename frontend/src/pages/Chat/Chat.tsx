import React, {useMemo, useState} from 'react';

import styles from './Chat.module.css';
import {Box, CircularProgress, Typography} from "@mui/material";
import {Turn} from "./Turn";
import {
  ContentType,
  ConversationMessage, InputMessage, MessageType, MessageUnion, Role, Usage,
} from "../../lib/converse/types/types";
import {v4 as uuidv4} from "uuid";

interface IOwnProps {
  messages: ConversationMessage[];
  contextWindow?: Usage;
  isStreaming: boolean;
  onUserMessage: (userMessages: MessageUnion[]) => void;
  children: React.ReactNode;
}

export const Chat: React.FC<IOwnProps> = props => {
  const {messages, contextWindow, isStreaming, onUserMessage, children} = props;

  const [userTextMessage, setUserTextMessage] = useState('');

  const MAX_CONTEXT_WINDOW_SIZE = 200000;

  const currentContextWindowSize = useMemo(() => Math.trunc(((contextWindow?.input_tokens || 0) / MAX_CONTEXT_WINDOW_SIZE) * 100 * 100) / 100, [contextWindow])

  const onSubmit = () => {
    // Create the user message
    const userMessage: InputMessage = {
      id: `msg_` + uuidv4(),
      type: MessageType.Message,
      role: Role.User,
      content: [
        {
          type: ContentType.InputText,
          text: userTextMessage,
        }
      ],
    };
    onUserMessage([userMessage]);
    setUserTextMessage('');
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      onSubmit();
    }
  };

  return <div className={styles.root}>
    <div className={styles.messageContainer}>
      {messages.map(((m, idx) => <Turn message={m} onUserMessage={onUserMessage} key={m.message_id} completed={!isStreaming || idx+1 < messages.length}/>))}
    </div>

    <div className={styles.composer}>
      <form onSubmit={onSubmit} className={styles.textInputContainer}>
        <textarea
          onKeyDown={handleKeyDown}
          autoComplete="off"
          placeholder="Ask anything"
          value={userTextMessage}
          onChange={e => setUserTextMessage(e.target.value)}
          autoFocus
        />

        <Box display="flex" justifyContent="space-between" alignItems="center" style={{width: '100%'}}>
          {children}
          <Box display="flex" gap="8px" alignItems="center">
            {!Number.isNaN(currentContextWindowSize) && <Box display="flex" gap="4px" alignItems="center" style={{
              backgroundColor: 'rgb(34 34 34 / 45%)',
              padding: 4,
              borderRadius: 4
            }}>
                <Typography style={{fontSize: 11}} variant="caption" color="textSecondary">
                  {currentContextWindowSize}%</Typography>
                <CircularProgress variant="determinate" value={currentContextWindowSize} size={15}/>
            </Box>}

            <button type="submit" className={styles.sendBtn} onClick={() => onSubmit()}>
              <svg width="20" height="20" viewBox="0 0 24 18" fill="currentColor" xmlns="http://www.w3.org/2000/svg"
                   className="icon">
                <path
                  d="M8.99992 16V6.41407L5.70696 9.70704C5.31643 10.0976 4.68342 10.0976 4.29289 9.70704C3.90237 9.31652 3.90237 8.6835 4.29289 8.29298L9.29289 3.29298L9.36907 3.22462C9.76184 2.90427 10.3408 2.92686 10.707 3.29298L15.707 8.29298L15.7753 8.36915C16.0957 8.76192 16.0731 9.34092 15.707 9.70704C15.3408 10.0732 14.7618 10.0958 14.3691 9.7754L14.2929 9.70704L10.9999 6.41407V16C10.9999 16.5523 10.5522 17 9.99992 17C9.44764 17 8.99992 16.5523 8.99992 16Z"></path>
              </svg>
            </button>
          </Box>
        </Box>

      </form>
    </div>

  </div>;
}