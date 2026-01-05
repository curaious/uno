export { useConversation } from './hooks';
export type {
  GetHeadersFn,
  UseConversationOptions,
  UseConversationReturn,
} from './hooks';

// Streaming utilities
export {
  ChunkProcessor,
  streamSSE,
} from './streaming';
export type {
  OnChangeCallback,
  SSEStreamOptions,
} from './streaming';

// Types
export {
  // Enums
  Role,
  MessageType,
  ContentType,
  ChunkType,
  // Type guards
  isEasyMessage,
  isInputMessage,
  isFunctionCallMessage,
  isFunctionCallOutputMessage,
  isReasoningMessage,
  isImageGenerationCallMessage,
} from './types';

export type {
  // Core types
  Conversation,
  Thread,
  ConversationMessage,
  ConverseConfig,
  // Message types
  MessageUnion,
  EasyMessage,
  InputMessage,
  OutputMessage,
  FunctionCallMessage,
  FunctionCallApprovalResponseMessage,
  FunctionCallOutputMessage,
  ReasoningMessage,
  ImageGenerationCallMessage,
  // Content types
  EasyInputContentUnion,
  InputContentUnion,
  FunctionCallOutputContentUnion,
  OutputContentUnion,
  InputTextContent,
  OutputTextContent,
  SummaryTextContent,
  InputImageContent,
  // Chunk types
  ResponseChunk,
  ChunkRunData,
  ChunkResponseData,
  ChunkOutputItemData,
  OutputMessageUnion,
  ChunkResponseUsage,
  Usage,
} from './types';

