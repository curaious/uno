import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { api } from '../api';

interface User {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface AuthContextType {
  isAuthenticated: boolean;
  loading: boolean;
  user: User | null;
  auth0Enabled: boolean;
  login: () => void;
  loginWithCredentials: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);
  const [user, setUser] = useState<User | null>(null);
  const [auth0Enabled, setAuth0Enabled] = useState<boolean>(false);

  const checkAuthStatus = async () => {
    setLoading(true);
    try {
      // Check if auth is enabled and get auth0 status
      const authEnabledResponse = await api.get('/auth/enabled');
      setAuth0Enabled(authEnabledResponse.data?.data?.auth0_enabled ?? false);

      // Try to get current user info (this will fail if not authenticated)
      const userResponse = await api.get('/auth/me');
      if (userResponse.data?.data) {
        setUser(userResponse.data.data);
        setIsAuthenticated(true);
      }
    } catch (error: any) {
      // Not authenticated
      setIsAuthenticated(false);
      setUser(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    checkAuthStatus();
  }, []);

  const login = () => {
    // Redirect to Auth0 login
    window.location.href = 'http://localhost:6060/api/auth/login';
  };

  const loginWithCredentials = async (email: string, password: string) => {
    const response = await api.post('/auth/login', { email, password });
    if (response.data?.data) {
      setUser(response.data.data.user);
      setIsAuthenticated(true);
      // After successful login, redirect to home
      window.location.href = '/';
    }
  };

  const logout = async () => {
    try {
      await api.post('/auth/logout');
    } catch (error) {
      console.error('Logout failed:', error);
    } finally {
      setIsAuthenticated(false);
      setUser(null);
      window.location.href = '/login';
    }
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated, loading, user, auth0Enabled, login, loginWithCredentials, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

