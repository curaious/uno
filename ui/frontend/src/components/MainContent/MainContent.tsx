import React, {useMemo} from 'react';
import {Route, Routes, Navigate, useParams} from "react-router";
import {Box} from "@mui/material";

import {ChatPage} from "../../pages/Chat/ChatPage";
import {Builder} from "../../pages/Builder/Builder";
import {Providers} from "../../pages/Providers/Providers";
import {VirtualKeys} from "../../pages/VirtualKeys/VirtualKeys";
import {Prompts} from "../../pages/Prompts/Prompts";
import {PromptVersions} from "../../pages/Prompts/PromptVersions";
import {ProjectsPage} from "../../pages/Projects/Projects";
import {Traces} from "../../pages/Traces/Traces";
import {ConversationTraces} from "../../pages/ConversationTraces/ConversationTraces";
import {AgentBuilder, AgentBuilderDetail} from "../../pages/AgentBuilder";
import {ProjectProvider} from "@curaious/uno-converse";
import {useProjectContext} from "../../contexts/ProjectContext";
import {api} from "../../api";

export const MainContent: React.FC = props => {
  const { selectedProject } = useProjectContext();

  return (
    <ProjectProvider baseUrl={api.defaults.baseURL || "/"} projectName={selectedProject?.name || ''}>
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
            <Route path="/agent-framework/prompts" element={<Prompts/>}/>
            <Route path="/agent-framework/prompts/:name/versions" element={<PromptVersions/>}/>
            <Route path="/agent-framework/agents" element={<AgentBuilder/>}/>
            <Route path="/agent-framework/agents/:id" element={<AgentBuilderDetail/>}/>
            <Route path="/agent-framework/builder" element={<Builder/>}/>
            <Route path="/agent-framework/chat" element={<ChatPage/>}/>
            <Route path="/agent-framework/traces" element={<Traces/>}/>
            <Route path="/agent-framework/traces/:traceId" element={<Traces/>}/>
            <Route path="/agent-framework/conversation-traces" element={<ConversationTraces/>}/>

            <Route path="*" element={<Navigate to="/agent-framework/projects" replace/>}/>
          </Routes>
        </Box>
      </Box>
    </ProjectProvider>
  );
};