import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  testMatch: '**/*.e2e.ts',
  timeout: 60_000,
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list']],
  // 验收编排以 max dev 开发态服务承载 e2e，页面按需编译存在秒级首屏延迟；
  // 断言等待窗放宽到 15s，避免把开发态编译抖动误判为页面缺陷。
  expect: { timeout: 15_000 },
  use: {
    baseURL: process.env.RONCIN_WEB_BASE_URL || 'http://127.0.0.1:8001',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    ...devices['Desktop Chrome'],
  },
});
