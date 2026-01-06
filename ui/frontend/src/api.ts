import axios, {AxiosResponse, InternalAxiosRequestConfig} from 'axios';
import {STORAGE_KEY} from "./contexts/ProjectContext";

export const api = axios.create({
  baseURL: 'http://localhost:6060/api/agent-server',
  withCredentials: true,
});

// Request interceptor to automatically add project_id query parameter
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
		// Skip adding project_id for project-independent endpoints
		const url = config.url || '';
		const projectIndependentEndpoints = [
			'/projects',
			'/providers/models',
			'/api-keys',
			'/virtual-keys'
		];
		
		if (projectIndependentEndpoints.some(endpoint => url.startsWith(endpoint))) {
			return config;
		}

    // Get project_id from localStorage
    try {
      const projectId = localStorage.getItem(STORAGE_KEY);
      if (projectId) {
        // Add project_id as query parameter
        config.params = {
          ...config.params,
          project_id: projectId
        };
      }
    } catch (err) {
      // If localStorage access fails, continue without project_id
      console.warn('Unable to read project_id from storage', err);
    }

    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

export interface Response<T> {
  data: T;
}