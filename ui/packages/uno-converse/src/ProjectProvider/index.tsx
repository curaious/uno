import React, {createContext, useContext, useMemo} from 'react';
import type { ReactNode, ReactElement } from "react";
import { useProject, UseProjectOptions, UseProjectReturn } from '../hooks/useProject';

/**
 * Context value for ProjectProvider
 */
export interface ProjectContextValue extends UseProjectReturn {}

/**
 * Props for ProjectProvider component
 */
export interface ProjectProviderProps {
  /** Base URL of the Uno Agent Server (e.g., 'https://api.example.com/api/agent-server') */
  baseUrl: string;
  /** Project Name used to fetch the project ID */
  projectName: string;
  /** Optional function to get custom headers for requests (e.g., for authentication) */
  getHeaders?: UseProjectOptions['getHeaders'];
  /** Auto-load project on mount (default: true) */
  autoLoad?: boolean;
  /** Child components */
  children: ReactNode;
}

// Create the context with undefined as default to ensure it's used within provider
const ProjectContext = createContext<ProjectContextValue | undefined>(undefined);

/**
 * ProjectProvider component that manages project state using React Context.
 * 
 * This provider wraps the useProject hook and makes the project state available
 * to all child components through context.
 * 
 * @example
 * ```tsx
 * import { ProjectProvider, useProjectContext } from '@praveen001/uno-converse';
 * 
 * function App() {
 *   return (
 *     <ProjectProvider
 *       baseUrl="https://api.example.com/api/agent-server"
 *       projectName="my-project"
 *       getHeaders={() => ({
 *         'Authorization': `Bearer ${getToken()}`,
 *       })}
 *     >
 *       <YourApp />
 *     </ProjectProvider>
 *   );
 * }
 * 
 * function YourApp() {
 *   const { projectId, projectLoading } = useProjectContext();
 *   
 *   if (projectLoading) {
 *     return <div>Loading project...</div>;
 *   }
 *   
 *   return <div>Project ID: {projectId}</div>;
 * }
 * ```
 */
export const ProjectProvider = ({
  baseUrl,
  projectName,
  getHeaders,
  autoLoad = true,
  children,
}: ProjectProviderProps): ReactElement => {
  // Use the useProject hook to manage project state
  const projectState = useProject({
    baseUrl,
    projectName,
    getHeaders,
    autoLoad,
  });

  // Memoize the context value to prevent unnecessary re-renders
  const contextValue = useMemo<ProjectContextValue>(
    () => projectState,
    [
      projectState.projectId,
      projectState.projectLoading,
      projectState.axiosInstance,
      projectState.buildParams,
      projectState.getRequestHeaders,
      projectState.baseUrl,
    ]
  );

  return (
    <ProjectContext.Provider value={contextValue}>
      {children}
    </ProjectContext.Provider>
  );
};

/**
 * Hook to access the project context.
 * 
 * Must be used within a ProjectProvider component.
 * 
 * @throws {Error} If used outside of ProjectProvider
 * 
 * @example
 * ```tsx
 * function MyComponent() {
 *   const { projectId, projectLoading } = useProjectContext();
 *   
 *   return (
 *     <div>
 *       {projectLoading ? 'Loading...' : `Project ID: ${projectId}`}
 *     </div>
 *   );
 * }
 * ```
 */
export const useProjectContext = (): ProjectContextValue => {
  const context = useContext(ProjectContext);
  
  if (context === undefined) {
    throw new Error('useProjectContext must be used within a ProjectProvider');
  }
  
  return context;
};

