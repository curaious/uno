import axios from 'axios';
import {useCallback, useEffect, useMemo, useState} from "react";
import {Agent} from "../types";
import {GetHeadersFn} from "./useConversation";
import { useProjectContext } from '../ProjectProvider';

/**
 * Options for the useAgent hook
 */
export interface UseAgentOptions {
  /** The name of the agent */
  name: string;
}

/**
 * Return type for the useAgent hook
 */
export interface UseAgentReturn {
  // Agent
  agent: Agent | null;
  /** Whether the agent is being loaded */
  agentLoading: boolean;
}

export function useAgent(options: UseAgentOptions): UseAgentReturn {
  const { name } = options;

  // Get project context (axios instance, projectId, etc.)
  const {
    axiosInstance,
    projectId,
    buildParams,
  } = useProjectContext();

  // Agents state
  const [agent, setAgent] = useState<Agent | null>(null);
  const [agentLoading, setAgentLoading] = useState(false);

  // ============================================
  // Agent Management
  // ============================================

  /**
   * Fetch the agent
   */
  const loadAgent = useCallback(async (): Promise<void> => {
    setAgentLoading(true);
    try {
      const response = await axiosInstance.get<{data: Agent}>('/agent-configs/by-name', {
        params: buildParams({ name }),
      });
      setAgent(response.data.data);
    } catch (error) {
      console.error('Failed to load agent:', error)
      throw error;
    } finally {
      setAgentLoading(false);
    }
  }, [axiosInstance, name]);

  // Fetch agent after project is fetched
  useEffect(() => {
    if (projectId) {
      loadAgent();
    }
  }, [projectId, loadAgent]);

  return { agent, agentLoading };
}