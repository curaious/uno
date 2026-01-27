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

export interface ImageGenerationToolConfig {
  enabled: boolean;
  // Future config options can be added here
  [key: string]: any;
}

export interface WebSearchToolConfig {
  enabled: boolean;
  // Future config options can be added here
  [key: string]: any;
}

export interface CodeExecutionToolConfig {
  enabled: boolean;
  // Future config options can be added here
  [key: string]: any;
}

export interface SandboxToolConfig {
  enabled: boolean;
  docker_image?: string;
  // Future config options can be added here
  [key: string]: any;
}

export interface ToolsConfig {
  image_generation?: ImageGenerationToolConfig;
  web_search?: WebSearchToolConfig;
  code_execution?: CodeExecutionToolConfig;
  sandbox?: SandboxToolConfig;
}

// Skill stored in the agent config
export interface SkillConfig {
  name: string;          // Skill name from SKILL.md frontmatter
  description: string;   // Skill description from SKILL.md frontmatter
  file_location: string; // Path to SKILL.md file relative to sandbox-data
}

// Temp skill that's been uploaded but not yet saved
export interface TempSkillUploadResponse {
  name: string;         // Skill name parsed from SKILL.md
  description: string;  // Skill description parsed from SKILL.md
  temp_path: string;    // Path to the temp folder where skill was extracted
  skill_folder: string; // Folder name of the skill (from zip name)
}

export interface AgentConfigData {
  runtime?: 'Local' | 'Restate' | 'Temporal';
  max_iteration?: number;
  model?: ModelConfig;
  prompt?: PromptConfig;
  schema?: SchemaConfig;
  mcp_servers?: MCPServerConfig[];
  history?: HistoryConfig;
  tools?: ToolsConfig;
  skills?: SkillConfig[];
}

export interface AgentConfig {
  id: string;
  agent_id: string;
  project_id: string;
  name: string;
  version: number;
  immutable: boolean;
  config: AgentConfigData;
  created_at: string;
  updated_at: string;
}

export interface AgentConfigSummary {
  id: string;
  agent_id: string;
  project_id: string;
  name: string;
  runtime: string;
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

export interface AgentConfigAlias {
  id: string;
  project_id: string;
  agent_id: string;
  name: string;
  version1: number;
  version2?: number;
  weight?: number;
  created_at: string;
  updated_at: string;
}

export interface CreateAliasRequest {
  name: string;
  version1: number;
  version2?: number;
  weight?: number;
}

export interface UpdateAliasRequest {
  name?: string;
  version1?: number;
  version2?: number;
  weight?: number;
}

