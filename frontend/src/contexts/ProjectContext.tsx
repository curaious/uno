import React, {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState
} from 'react';
import {api} from '../api';

export interface Project {
  id: string;
  name: string;
  default_key?: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateProjectRequest {
  name: string;
  default_key?: string | null;
}

export interface UpdateProjectRequest {
  name?: string;
  default_key?: string | null;
}

interface ProjectContextValue {
  projects: Project[];
  selectedProjectId: string | null;
  selectedProject: Project | null;
  loading: boolean;
  error: string | null;
  refreshProjects: () => Promise<void>;
  selectProject: (projectId: string) => void;
  createProject: (name: string, defaultKey?: string | null) => Promise<Project>;
  updateProject: (projectId: string, payload: UpdateProjectRequest) => Promise<Project>;
  deleteProject: (projectId: string) => Promise<void>;
}

export const STORAGE_KEY = 'planner.selectedProjectName';

const ProjectContext = createContext<ProjectContextValue | undefined>(undefined);

const getErrorMessage = (error: any, fallback: string) => {
  return (
    error?.response?.data?.message ||
    error?.response?.data?.errorDetails?.error ||
    error?.message ||
    fallback
  );
};

interface ProjectProviderProps {
  children: ReactNode;
}

export const ProjectProvider: React.FC<ProjectProviderProps> = ({children}) => {
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(() => {
    if (typeof window === 'undefined') {
      return null;
    }
    try {
      return localStorage.getItem(STORAGE_KEY);
    } catch (err) {
      console.warn('Unable to read selected project from storage', err);
      return null;
    }
  });
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const persistSelectedProject = useCallback((projectId: string | null) => {
        console.log("persist", projectId);
    try {
      if (projectId) {
        localStorage.setItem(STORAGE_KEY, projectId);
      } else {
        localStorage.removeItem(STORAGE_KEY);
      }
    } catch (err) {
      console.warn('Unable to persist selected project', err);
    }
  }, []);

  const resolveSelection = useCallback((availableProjects: Project[], currentId: string | null) => {
    if (availableProjects.length === 0) {
      persistSelectedProject(null);
      return null;
    }

    if (currentId && availableProjects.some(project => project.id === currentId)) {
      return currentId;
    }

    const fallback = availableProjects[0].id;
    persistSelectedProject(fallback);
    return fallback;
  }, [persistSelectedProject]);

  const refreshProjects = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.get('/projects');
      const fetchedProjects: Project[] = response.data?.data || [];
      setProjects(fetchedProjects);
      setError(null);
      setSelectedProjectId(prev => resolveSelection(fetchedProjects, prev));
    } catch (err) {
      const message = getErrorMessage(err, 'Failed to load projects');
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [resolveSelection]);

  useEffect(() => {
    void refreshProjects();
  }, [refreshProjects]);

  const selectProject = useCallback((projectId: string) => {
    setSelectedProjectId(projectId);
    persistSelectedProject(projectId);
  }, [persistSelectedProject]);

  const createProject = useCallback(async (name: string, defaultKey?: string | null) => {
    try {
      const response = await api.post('/projects', {name, default_key: defaultKey || null} satisfies CreateProjectRequest);
      const newProject: Project = response.data?.data;
      setProjects(prev => {
        const updated = [newProject, ...prev];
        setSelectedProjectId(resolveSelection(updated, newProject.id));
        return updated;
      });
      setError(null);
      persistSelectedProject(newProject.id);
      return newProject;
    } catch (err) {
      const message = getErrorMessage(err, 'Failed to create project');
      setError(message);
      throw err;
    }
  }, [persistSelectedProject, resolveSelection]);

  const updateProject = useCallback(async (projectId: string, payload: UpdateProjectRequest) => {
    try {
      const response = await api.put(`/projects/${encodeURIComponent(projectId)}`, payload);
      const updatedProject: Project = response.data?.data;
      setProjects(prev => prev.map(project => project.id === projectId ? updatedProject : project));
      setError(null);
      return updatedProject;
    } catch (err) {
      const message = getErrorMessage(err, 'Failed to update project');
      setError(message);
      throw err;
    }
  }, []);

  const deleteProject = useCallback(async (projectId: string) => {
    try {
      await api.delete(`/projects/${encodeURIComponent(projectId)}`);
      setProjects(prev => {
        const filtered = prev.filter(project => project.id !== projectId);
        setSelectedProjectId(currentId => resolveSelection(filtered, currentId === projectId ? null : currentId));
        return filtered;
      });
      setError(null);
    } catch (err) {
      const message = getErrorMessage(err, 'Failed to delete project');
      setError(message);
      throw err;
    }
  }, [resolveSelection]);

  const selectedProject = useMemo(() => {
    if (!selectedProjectId) {
      return null;
    }
    return projects.find(project => project.id === selectedProjectId) || null;
  }, [projects, selectedProjectId]);

  const contextValue = useMemo<ProjectContextValue>(() => ({
    projects,
    selectedProjectId,
    selectedProject,
    loading,
    error,
    refreshProjects,
    selectProject,
    createProject,
    updateProject,
    deleteProject
  }), [
    projects,
    selectedProject,
    selectedProjectId,
    loading,
    error,
    refreshProjects,
    selectProject,
    createProject,
    updateProject,
    deleteProject
  ]);

  return (
    <ProjectContext.Provider value={contextValue}>
      {children}
    </ProjectContext.Provider>
  );
};

export const useProjectContext = () => {
  const context = useContext(ProjectContext);
  if (!context) {
    throw new Error('useProjectContext must be used within a ProjectProvider');
  }
  return context;
};

