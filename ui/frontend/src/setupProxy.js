const { createProxyMiddleware } = require('http-proxy-middleware');

module.exports = function(app) {
  // SSE endpoint - needs special handling to disable buffering
  app.use(
    '/api/agent-server/converse',
    createProxyMiddleware({
      target: 'http://127.0.0.1:6060',
      changeOrigin: true,
      // Critical for SSE streaming
      onProxyRes: (proxyRes) => {
        // Disable buffering for SSE
        proxyRes.headers['X-Accel-Buffering'] = 'no';
        proxyRes.headers['Cache-Control'] = 'no-cache';
        proxyRes.headers['Connection'] = 'keep-alive';
      },
    })
  );

  // All other API requests
  app.use(
    '/api',
    createProxyMiddleware({
      target: 'http://127.0.0.1:6060',
      changeOrigin: true,
    })
  );
};
