import { resolve } from 'node:path';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  resolve: {
    alias: [
      { find: /^@\/(.*)/, replacement: `${resolve(__dirname, 'src')}/$1` },
      { find: /^@@\/(.*)/, replacement: `${resolve(__dirname, 'src/.umi')}/$1` },
      { find: '@root', replacement: resolve(__dirname) },
    ],
  },
  test: {
    environment: 'happy-dom',
    globals: true,
    setupFiles: ['./tests/setupTests.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    // Exclude Umi integration tests that depend on @umijs/max test infrastructure
    // These require Umi's Jest runner and cannot be used with Vitest directly
    exclude: [
      'src/pages/user/login/login.test.tsx',
      'node_modules',
      'dist',
      '.umi',
    ],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/.umi/**',
        'src/services/ant-design-pro/**',
        'src/**/*.d.ts',
        'src/**/index.style.ts',
      ],
    },
    passWithNoTests: true,
    testTimeout: 15000,
  },
});
