import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}'],
    exclude: ['node_modules', 'dist', '.idea', '.git', '.cache'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      reportsDirectory: './coverage',
      exclude: [
        'node_modules/',
        'src/test/',
        '**/*.d.ts',
        '**/*.config.*',
        '**/index.ts',
        'src/main.tsx',
        'src/vite-env.d.ts',
      ],
      thresholds: {
        statements: 50,
        branches: 50,
        functions: 50,
        lines: 50,
      }
    },
    // Reporter options
    reporters: ['default', 'html'],
    // Timeout for async tests
    testTimeout: 10000,
    // Hook timeout
    hookTimeout: 10000,
    // Retry failed tests
    retry: 0,
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})

