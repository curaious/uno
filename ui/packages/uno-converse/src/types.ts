// Message Roles
export enum Role {
  User = "user",
  Developer = "developer",
  System = "system",
  Assistant = "assistant",
}

// Message Types
export enum MessageType {
  Message = "message",
  FunctionCall = "function_call",
  FunctionCallOutput = "function_call_output",
  Reasoning = "reasoning",
  ImageGenerationCall = "image_generation_call",
  FunctionCallApprovalResponse = "function_call_approval_response",
}

// Content Types
export enum ContentType {
  InputText = "input_text",
  OutputText = "output_text",
  SummaryText = "summary_text",
  InputImage = "input_image",
}

// Chunk Types
export enum ChunkType {
  ChunkTypeRunCreated = "run.created",
  ChunkTypeRunInProgress = "run.in_progress",
  ChunkTypeRunPaused = "run.paused",
  ChunkTypeRunCompleted = "run.completed",
  ChunkTypeResponseCreated = "response.created",
  ChunkTypeResponseInProgress = "response.in_progress",
  ChunkTypeResponseCompleted = "response.completed",
  ChunkTypeOutputItemAdded = "response.output_item.added",
  ChunkTypeOutputItemDone = "response.output_item.done",
  ChunkTypeContentPartAdded = "response.content_part.added",
  ChunkTypeContentPartDone = "response.content_part.done",
  ChunkTypeOutputTextDelta = "response.output_text.delta",
  ChunkTypeOutputTextDone = "response.output_text.done",
  ChunkTypeFunctionCallArgumentsDelta = "response.function_call_arguments.delta",
  ChunkTypeFunctionCallArgumentsDone = "response.function_call_arguments.done",
  ChunkTypeReasoningSummaryPartAdded = "response.reasoning_summary_part.added",
  ChunkTypeReasoningSummaryPartDone = "response.reasoning_summary_part.done",
  ChunkTypeReasoningSummaryTextDelta = "response.reasoning_summary_text.delta",
  ChunkTypeReasoningSummaryTextDone = "response.reasoning_summary_text.done",

  // Image generation
  ChunkTypeImageGenerationCallInProgress = "response.image_generation_call.in_progress",
  ChunkTypeImageGenerationCallGenerating = "response.image_generation_call.generating",
  ChunkTypeImageGenerationCallPartialImage = "response.image_generation_call.partial_image",

  // Extra
  ChunkTypeFunctionCallOutput = "function_call_output",
}

/**
 * Represents a conversation container
 */
export interface Conversation {
  namespace_id: string;
  conversation_id: string;
  name: string;
  created_at: string;
  last_updated: string;
}

/**
 * Represents a thread within a conversation
 */
export interface Thread {
  conversation_id: string;
  origin_message_id: string;
  thread_id: string;
  meta: Record<string, any>;
  created_at: string;
  last_updated: string;
}

/**
 * Represents a message within a conversation thread
 */
export interface ConversationMessage {
  conversation_id: string;
  thread_id: string;
  message_id: string;
  messages: MessageUnion[];
  meta: Record<string, any>;
  isStreaming?: boolean;
}

// Message union type - discriminated by 'type' field
export type MessageUnion =
  | EasyMessage
  | InputMessage
  | OutputMessage
  | FunctionCallMessage
  | FunctionCallApprovalResponseMessage
  | FunctionCallOutputMessage
  | ReasoningMessage
  | ImageGenerationCallMessage;

export interface EasyMessage {
  type: MessageType.Message;
  id: string;
  role?: Role;
  content: EasyInputContentUnion;
}

export interface InputMessage {
  type: MessageType.Message;
  id?: string;
  role?: Role;
  content?: InputContentUnion[];
}

export interface OutputMessage {
  id: string;
  type?: MessageType.Message;
  role?: Role;
  content?: OutputContentUnion[];
}

export interface FunctionCallMessage {
  type: MessageType.FunctionCall;
  id: string;
  call_id?: string;
  name: string;
  arguments: string;
}

export interface FunctionCallApprovalResponseMessage {
  type: MessageType.FunctionCallApprovalResponse;
  id: string;
  approved_call_ids: string[];
  rejected_call_ids: string[];
}

export interface FunctionCallOutputMessage {
  type: MessageType.FunctionCallOutput;
  id: string;
  call_id: string;
  output: FunctionCallOutputContentUnion;
}

export interface ReasoningMessage {
  type: MessageType.Reasoning;
  id: string;
  summary?: SummaryTextContent[];
  encrypted_content?: string;
}

export interface ImageGenerationCallMessage {
  type: MessageType.ImageGenerationCall;
  id: string;
  status: string;
  background: string;
  output_format: string;
  quality: string;
  size: string;
  result: string;
}

// Content Unions
export type EasyInputContentUnion = string | InputContentUnion;
export type InputContentUnion = InputTextContent | OutputTextContent | InputImageContent;
export type FunctionCallOutputContentUnion = string | InputContentUnion;
export type OutputContentUnion =
  | OutputTextContent
  | FunctionCallMessage
  | SummaryTextContent;

// Content types
export interface InputTextContent {
  type: ContentType.InputText;
  text: string;
}

export interface OutputTextContent {
  type: ContentType.OutputText;
  text: string;
}

export interface SummaryTextContent {
  type: ContentType.SummaryText;
  text: string;
}

export interface InputImageContent {
  type: ContentType.InputImage;
  image_url: string;
  detail: string;
}

/**
 * Configuration for the converse API endpoint
 */
export interface ConverseConfig {
  namespace: string;
  agentId: string;
  baseUrl?: string;
  context?: Record<string, unknown>;
  headers?: Record<string, string>;
}

// Streaming chunk types
export interface ResponseChunk {
  type: ChunkType;
  sequence_number: number;

  // Only on run items
  run_state?: ChunkRunData;

  // Only on response items
  response?: ChunkResponseData;

  // On non-run and non-response items
  output_index?: number;

  // Only on output_item
  item?: ChunkOutputItemData;

  // Only on content_part and delta
  item_id?: string;
  content_index?: number;

  // Only on content_part
  part?: OutputContentUnion;

  // Only on output_text delta/done
  delta?: string;
  text?: string;

  // Only on function_call arguments delta/done
  arguments?: string;

  // Only on reasoning summary part/delta/done
  summary_index?: number;

  // Only on function_call_output
  output: string;

  // Only on image_generation_call.partial_image
  partial_image_index?: number;
  partial_image_b64?: string;
  background?: string;
  output_format?: string;
  quality?: string;
  size?: string;
  status?: string;
}

export interface ChunkRunData {
  id: string;
  object: "run";
  status: "created" | "in_progress" | "paused" | "resumed" | "completed" | "aborted";
  pending_tool_calls: FunctionCallMessage[];
  usage: ChunkResponseUsage;
  traceid: string;
}

export interface ChunkResponseData {
  id: string;
  object: string;
  created_at: number;
  status: string;
  background: boolean;
  error: unknown;
  incomplete_details: unknown;
  output: OutputMessageUnion[];
  usage: ChunkResponseUsage;
}

export interface ChunkOutputItemData {
  type: string; // "function_call", "message", "reasoning"

  // Common fields
  id: string;
  status: string;

  // For output_item of type "message"
  content: OutputContentUnion[];
  role: Role;

  // For output_item of type "function_call"
  call_id?: string;
  name?: string;
  arguments?: string;

  // For "reasoning"
  encrypted_content?: string;
  summary?: OutputContentUnion[];

  // For output_item of type "image_generation_call"
  background?: string;
  output_format?: string;
  quality?: string;
  result?: string;
  size?: string;
}

export type OutputMessageUnion =
  | (OutputMessage & { id: string })
  | (FunctionCallMessage & { id: string })
  | (ReasoningMessage & { id: string })
  | (ImageGenerationCallMessage & { id: string });

export interface ChunkResponseUsage {
  input_tokens: number;
  input_tokens_details: {
    cached_tokens: number;
  };
  output_tokens: number;
  output_tokens_details: {
    reasoning_tokens: number;
  };
  total_tokens: number;
}

export interface Usage {
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  input_tokens_details: {
    cached_tokens: number;
  };
  output_tokens_details: {
    reasoning_tokens: number;
  };
}

// Type guards
export function isEasyMessage(msg: MessageUnion): msg is EasyMessage {
  return msg.type === MessageType.Message && 'content' in msg && (typeof msg.content === 'string' || Array.isArray(msg.content));
}

export function isInputMessage(msg: MessageUnion): msg is InputMessage {
  return msg.type === MessageType.Message && 'content' in msg && Array.isArray(msg.content);
}

export function isFunctionCallMessage(msg: MessageUnion): msg is FunctionCallMessage {
  return msg.type === MessageType.FunctionCall;
}

export function isFunctionCallOutputMessage(msg: MessageUnion): msg is FunctionCallOutputMessage {
  return msg.type === MessageType.FunctionCallOutput;
}

export function isReasoningMessage(msg: MessageUnion): msg is ReasoningMessage {
  return msg.type === MessageType.Reasoning;
}

export function isImageGenerationCallMessage(msg: MessageUnion): msg is ImageGenerationCallMessage {
  return msg.type === MessageType.ImageGenerationCall;
}

