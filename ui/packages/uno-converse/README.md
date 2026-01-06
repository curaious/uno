# @curaious/uno-converse

React hooks and utilities for building conversation UIs with [Uno Agent Server](https://github.com/curaious/uno).

## Features

- 🎣 **React Hook** - `useConversation` hook for complete conversation state management
- 🌊 **Streaming Support** - Built-in SSE streaming with real-time message updates
- 📦 **Type Safe** - Full TypeScript support with comprehensive type definitions
- 🔌 **Flexible API Client** - Bring your own API client or use the built-in one
- 🎯 **Zero Dependencies** - Only React as a peer dependency

## Installation

```bash
# npm
npm install @curaious/uno-converse

# yarn
yarn add @curaious/uno-converse

# pnpm
pnpm add @curaious/uno-converse
```

### GitHub Packages Registry

This package is published to GitHub Packages. You'll need to configure npm to use the GitHub registry for `@curaious` scoped packages:

```bash
# Create or edit ~/.npmrc
echo "@curaious:registry=https://npm.pkg.github.com" >> ~/.npmrc
```

You may also need to authenticate with a GitHub personal access token:

```bash
echo "//npm.pkg.github.com/:_authToken=YOUR_GITHUB_TOKEN" >> ~/.npmrc
```

## Quick Start

```tsx
import { useConversation, createApiClient, MessageType, ContentType } from '@curaious/uno-converse';

// Create an API client
const client = createApiClient({
  baseUrl: 'https://your-uno-server.com/api/agent-server',
  headers: {
    'Authorization': 'Bearer your-api-key',
  },
});

function ChatApp() {
  const {
    allMessages,
    isStreaming,
    sendMessage,
    startNewChat,
    conversations,
    selectConversation,
  } = useConversation({
    namespace: 'my-app',
    client,
  });

  const handleSend = async (text: string) => {
    await sendMessage(
      [{
        type: MessageType.Message,
        id: Date.now().toString(),
        content: [{ type: ContentType.InputText, text }],
      }],
      {
        baseUrl: client.baseUrl,
        namespace: 'my-app',
        agentName: 'my-agent',
      }
    );
  };

  return (
    <div>
      <button onClick={startNewChat}>New Chat</button>
      
      <div className="messages">
        {allMessages.map((msg) => (
          <Message key={msg.message_id} message={msg} />
        ))}
        {isStreaming && <LoadingIndicator />}
      </div>
      
      <ChatInput onSend={handleSend} disabled={isStreaming} />
    </div>
  );
}
```

## API Reference

### `useConversation(options)`

The main hook for managing conversation state.

#### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `namespace` | `string` | required | Namespace for organizing conversations |
| `client` | `UnoApiClient` | required | API client for communicating with Uno server |
| `autoLoad` | `boolean` | `true` | Auto-load conversations on mount |

#### Return Value

```typescript
interface UseConversationReturn {
  // State
  conversations: Conversation[];
  conversationsLoading: boolean;
  threads: Thread[];
  threadsLoading: boolean;
  currentThread: Thread | null;
  messages: ConversationMessage[];
  streamingMessage: ConversationMessage | null;
  messagesLoading: boolean;
  isStreaming: boolean;
  currentConversationId: string | null;
  currentThreadId: string | null;
  allMessages: ConversationMessage[];

  // Actions
  loadConversations: () => Promise<void>;
  selectConversation: (conversationId: string) => void;
  loadThreads: (conversationId: string) => Promise<void>;
  selectThread: (threadId: string) => void;
  sendMessage: (messages: MessageUnion[], config: ConverseConfig) => Promise<void>;
  startNewChat: () => void;
}
```

### `createApiClient(options)`

Creates a default API client using fetch.

```typescript
const client = createApiClient({
  baseUrl: 'https://your-server.com/api/agent-server',
  headers: { 'Authorization': 'Bearer token' },
  projectId: 'optional-project-id',
});
```

### Custom API Client

You can implement your own API client:

```typescript
import type { UnoApiClient } from '@curaious/uno-converse';

const customClient: UnoApiClient = {
  baseUrl: 'https://your-server.com',
  headers: { 'Authorization': 'Bearer token' },
  
  async getConversations(namespace) {
    // Your implementation
  },
  
  async getThreads(conversationId, namespace) {
    // Your implementation
  },
  
  async getMessages(threadId, namespace) {
    // Your implementation
  },
  
  async getMessage(messageId, namespace) {
    // Your implementation
  },
};
```

### Streaming Utilities

For advanced use cases, you can use the streaming utilities directly:

```typescript
import { streamSSE, ChunkProcessor } from '@curaious/uno-converse';

// Stream SSE events
await streamSSE(
  'https://your-server.com/converse',
  { method: 'POST', body: JSON.stringify(payload) },
  {
    onChunk: (data) => console.log('Chunk:', data),
    onComplete: () => console.log('Done'),
    onError: (err) => console.error('Error:', err),
  }
);

// Process chunks into conversation messages
const processor = new ChunkProcessor(
  'conversation-id',
  'thread-id',
  (conversation) => updateUI(conversation)
);

processor.processChunk(jsonString);
```

## Message Types

The library exports comprehensive types for all message formats:

```typescript
import {
  MessageType,
  ContentType,
  Role,
  type MessageUnion,
  type InputMessage,
  type FunctionCallMessage,
  type ReasoningMessage,
} from '@curaious/uno-converse';

// Create a user message
const userMessage: InputMessage = {
  type: MessageType.Message,
  role: Role.User,
  content: [
    { type: ContentType.InputText, text: 'Hello!' },
  ],
};
```

## Tool Call Approvals

Handle human-in-the-loop scenarios with function call approvals:

```typescript
const { sendMessage } = useConversation({ ... });

// Approve or reject pending tool calls
const handleApproval = async (approvedIds: string[], rejectedIds: string[]) => {
  await sendMessage(
    [{
      type: MessageType.FunctionCallApprovalResponse,
      id: Date.now().toString(),
      approved_call_ids: approvedIds,
      rejected_call_ids: rejectedIds,
    }],
    converseConfig
  );
};
```

## Type Guards

The library exports type guard functions for runtime type checking:

```typescript
import {
  isEasyMessage,
  isInputMessage,
  isFunctionCallMessage,
  isReasoningMessage,
} from '@curaious/uno-converse';

allMessages.forEach((msg) => {
  msg.messages.forEach((m) => {
    if (isFunctionCallMessage(m)) {
      console.log('Function call:', m.name, m.arguments);
    } else if (isReasoningMessage(m)) {
      console.log('Reasoning:', m.summary);
    }
  });
});
```

## License

MIT

