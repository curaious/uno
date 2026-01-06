import React, { useState } from 'react';
import { Box, Button, Typography, Paper, Container, TextField, Divider, Alert, CircularProgress } from '@mui/material';
import { useAuth } from '../../contexts/AuthContext';
import styles from './Login.module.css';

const LoginPage: React.FC = () => {
  const { login, loginWithCredentials, auth0Enabled } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleCredentialLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !password) {
      setError('Please enter both email and password');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      await loginWithCredentials(email, password);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Invalid email or password');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container component="main" maxWidth="xs" className={styles.container}>
      <Paper elevation={3} className={styles.paper}>
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            padding: 4,
          }}
        >
          <img src="/logo192.png" alt="Uno Logo" className={styles.logo} />
          <Typography component="h1" variant="h5" sx={{ mt: 1, mb: 1, fontWeight: 'bold' }}>
            Welcome to Uno
          </Typography>
          <Typography variant="body2" align="center" sx={{ mb: 3, color: 'text.secondary' }}>
            Sign in to your account
          </Typography>

          {error && (
            <Alert severity="error" sx={{ width: '100%', mb: 2 }}>
              {error}
            </Alert>
          )}

          <Box component="form" onSubmit={handleCredentialLogin} sx={{ width: '100%' }}>
            <TextField
              margin="normal"
              required
              fullWidth
              id="email"
              label="Email"
              name="email"
              type="email"
              autoComplete="email"
              autoFocus
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={loading}
              variant="outlined"
              size="small"
            />
            <TextField
              margin="normal"
              required
              fullWidth
              name="password"
              label="Password"
              type="password"
              id="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={loading}
              variant="outlined"
              size="small"
            />
            <Button
              type="submit"
              fullWidth
              variant="contained"
              color="primary"
              disabled={loading}
              sx={{ mt: 2, mb: 2, py: 1, textTransform: 'none', fontWeight: 'bold' }}
            >
              {loading ? <CircularProgress size={24} color="inherit" /> : 'Login'}
            </Button>
          </Box>

          {auth0Enabled && (
            <>
              <Box sx={{ width: '100%', my: 2, display: 'flex', alignItems: 'center' }}>
                <Divider sx={{ flexGrow: 1 }} />
                <Typography variant="body2" sx={{ px: 2, color: 'text.secondary' }}>
                  OR
                </Typography>
                <Divider sx={{ flexGrow: 1 }} />
              </Box>

              <Button
                fullWidth
                variant="outlined"
                onClick={login}
                disabled={loading}
                sx={{ 
                  py: 1,
                  textTransform: 'none',
                  borderRadius: 1,
                  borderColor: 'rgba(255, 255, 255, 0.23)',
                  color: '#fff',
                  '&:hover': {
                    borderColor: '#fff',
                    backgroundColor: 'rgba(255, 255, 255, 0.05)'
                  }
                }}
              >
                Sign In with SSO
              </Button>
            </>
          )}

          <Typography variant="body2" align="center" sx={{ mt: 3, color: 'text.secondary', fontSize: '0.75rem' }}>
            © {new Date().getFullYear()} Amagi. All rights reserved.
          </Typography>
        </Box>
      </Paper>
    </Container>
  );
};

export default LoginPage;
