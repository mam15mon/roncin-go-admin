import { spawnSync } from 'node:child_process';
import { writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

// 依据 server/internal/access/manifest.go 生成 web/src/permissions.generated.ts，
// 让前端权限键名与后端权限清单在编译期对齐：后端权限码改名或删除后，若前端
// access.ts 仍引用旧键名，`pnpm --dir web tsc` 会直接失败。
const repoRoot = join(dirname(fileURLToPath(import.meta.url)), '..');

const exported = spawnSync('go', ['-C', 'server', 'run', './cmd/export-permission-manifest'], {
  cwd: repoRoot,
  encoding: 'utf8',
});
if (exported.error || exported.status !== 0) {
  console.error('导出权限清单失败:', exported.error ?? exported.stderr);
  process.exit(1);
}

const keys = JSON.parse(exported.stdout);
if (!Array.isArray(keys) || keys.length === 0 || keys.some((key) => typeof key !== 'string')) {
  console.error('导出的权限清单格式异常');
  process.exit(1);
}

const content = `// 自动生成：依据 server/internal/access/manifest.go 生成，请勿手工修改。
// 再生成命令：pnpm run generate:permission-keys
export const manifestPermissionKeys = [
${keys.map((key) => `  '${key}',`).join('\n')}
] as const;

export type ManifestPermissionKey = (typeof manifestPermissionKeys)[number];
`;

const outputPath = join(repoRoot, 'web/src/permissions.generated.ts');
writeFileSync(outputPath, content);
console.log(`[generate-permission-keys] 已生成 ${keys.length} 个权限键名常量`);
