import { execSync } from 'node:child_process';
import { fileURLToPath, URL as NodeURL } from 'node:url';
import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import { getProxyConfig } from './config/proxy';

const webRoot = fileURLToPath(new NodeURL('.', import.meta.url));

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

export default defineConfig(({ mode }) => ({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: [
      { find: /^@\/(.*)/, replacement: `${webRoot}src/$1` },
      { find: '@root', replacement: webRoot },
    ],
  },
  // VITE_ 前缀的环境变量（如部署注入的 VITE_SENTRY_DSN）由 Vite 自动注入
  // import.meta.env；此处仅补充运行期才能计算的注入项。环境语义走 mode
  // （--mode dev/test/pre，生产构建默认 production），经 import.meta.env.MODE
  // 消费。NODE_ENV 由 Vite 内建替换，勿在此覆盖。
  define: {
    'import.meta.env.VITE_COMMIT_HASH': JSON.stringify(resolveCommitHash()),
  },
  css: {
    preprocessorOptions: {
      less: { javascriptEnabled: true },
    },
  },
  server: {
    host: '0.0.0.0',
    port: 8001,
    // 生产构建不挂开发代理；vitest（process.env.VITEST）虽默认 mode=test，
    // 但不启动开发服务器，跳过 fail-fast 校验。
    ...(mode === 'production' || process.env.VITEST
      ? {}
      : { proxy: getProxyConfig(mode) }),
  },
  preview: {
    port: 8000,
  },
  build: {
    outDir: 'dist',
    sourcemap: mode === 'production',
  },
  // test 段自 vitest.config.ts 平移（@@/.umi 痕迹别名已去除）；原先排除的
  // login.test.tsx 已在早期提交删除（登录页测试由 user/login/index.test.tsx 覆盖）。
  test: {
    environment: 'happy-dom',
    globals: true,
    setupFiles: ['./tests/setupTests.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    fileParallelism: true,
    maxConcurrency: 32,
    exclude: ['node_modules', 'dist'],
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
