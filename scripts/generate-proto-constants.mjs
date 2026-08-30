import { readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), '..');
const apiRoot = join(repoRoot, 'server/api');

function listProtoFiles(directory) {
  return readdirSync(directory, { withFileTypes: true })
    .flatMap((entry) => {
      const path = join(directory, entry.name);
      return entry.isDirectory() ? listProtoFiles(path) : entry.name.endsWith('.proto') ? [path] : [];
    })
    .sort();
}

function parseProtoEnums(path) {
  const source = readFileSync(path, 'utf8')
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/\/\/.*$/gm, '');
  const packageName = source.match(/^\s*package\s+([\w.]+)\s*;/m)?.[1];
  if (!packageName) {
    throw new Error(`${relative(repoRoot, path)} 缺少 package 声明`);
  }
  const enums = [];
  for (const match of source.matchAll(/\benum\s+(\w+)\s*\{([\s\S]*?)\}/g)) {
    const values = [...match[2].matchAll(/^\s*([A-Z][A-Z0-9_]*)\s*=\s*(-?\d+)\b/gm)].map(
      (value) => ({ name: value[1], number: Number(value[2]) }),
    );
    if (values.length > 0) {
      enums.push({ name: match[1], packageName, path, values });
    }
  }
  return enums;
}

const parsedEnums = listProtoFiles(apiRoot).flatMap(parseProtoEnums);
const regularEnums = parsedEnums.filter((item) => item.name !== 'ErrorReason');
const nameCounts = new Map();
for (const item of regularEnums) {
  nameCounts.set(item.name, (nameCounts.get(item.name) ?? 0) + 1);
}

function upperCamel(value) {
  return value
    .split(/[^a-zA-Z0-9]+/)
    .filter(Boolean)
    .map((part) => `${part[0].toUpperCase()}${part.slice(1)}`)
    .join('');
}

for (const item of regularEnums) {
  item.exportName =
    nameCounts.get(item.name) === 1
      ? item.name
      : `${upperCamel(item.packageName.split('.')[0])}${item.name}`;
}

const enumContent = `// 自动生成：依据 server/api/**/*.proto 生成，请勿手工修改。
// 再生成命令：pnpm run generate:proto-constants
${regularEnums
  .map(
    (item) => `export const ${item.exportName} = {
${item.values.map((value) => `  ${value.name}: ${value.number},`).join('\n')}
} as const;

export type ${item.exportName} = (typeof ${item.exportName})[keyof typeof ${item.exportName}];`,
  )
  .join('\n\n')}
`;

function lowerCamel(value) {
  return value.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
}

const errorGroups = new Map();
for (const item of parsedEnums.filter((enumItem) => enumItem.name === 'ErrorReason')) {
  const domain = lowerCamel(item.packageName.split('.')[0]);
  if (errorGroups.has(domain)) {
    throw new Error(`错误原因域 ${domain} 存在多个 ErrorReason 枚举`);
  }
  errorGroups.set(domain, item.values);
}

const errorReasonContent = `// 自动生成：依据 server/api/**/error_reason.proto 生成，请勿手工修改。
// 再生成命令：pnpm run generate:proto-constants
${[...errorGroups.entries()]
  .map(([domain, values]) => {
    const exportName = `${domain}ErrorReasons`;
    const typeName = `${domain[0].toUpperCase()}${domain.slice(1)}ErrorReason`;
    return `export const ${exportName} = {
${values
  .filter((value) => !value.name.endsWith('_UNSPECIFIED'))
  .map((value) => {
    const reason = value.name.replace(/^ERROR_REASON_/, '');
    return `  ${reason}: '${reason}',`;
  })
  .join('\n')}
} as const;

export type ${typeName} = (typeof ${exportName})[keyof typeof ${exportName}];`;
  })
  .join('\n\n')}
`;

const outputs = [
  [join(repoRoot, 'web/src/enums.generated.ts'), enumContent],
  [join(repoRoot, 'web/src/errorReasons.generated.ts'), errorReasonContent],
];

if (process.argv.includes('--check')) {
  let stale = false;
  for (const [path, content] of outputs) {
    let current = '';
    try {
      current = readFileSync(path, 'utf8');
    } catch {
      stale = true;
      continue;
    }
    stale ||= current !== content;
  }
  if (stale) {
    console.error('Proto 常量生成物已过期，请执行 pnpm run generate:proto-constants');
    process.exit(1);
  }
  console.log(`[generate-proto-constants] 已验证 ${regularEnums.length} 个枚举和 ${errorGroups.size} 个错误原因域`);
  process.exit(0);
}

for (const [path, content] of outputs) {
  writeFileSync(path, content);
}
console.log(`[generate-proto-constants] 已生成 ${regularEnums.length} 个枚举和 ${errorGroups.size} 个错误原因域`);
