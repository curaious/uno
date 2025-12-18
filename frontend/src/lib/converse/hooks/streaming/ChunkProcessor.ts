import {
  ChunkType,
  ContentType,
  ConversationMessage,
  FunctionCallMessage, FunctionCallOutputMessage, ImageGenerationCallMessage,
  InputMessage,
  MessageUnion,
  ReasoningMessage,
  ResponseChunk
} from "../../types/types";

export type OnChangeCallback = (conversation: ConversationMessage) => void;

/**
 * Processes streaming chunks from the LLM response.
 * Builds up messages incrementally as chunks arrive.
 */
export class ChunkProcessor {
  private messages: MessageUnion[] = [];
  private currentOutputItem: MessageUnion | null = null;
  private _onChange: OnChangeCallback;
  private conversation: ConversationMessage;
  private receivedContent: boolean = false;

  constructor(
    conversationId: string,
    threadId: string,
    messageId: string,
    onChange: OnChangeCallback
  ) {
    this.conversation = {
      conversation_id: conversationId,
      thread_id: threadId,
      message_id: messageId,
      messages: [],
      meta: {},
    };
    this._onChange = onChange;
  }

  getMessages(): MessageUnion[] {
    return this.messages;
  }

  getConversation(): ConversationMessage {
    return this.conversation;
  }

  private emitChange(): void {
    this.conversation.messages = [...this.messages];
    this._onChange({ ...this.conversation });
  }

  processChunk(data: string): void {
    try {
      const chunk: ResponseChunk = JSON.parse(data);
      this.handleChunk(chunk);
    } catch (e) {
      console.error('Failed to parse chunk:', e, data);
    }
  }

  private handleChunk(chunk: ResponseChunk): void {
    switch (chunk.type) {
      // Run lifecycle
      case ChunkType.ChunkTypeRunCreated:
      case ChunkType.ChunkTypeRunCompleted:
        break;

      // Response lifecycle
      case ChunkType.ChunkTypeResponseCreated:
      case ChunkType.ChunkTypeResponseInProgress:
        break;

      case ChunkType.ChunkTypeResponseCompleted:
        // Could extract usage info from chunk.response?.usage here
        if (chunk.response?.usage) {
          this.conversation.meta.usage = chunk.response.usage;
          this.emitChange();
        }
        break;

      // Output item lifecycle
      case ChunkType.ChunkTypeOutputItemAdded:
        this.handleOutputItemAdded(chunk);
        break;

      case ChunkType.ChunkTypeOutputItemDone:
        break;

      // Content parts
      case ChunkType.ChunkTypeContentPartAdded:
        this.handleContentPartAdded(chunk);
        break;

      case ChunkType.ChunkTypeContentPartDone:
        break;

      // Text deltas
      case ChunkType.ChunkTypeOutputTextDelta:
        this.receivedContent = true;
        this.handleOutputTextDelta(chunk);
        break;

      case ChunkType.ChunkTypeOutputTextDone:
        break;

      // Reasoning summary
      case ChunkType.ChunkTypeReasoningSummaryPartAdded:
        this.handleReasoningSummaryPartAdded(chunk);
        break;

      case ChunkType.ChunkTypeReasoningSummaryPartDone:
        break;

      case ChunkType.ChunkTypeReasoningSummaryTextDelta:
        this.handleReasoningSummaryTextDelta(chunk);
        break;

      case ChunkType.ChunkTypeReasoningSummaryTextDone:
        break;

      // Function calls
      case ChunkType.ChunkTypeFunctionCallArgumentsDelta:
        this.handleFunctionCallArgumentsDelta(chunk);
        break;

      case ChunkType.ChunkTypeFunctionCallArgumentsDone:
        break;

      case ChunkType.ChunkTypeFunctionCallOutput:
        this.handleFunctionCallOutput(chunk)
        break;

        // Image Generation Calls
      case ChunkType.ChunkTypeImageGenerationCallInProgress:
        break;

      case ChunkType.ChunkTypeImageGenerationCallGenerating:
        break;

      case ChunkType.ChunkTypeImageGenerationCallPartialImage:
        this.handleImageGenerationCallPartialImage(chunk);
        break;
    }
  }

  private handleOutputItemAdded(chunk: ResponseChunk): void {
    if (!chunk.item) return;

    switch (chunk.item.type) {
      case "message":
        this.currentOutputItem = {
          id: chunk.item.id,
          type: "message",
          role: chunk.item.role || "assistant",
          content: [],
        } as InputMessage;
        break;

      case "function_call":
        this.currentOutputItem = {
          id: chunk.item.id,
          type: "function_call",
          name: chunk.item.name || "",
          call_id: chunk.item.call_id || "",
          arguments: "",
        } as FunctionCallMessage;
        break;

      case "reasoning":
        this.currentOutputItem = {
          id: chunk.item.id,
          type: "reasoning",
          summary: [], // Initialize summary as empty array
        } as ReasoningMessage;
        break;

      case "image_generation_call":
        this.currentOutputItem = {
          id: chunk.item.id,
          type: "image_generation_call",
          status: chunk.item.status,
        } as ImageGenerationCallMessage
    }

    if (this.currentOutputItem) {
      this.messages.push(this.currentOutputItem);
      this.emitChange();
    }
  }

  private handleContentPartAdded(chunk: ResponseChunk): void {
    if (!this.currentOutputItem || this.currentOutputItem.type !== "message") return;

    const message = this.currentOutputItem as InputMessage;
    if (chunk.part?.type === ContentType.OutputText) {
      message.content = message.content || [];
      message.content.push({
        type: ContentType.OutputText,
        text: "",
      });
      this.emitChange();
    }
  }

  private handleOutputTextDelta(chunk: ResponseChunk): void {
    if (!this.currentOutputItem || this.currentOutputItem.type !== "message") return;

    const message = this.currentOutputItem as InputMessage;
    const contents = message.content;
    if (!contents?.length || !chunk.delta) return;

    const lastContent = contents[contents.length - 1];
    if (lastContent && 'text' in lastContent) {
      lastContent.text += chunk.delta;
      this.emitChange();
    }
  }

  private handleReasoningSummaryPartAdded(chunk: ResponseChunk): void {
    if (!this.currentOutputItem || this.currentOutputItem.type !== "reasoning") return;

    const reasoning = this.currentOutputItem as ReasoningMessage;
    if (chunk.part?.type === ContentType.SummaryText) {
      reasoning.summary = reasoning.summary || [];
      reasoning.summary.push({
        type: ContentType.SummaryText,
        text: "",
      });
      this.emitChange();
    }
  }

  private handleReasoningSummaryTextDelta(chunk: ResponseChunk): void {
    if (!this.currentOutputItem || this.currentOutputItem.type !== "reasoning") return;

    const reasoning = this.currentOutputItem as ReasoningMessage;
    const summaries = reasoning.summary;
    if (!summaries?.length || !chunk.delta) return;

    const lastSummary = summaries[summaries.length - 1];
    if (lastSummary) {
      lastSummary.text += chunk.delta;
      this.emitChange();
    }
  }

  private handleFunctionCallArgumentsDelta(chunk: ResponseChunk): void {
    if (!this.currentOutputItem || this.currentOutputItem.type !== "function_call") return;

    const functionCall = this.currentOutputItem as FunctionCallMessage;
    functionCall.arguments += chunk.delta || "";
    this.emitChange();
  }

  private handleFunctionCallOutput(chunk: ResponseChunk): void {
    this.currentOutputItem = chunk as any as FunctionCallOutputMessage;
    this.messages.push(this.currentOutputItem);
    console.log("chunk", this.currentOutputItem);
    this.emitChange();
  }

  private handleImageGenerationCallPartialImage(chunk: ResponseChunk): void {
    if (!this.currentOutputItem || this.currentOutputItem.type !== "image_generation_call") return;

    const image = this.currentOutputItem as ImageGenerationCallMessage;
    image.result = chunk.partial_image_b64!
    image.quality = chunk.quality!
    image.size = chunk.size!
    image.output_format = chunk.output_format!
    image.background = chunk.background!

    this.emitChange();
  }
}

