// Agent Config types for the Agent Builder UI

export interface ModelConfig {
  provider_type: string;
  model_id: string;
  parameters?: Record<string, any>;
}

export interface PromptConfig {
  raw_prompt?: string;
  prompt_id?: string;
  label?: string;
  version?: number;
}

export interface SchemaConfig {
  name: string;
  description?: string;
  schema?: any;
  source_type?: string;
  source_content?: string;
}

export interface MCPServerConfig {
  name: string;
  endpoint: string;
  headers?: Record<string, string>;
  tool_filters?: string[];
  tools_requiring_human_approval?: string[];
}

export interface SummarizerConfig {
  type: 'llm' | 'sliding_window' | 'none';
  llm_token_threshold?: number;
  llm_keep_recent_count?: number;
  llm_summarizer_prompt?: PromptConfig;
  llm_summarizer_model?: ModelConfig;
  sliding_window_keep_count?: number;
}

export interface HistoryConfig {
  enabled: boolean;
  summarizer?: SummarizerConfig;
}

export interface AgentConfigData {
  runtime?: 'Local' | 'Restate' | 'Temporal';
  model?: ModelConfig;
  prompt?: PromptConfig;
  schema?: SchemaConfig;
  mcp_servers?: MCPServerConfig[];
  history?: HistoryConfig;
}

export interface AgentConfig {
  id: string;
  project_id: string;
  name: string;
  version: number;
  config: AgentConfigData;
  created_at: string;
  updated_at: string;
}

export interface AgentConfigSummary {
  id: string;
  project_id: string;
  name: string;
  latest_version: number;
  created_at: string;
  updated_at: string;
}

export interface CreateAgentConfigRequest {
  name: string;
  config: AgentConfigData;
}

export interface UpdateAgentConfigRequest {
  config: AgentConfigData;
}

