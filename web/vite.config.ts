import { execSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { fileURLToPath, URL as NodeURL } from 'node:url';
import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import { getProxyConfig } from './config/proxy';

const webRoot = fileURLToPath(new NodeURL('.', import.meta.url));

// UMI_ENV 沿用既有环境命名（dev/test/pre/prod），仅由构建配置与 Sentry 消费。
const { UMI_ENV = 'dev' } = process.env;

// 计算提交哈希：环境变量优先，缺省回退 git；两者皆空时 Sentry release 为 undefined。
function resolveCommitHash(): string {
  if (process.env.COMMIT_HASH) return process.env.COMMIT_HASH;
  try {
    return execSync('git rev-parse HEAD', {
      stdio: ['ignore', 'pipe', 'ignore'],
      encoding: 'utf8',
    }).trim();
  } catch {
    return '';
  }
}

const pkg = JSON.parse(
  readFileSync(new NodeURL('./package.json', import.meta.url), 'utf8'),
) as { version: string };

export default defineConfig(({ mode }) => ({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: [
      { find: /^@\/(.*)/, replacement: `${webRoot}src/$1` },
      { find: '@root', replacement: webRoot },
    ],
  },
  // define 仅注入实测存在的读取键（见任务 design.md D6）；
  // NODE_ENV 由 Vite 内建替换，勿在此覆盖。
  define: {
    'process.env.CI': JSON.stringify(process.env.CI ?? ''),
    'process.env.COMMIT_HASH': JSON.stringify(resolveCommitHash()),
    'process.env.UMI_ENV': JSON.stringify(UMI_ENV),
    'process.env.SENTRY_DSN': JSON.stringify(process.env.SENTRY_DSN ?? ''),
    __APP_VERSION__: JSON.stringify(pkg.version),
  },
  css: {
    preprocessorOptions: {
      less: { javascriptEnabled: true },
    },
  },
  server: {
    host: '0.0.0.0',
    port: 8001,
    proxy: getProxyConfig(UMI_ENV),
  },
  preview: {
    port: 8000,
  },
  build: {
    outDir: 'dist',
    sourcemap: mode === 'production',
  },
  // test 段自 vitest.config.ts 平移（@@/.umi 痕迹别名已去除），
  // 阶段 4 恢复被排除的 login.test.tsx 后同步删除 exclude 中的对应项。
  test: {
    environment: 'happy-dom',
    globals: true,
    setupFiles: ['./tests/setupTests.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    fileParallelism: true,
    maxConcurrency: 32,
    exclude: [
      'src/pages/user/login/login.test.tsx',
      'node_modules',
      'dist',
    ],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/services/ant-design-pro/**',
        'src/**/*.d.ts',
        'src/**/index.style.ts',
      ],
    },
    passWithNoTests: true,
    testTimeout: 30000,
  },
}));
