import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      // react-transition-group ships CJS only; Vitest's ESM resolver chokes on
      // bare-directory imports that MUI makes. Point directly at the CJS files.
      'react-transition-group/TransitionGroupContext':
        'react-transition-group/cjs/TransitionGroupContext.js',
      'react-transition-group/Transition':
        'react-transition-group/cjs/Transition.js',
      'react-transition-group/CSSTransition':
        'react-transition-group/cjs/CSSTransition.js',
      'react-transition-group/TransitionGroup':
        'react-transition-group/cjs/TransitionGroup.js',
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    server: {
      deps: {
        // Force @mui/material and react-transition-group through Vite's bundler
        // so resolve.alias remaps the bare-directory imports that Node ESM can't handle.
        inline: ['@mui/material', 'react-transition-group', '@mui/system', '@mui/utils'],
      },
    },
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      thresholds: {
        lines: 80,
        functions: 80,
        branches: 80,
        statements: 80,
      },
    },
  },
})
