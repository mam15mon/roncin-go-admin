import { execSync, spawnSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const distDirectory = join(repositoryRoot, 'web', 'dist');

// Resolve commit hash to match release configured in web/src/sentry.ts
const commitHash =
  process.env.COMMIT_HASH ||
  process.env.CF_PAGES_COMMIT_SHA ||
  (() => {
    try {
      return execSync('git rev-parse HEAD', {
        cwd: repositoryRoot,
        encoding: 'utf-8',
        stdio: ['ignore', 'pipe', 'ignore'],
      }).trim();
    } catch {
      return '';
    }
  })();

const authToken = process.env.SENTRY_AUTH_TOKEN;
const org = process.env.SENTRY_ORG;
const project = process.env.SENTRY_PROJECT;

if (!authToken) {
  console.log(
    '[sentry] SENTRY_AUTH_TOKEN 未配置，跳过 SourceMap 上传（本地开发与测试默认跳过，不影响构建）。',
  );
  process.exit(0);
}

if (!org || !project) {
  console.warn(
    '[sentry] 检测到 SENTRY_AUTH_TOKEN 但未配置 SENTRY_ORG 或 SENTRY_PROJECT，跳过 SourceMap 上传。',
  );
  process.exit(0);
}

if (!commitHash) {
  console.warn('[sentry] 无法获取 COMMIT_HASH，跳过 SourceMap 上传。');
  process.exit(0);
}

if (!existsSync(distDirectory)) {
  console.error(`[sentry] 前端构建目录不存在：${distDirectory}`);
  process.exit(1);
}

console.log(
  `[sentry] 开始上传 SourceMap 到 Sentry (Org: ${org}, Project: ${project}, Release: ${commitHash})...`,
);

const result = spawnSync(
  'pnpm',
  [
    'exec',
    'sentry-cli',
    'sourcemaps',
    'upload',
    '--release',
    commitHash,
    '--url-prefix',
    '~/',
    distDirectory,
  ],
  {
    cwd: repositoryRoot,
    stdio: 'inherit',
    env: process.env,
  },
);

if (result.status !== 0) {
  console.error('[sentry] SourceMap 上传失败。');
  process.exit(result.status || 1);
}

console.log('[sentry] SourceMap 上传完成，Sentry 堆栈将自动解析为原始文件名与代码行号。');
