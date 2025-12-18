import React, {
  createContext,
  ReactNode,
  useContext,
  useState,
  useCallback,
  useMemo
} from 'react';

export type AppType = 'llm-gateway' | 'agent-framework';

const STORAGE_KEY = 'app.selectedApp';

interface AppContextValue {
  selectedApp: AppType;
  setSelectedApp: (app: AppType) => void;
}

const AppContext = createContext<AppContextValue | undefined>(undefined);

interface AppProviderProps {
  children: ReactNode;
}

export const AppProvider: React.FC<AppProviderProps> = ({ children }) => {
  const [selectedApp, setSelectedAppState] = useState<AppType>(() => {
    if (typeof window === 'undefined') {
      return 'llm-gateway';
    }
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      return (stored === 'llm-gateway' || stored === 'agent-framework') ? stored : 'llm-gateway';
    } catch (err) {
      console.warn('Unable to read selected app from storage', err);
      return 'llm-gateway';
    }
  });

  const setSelectedApp = useCallback((app: AppType) => {
    setSelectedAppState(app);
    try {
      localStorage.setItem(STORAGE_KEY, app);
    } catch (err) {
      console.warn('Unable to persist selected app', err);
    }
  }, []);

  const contextValue = useMemo<AppContextValue>(() => ({
    selectedApp,
    setSelectedApp
  }), [selectedApp, setSelectedApp]);

  return (
    <AppContext.Provider value={contextValue}>
      {children}
    </AppContext.Provider>
  );
};

export const useAppContext = () => {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('useAppContext must be used within an AppProvider');
  }
  return context;
};


