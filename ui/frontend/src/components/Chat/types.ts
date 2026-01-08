export interface Content {
  type: ContentType;
  text: string;
  reasoning?: Reasoning;
  tool_call?: ToolCall;
  tool_result?: ToolResult;
  usage?: Usage;
}

export enum ContentType {
  TEXT = 'text',
  MESSAGE_START = 'message_start',
  MESSAGE_END = 'message_end',
  BLOCK_START = 'block_start',
  BLOCK_END = 'block_end',
  REASONING = 'reasoning',
  TOOL_CALL = 'tool_call',
  TOOL_CALL_RESULT = 'tool_call_result',
  USAGE = 'usage',
  CONTEXT_WINDOW_SIZE = 'context_window_size',
}

export interface Reasoning {
  id: string;
  text: string;
  encrypted_content: string;
}

export interface ToolCall {
  id: string;
  call_id: string;
  name: string;
  args: {
    [key: string]: any;
  }
}

export interface ToolResult {
  id: string;
  call_id: string;
  content: Content[];
}

export interface Usage {
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  cached_input_tokens: number;
}

export type ProviderType = 'OpenAI' | 'Anthropic' | 'Gemini' | 'xAI' | 'Ollama';

export interface ProviderConfig {
  provider_type: ProviderType;
  base_url?: string;
  custom_headers?: { [key: string]: string };
  created_at: string;
  updated_at: string;
}

export interface CreateProviderConfigRequest {
  provider_type: ProviderType;
  base_url?: string;
  custom_headers?: { [key: string]: string };
}

export interface UpdateProviderConfigRequest {
  base_url?: string;
  custom_headers?: { [key: string]: string };
}

export interface APIKey {
  id: string;
  provider_type: ProviderType;
  name: string;
  api_key: string;
  enabled: boolean;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateAPIKeyRequest {
  provider_type: ProviderType;
  name: string;
  api_key: string;
  enabled?: boolean;
  is_default?: boolean;
}

export interface UpdateAPIKeyRequest {
  name?: string;
  api_key?: string;
  enabled?: boolean;
  is_default?: boolean;
}

export interface MCPServer {
  id: string;
  name: string;
  endpoint: string;
  headers: { [key: string]: string };
  created_at: string;
  updated_at: string;
}

export interface CreateMCPServerRequest {
  name: string;
  endpoint: string;
  headers: { [key: string]: string };
}

export interface UpdateMCPServerRequest {
  name?: string;
  endpoint?: string;
  headers?: { [key: string]: string };
}

export interface MCPTool {
  name: string;
  description?: string;
  inputSchema?: any;
  outputSchema?: any;
}

export interface MCPPrompt {
  name: string;
  description?: string;
  arguments?: any[];
}

export interface MCPResource {
  uri: string;
  name?: string;
  description?: string;
  mimeType?: string;
}

export interface MCPInspectResponse {
  tools: MCPTool[];
  prompts?: MCPPrompt[];
  resources?: MCPResource[];
}

export interface Prompt {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface PromptVersion {
  id: string;
  prompt_id: string;
  version: number;
  template: string;
  commit_message: string;
  label?: string;
  created_at: string;
  updated_at: string;
}

export interface PromptWithLatestVersion {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
  latest_version?: number;
  latest_commit_message?: string;
  latest_label?: string;
}

export interface PromptVersionWithPrompt {
  id: string;
  prompt_id: string;
  version: number;
  template: string;
  commit_message: string;
  label?: string;
  created_at: string;
  updated_at: string;
  prompt_name: string;
}

export interface CreatePromptRequest {
  name: string;
  template: string;
  commit_message: string;
  label?: string;
}

export interface CreatePromptVersionRequest {
  template: string;
  commit_message: string;
  label?: string;
}

export interface UpdatePromptVersionLabelRequest {
  label?: string;
}

export interface Model {
  id: string;
  project_id: string;
  provider_type: ProviderType;
  name: string;
  model_id: string;
  parameters: { [key: string]: any };
  created_at: string;
  updated_at: string;
}

export interface ModelWithProvider extends Model {
  provider_type: ProviderType;
}

export interface CreateModelRequest {
  provider_type: ProviderType;
  name: string;
  model_id: string;
  parameters?: { [key: string]: any };
}

export interface UpdateModelRequest {
  provider_type?: ProviderType;
  name?: string;
  model_id?: string;
  parameters?: { [key: string]: any };
}

export interface RateLimit {
  unit: '1min' | '1h' | '6h' | '12h' | '1d' | '1w' | '1mo';
  limit: number;
}

export interface VirtualKey {
  id: string;
  name: string;
  secret_key: string;
  providers: ProviderType[];
  model_ids: string[];
  rate_limits?: RateLimit[];
  created_at: string;
  updated_at: string;
}

export interface CreateVirtualKeyRequest {
  name: string;
  providers: ProviderType[];
  model_ids?: string[];
  rate_limits?: RateLimit[];
}

export interface UpdateVirtualKeyRequest {
  name?: string;
  providers?: ProviderType[];
  model_ids?: string[];
  rate_limits?: RateLimit[];
}

export interface ProviderModelsData {
  models: string[];
}

export interface ProviderModelsResponse {
  providers: {
    [key: string]: ProviderModelsData;
  };
}

export interface ReasoningConfig {
  effort?: 'low' | 'medium' | 'high';
  budget_tokens?: number;
}

export interface ModelParameters {
  temperature?: number;
  top_p?: number;
  max_output_tokens?: number;
  max_tool_calls?: number;
  parallel_tool_calls?: boolean;
  top_logprobs?: number;
  background?: boolean;
  store?: boolean;
  reasoning?: ReasoningConfig;
}

export interface Agent {
  id: string;
  project_id: string;
  name: string;
  model_id: string;
  prompt_id: string;
  prompt_label?: string;
  schema_id?: string;
  enable_history?: boolean;
  summarizer_type?: 'llm' | 'sliding_window' | 'none';
  llm_summarizer_token_threshold?: number;
  llm_summarizer_keep_recent_count?: number;
  llm_summarizer_prompt_id?: string;
  llm_summarizer_prompt_label?: string;
  llm_summarizer_model_id?: string;
  sliding_window_keep_count?: number;
  created_at: string;
  updated_at: string;
}

export interface AgentMCPServer {
  agent_id: string;
  mcp_server_id: string;
  tool_filters: string[];
  tools_requiring_human_approval: string[];
}

export interface AgentMCPServerDetail extends AgentMCPServer {
  mcp_server_name: string;
}

export interface AgentWithDetails extends Agent {
  model_name: string;
  prompt_name: string;
  schema_name?: string;
  schema_data?: JSONSchemaDefinition;
  mcp_servers: AgentMCPServerDetail[];
}

export interface AgentMCPServerReq {
  mcp_server_id: string;
  tool_filters?: string[];
  tools_requiring_human_approval?: string[];
}

export interface CreateAgentRequest {
  name: string;
  model_id: string;
  prompt_id: string;
  prompt_label?: string;
  schema_id?: string;
  enable_history?: boolean;
  summarizer_type?: 'llm' | 'sliding_window' | 'none';
  llm_summarizer_token_threshold?: number;
  llm_summarizer_keep_recent_count?: number;
  llm_summarizer_prompt_id?: string;
  llm_summarizer_prompt_label?: string;
  llm_summarizer_model_id?: string;
  sliding_window_keep_count?: number;
  mcp_servers?: AgentMCPServerReq[];
}

export interface UpdateAgentRequest {
  name?: string;
  model_id?: string;
  prompt_id?: string;
  prompt_label?: string;
  schema_id?: string;
  clear_schema_id?: boolean;
  enable_history?: boolean;
  summarizer_type?: 'llm' | 'sliding_window' | 'none';
  llm_summarizer_token_threshold?: number;
  llm_summarizer_keep_recent_count?: number;
  llm_summarizer_prompt_id?: string;
  llm_summarizer_prompt_label?: string;
  llm_summarizer_model_id?: string;
  sliding_window_keep_count?: number;
  mcp_servers?: AgentMCPServerReq[];
}

// JSON Schema types
export type SchemaSourceType = 'manual' | 'go_struct' | 'typescript';

export interface Schema {
  id: string;
  project_id: string;
  name: string;
  description?: string;
  schema: JSONSchemaDefinition;
  source_type: SchemaSourceType;
  source_content?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSchemaRequest {
  name: string;
  description?: string;
  schema: JSONSchemaDefinition;
  source_type?: SchemaSourceType;
  source_content?: string;
}

export interface UpdateSchemaRequest {
  name?: string;
  description?: string;
  schema?: JSONSchemaDefinition;
  source_type?: SchemaSourceType;
  source_content?: string;
}

// JSON Schema definition types
export type JSONSchemaType = 'string' | 'number' | 'integer' | 'boolean' | 'object' | 'array' | 'null';

export interface JSONSchemaProperty {
  type?: JSONSchemaType | JSONSchemaType[];
  description?: string;
  properties?: Record<string, JSONSchemaProperty>;
  items?: JSONSchemaProperty;
  required?: string[];
  enum?: any[];
  default?: any;
  format?: string;
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  $ref?: string;
  title?: string;
  additionalProperties?: boolean | JSONSchemaProperty;
  oneOf?: JSONSchemaProperty[];
  anyOf?: JSONSchemaProperty[];
  allOf?: JSONSchemaProperty[];
}

export interface JSONSchemaDefinition extends JSONSchemaProperty {
  $schema?: string;
  title?: string;
  definitions?: Record<string, JSONSchemaProperty>;
}