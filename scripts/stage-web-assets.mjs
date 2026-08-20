import { cp, readdir, rm, stat } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const sourceDirectory = join(repositoryRoot, 'web', 'dist');
const targetDirectory = join(
  repositoryRoot,
  'server',
  'internal',
  'webassets',
  'dist',
);

const sourceInfo = await stat(sourceDirectory);
if (!sourceInfo.isDirectory()) {
  throw new Error(`前端构建产物目录无效：${sourceDirectory}`);
}

for (const entry of await readdir(targetDirectory, { withFileTypes: true })) {
  if (entry.name === '.gitkeep') {
    continue;
  }
  await rm(join(targetDirectory, entry.name), { recursive: true, force: true });
}

for (const entry of await readdir(sourceDirectory, { withFileTypes: true })) {
  await cp(join(sourceDirectory, entry.name), join(targetDirectory, entry.name), {
    recursive: true,
    force: true,
  });
}
