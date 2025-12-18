import React from 'react';
import {BrowserRouter} from "react-router";
import CssBaseline from '@mui/material/CssBaseline';

import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';
import {createTheme, ThemeProvider} from "@mui/material";

import styles from './App.module.css';
import {Sidebar} from "./components/Sidebar/Sidebar";
import {MainContent} from "./components/MainContent/MainContent";
import {ProjectProvider, useProjectContext} from "./contexts/ProjectContext";
import {AppProvider, useAppContext} from "./contexts/AppContext";
import {MiniSidebar} from "./components/MiniSidebar/MiniSidebar";
import {NoProjects} from "./components/NoProjects/NoProjects";

const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#10a37f',
      dark: '#10a37f',
      light: '#10a37f',
    },
    secondary: {
      main: '#d7e1da',
      dark: '#d7e1da',
      light: '#d7e1da',
    },
    background: {
      default: '#121212',
      paper: '#171719',
    },

    action: {
      hover: '#212124',
      active: '#252529',
    },
    common: {
      black: '#000',
      white: '#fff',
    },
    error: {
      main: '#7c1913',
    }
  },
  shape: {
    borderRadius: 6,
  },
  typography: {
    h1: {
      fontSize: '1.2rem',
      fontWeight: 600,
      letterSpacing: '0.0075em',
      lineHeight: 1.2,
    },
    h2: {
      fontSize: '1.06rem',
      fontWeight: 500,
      letterSpacing: '0.0075em',
      lineHeight: 1.2,
    }
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
        }
      }
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
        }
      }
    },
    MuiIconButton: {
      styleOverrides: {
        root: {
          color: '#fff'
        }
      }
    },
    MuiFormLabel: {
      styleOverrides: {
        root: {
          color: '#fff',
          fontSize: 13,
          fontWeight: 500,
          letterSpacing: '0.0075em',
          lineHeight: 1.2,
        }
      }
    },
    MuiFormControlLabel: {
      styleOverrides: {
        label: {
          color: '#fff',
          fontSize: 14,
          fontWeight: 400,
          letterSpacing: '0.0075em',
          lineHeight: 1.2,
        }
      }
    },
    MuiFormHelperText: {
      styleOverrides: {
        root: {
          marginLeft: 2,
          marginTop: 2,
        }
      }
    }
  }
});

const AppContent: React.FC = () => {
  const { selectedApp } = useAppContext();
  const { projects, loading } = useProjectContext();
  
  // For agent-framework, hide sidebar if no projects exist
  const shouldShowSidebar = selectedApp === 'llm-gateway' || (selectedApp === 'agent-framework' && projects.length > 0);
  const shouldShowNoProjects = selectedApp === 'agent-framework' && !loading && projects.length === 0;

  if (shouldShowNoProjects) {
    return (
      <div className={styles.root}>
        <CssBaseline/>
        <div className={styles.content}>
          <MiniSidebar/>
          <NoProjects/>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.root}>
      <CssBaseline/>
      <div className={styles.content}>
        <MiniSidebar/>
        {shouldShowSidebar && <Sidebar/>}
        <MainContent/>
      </div>
    </div>
  );
};

function App() {
  return (
    <ThemeProvider theme={darkTheme}>
      <AppProvider>
        <ProjectProvider>
          <BrowserRouter>
            <AppContent/>
          </BrowserRouter>
        </ProjectProvider>
      </AppProvider>
    </ThemeProvider>
  );
}

export default App;
