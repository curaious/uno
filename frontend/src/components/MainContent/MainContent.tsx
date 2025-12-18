import React from 'react';
import {Route, Routes, Navigate, useParams} from "react-router";
import {Box} from "@mui/material";

import {ChatPage} from "../../pages/Chat/ChatPage";
import {Builder} from "../../pages/Builder/Builder";
import {Providers} from "../../pages/Providers/Providers";
import {Models} from "../../pages/Models/Models";
import {Agents} from "../../pages/Agents/Agents";
import {VirtualKeys} from "../../pages/VirtualKeys/VirtualKeys";
import {MCPServers} from "../../pages/MCPServers/MCPServers";
import {MCPInspect} from "../../pages/MCPServers/MCPInspect";
import {Prompts} from "../../pages/Prompts/Prompts";
import {PromptVersions} from "../../pages/Prompts/PromptVersions";
import {Schemas} from "../../pages/Schemas/Schemas";
import {ProjectsPage} from "../../pages/Projects/Projects";
import {Traces} from "../../pages/Traces/Traces";
import {ConversationTraces} from "../../pages/ConversationTraces/ConversationTraces";

// Redirect components for dynamic routes
const RedirectMCPInspect: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  return <Navigate to={`/agent-framework/mcp-servers/${id}/inspect`} replace/>;
};

const RedirectPromptVersions: React.FC = () => {
  const { name } = useParams<{ name: string }>();
  return <Navigate to={`/agent-framework/prompts/${name}/versions`} replace/>;
};

export const MainContent: React.FC = props => {
  return (
    <Box display="flex" flexDirection="column" flex={1} style={{ padding: 16, background: '#000' }}>
      <Box display="flex" flexDirection="column" flex={1} style={{ background: 'oklch(21% .006 285.885)' }}>
        <Routes>
          {/* Gateway routes */}
          <Route path="/gateway/providers" element={<Providers/>}/>
          <Route path="/gateway/virtual-keys" element={<VirtualKeys/>}/>
          <Route path="/gateway/traces" element={<Traces/>}/>
          <Route path="/gateway/traces/:traceId" element={<Traces/>}/>
          
          {/* Agent Framework routes */}
          <Route path="/agent-framework" element={<Navigate to="/agent-framework/projects" replace/>}/>
          <Route path="/agent-framework/projects" element={<ProjectsPage/>}/>
          <Route path="/agent-framework/models" element={<Models/>}/>
          <Route path="/agent-framework/prompts" element={<Prompts/>}/>
          <Route path="/agent-framework/prompts/:name/versions" element={<PromptVersions/>}/>
          <Route path="/agent-framework/schemas" element={<Schemas/>}/>
          <Route path="/agent-framework/mcp-servers" element={<MCPServers/>}/>
          <Route path="/agent-framework/mcp-servers/:id/inspect" element={<MCPInspect/>}/>
          <Route path="/agent-framework/agents" element={<Agents/>}/>
          <Route path="/agent-framework/builder" element={<Builder/>}/>
          <Route path="/agent-framework/chat" element={<ChatPage/>}/>
          <Route path="/agent-framework/traces" element={<Traces/>}/>
          <Route path="/agent-framework/traces/:traceId" element={<Traces/>}/>
          <Route path="/agent-framework/conversation-traces" element={<ConversationTraces/>}/>
          
          {/* Legacy routes - redirect to new paths */}
          <Route path="/" element={<Navigate to="/agent-framework/projects" replace/>}/>
          <Route path="/projects" element={<Navigate to="/agent-framework/projects" replace/>}/>
          <Route path="/providers" element={<Navigate to="/gateway/providers" replace/>}/>
          <Route path="/models" element={<Navigate to="/agent-framework/models" replace/>}/>
          <Route path="/virtual-keys" element={<Navigate to="/gateway/virtual-keys" replace/>}/>
          <Route path="/agents" element={<Navigate to="/agent-framework/agents" replace/>}/>
          <Route path="/mcp-servers" element={<Navigate to="/agent-framework/mcp-servers" replace/>}/>
          <Route path="/mcp-servers/:id/inspect" element={<RedirectMCPInspect/>}/>
          <Route path="/prompts" element={<Navigate to="/agent-framework/prompts" replace/>}/>
          <Route path="/prompts/:name/versions" element={<RedirectPromptVersions/>}/>
          <Route path="/builder" element={<Navigate to="/agent-framework/builder" replace/>}/>
          <Route path="/chat" element={<Navigate to="/agent-framework/chat" replace/>}/>
          <Route path="/traces" element={<Navigate to="/agent-framework/traces" replace/>}/>
          <Route path="/conversation-traces" element={<Navigate to="/agent-framework/conversation-traces" replace/>}/>
          
          <Route path="*" element={<Navigate to="/agent-framework/projects" replace/>}/>
        </Routes>
      </Box>
    </Box>
  );
};