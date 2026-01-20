import axios, { AxiosInstance } from 'axios';
import {useCallback, useEffect, useMemo, useRef, useState} from "react";
import {Agent} from "../types";
import {GetHeadersFn} from "./useConversation";

/**
 * Options for the useProject hook
 */
export interface UseProjectOptions {
  /** Base URL of the Uno Agent Server (e.g., 'https://api.example.com/api/agent-server') */
  baseUrl: string;
  /** Project Name used to fetch the project ID */
  projectName: string;
  /** Optional function to get custom headers for requests (e.g., for authentication) */
  getHeaders?: GetHeadersFn;
  /** Auto-load conversations on mount (default: true) */
  autoLoad?: boolean;
}

/**
 * Return type for the useProject hook
 */
export interface UseProjectReturn {
  // Project state
  /** The fetched project ID */
  projectId: string;
  /** Whether the project ID is being fetched */
  projectLoading: boolean;
  /** Axios instance configured with baseUrl and custom headers */
  axiosInstance: AxiosInstance;
  /** Function to build query params with project_id automatically added */
  buildParams: (params?: Record<string, string>) => Record<string, string>;
  /** Function to get request headers (combines default + custom headers) */
  getRequestHeaders: () => Promise<Record<string, string>>;
  /** Base URL used for the axios instance */
  baseUrl: string;
}

export function useProject(options: UseProjectOptions): UseProjectReturn {
  const { projectName, baseUrl, getHeaders, autoLoad = true } = options;

  // Project state
  const [projectId, setProjectId] = useState<string>('');
  const [projectLoading, setProjectLoading] = useState(false);
  
  // Use ref to store current projectId for interceptor access
  const projectIdRef = useRef<string>('');
  
  // Update ref whenever projectId changes
  useEffect(() => {
    projectIdRef.current = projectId;
  }, [projectId]);

  // Create axios instance with request interceptor for custom headers and project_id
  const axiosInstance = useMemo(() => {
    const instance = axios.create({
      baseURL: baseUrl,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add request interceptor to inject custom headers and project_id
    instance.interceptors.request.use(async (config) => {
      // Add custom headers if getHeaders function is provided
      if (getHeaders) {
        const customHeaders = await getHeaders();
        Object.assign(config.headers, customHeaders);
      }
      
      // Automatically add project_id to query params if available
      const currentProjectId = projectIdRef.current;
      if (currentProjectId) {
        if (config.params) {
          config.params = {
            ...config.params,
            project_id: currentProjectId,
          };
        } else {
          config.params = { project_id: currentProjectId };
        }
      }
      
      return config;
    });

    return instance;
  }, [baseUrl, getHeaders]);


  // ============================================
  // API Helper Functions
  // ============================================

  /**
   * Build query params (project_id is automatically added by axios interceptor)
   */
  const buildParams = useCallback((params?: Record<string, string>) => {
    return params || {};
  }, []);

  /**
   * Get headers for streaming requests (combines default + custom headers)
   */
  const getRequestHeaders = useCallback(async (): Promise<Record<string, string>> => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    // Add custom headers if getHeaders function is provided
    if (getHeaders) {
      const customHeaders = await getHeaders();
      Object.assign(headers, customHeaders);
    }

    return headers;
  }, [getHeaders]);

  // ============================================
  // Project Management
  // ============================================

  /**
   * Fetch the project ID using the project name
   */
  const fetchProjectId = useCallback(async () => {
    if (!projectName) {
      return;
    }

    setProjectLoading(true);
    try {
      const response = await axiosInstance.get<{ data: string } | string>('/project/id', {
        params: { name: projectName },
      });
      const id = typeof response.data === 'string' ? response.data : response.data.data;
      setProjectId(id || '');
    } catch (error) {
      console.error('Failed to fetch project ID:', error);
      throw error;
    } finally {
      setProjectLoading(false);
    }
  }, [axiosInstance, projectName]);

  // ============================================
  // Effects for auto-loading
  // ============================================

  // Fetch project ID on mount
  useEffect(() => {
    if (autoLoad && projectName) {
      fetchProjectId();
    }
  }, [autoLoad, projectName, fetchProjectId]);

  return {
    // Project state
    projectId,
    projectLoading,
    // API client
    axiosInstance,
    buildParams,
    getRequestHeaders,
    baseUrl,
  }
}