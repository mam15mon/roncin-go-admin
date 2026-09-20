// 按需运行的疑似重复报告；有候选不阻断 CI，解析/执行错误必须失败。
import { createHash } from 'node:crypto';
import { readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { extractFunctions } from '../web/scripts/duplicate-functions.mjs';

const repository = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const excludedDirectories = new Set([
  'node_modules',
  'vendor',
  'dist',
  'build',
  'coverage',
  '.git',
  '.cache',
  'bin',
  '__tests__',
  '__mocks__',
  'tests',
  'testdata',
]);
const generated =
  /(?:Code generated .*DO NOT EDIT|@generated|此文件.*自动生成|This file is auto-generated)/i;
export const limitations = [
  '仅同语言完整函数结构匹配，不能证明业务等价；零候选不代表没有业务重复。',
  '保留字面量、操作符、外部符号、属性名及类型；不解析跨文件别名或跨语言语义。',
  'JS 解构/默认/剩余参数、var、嵌套函数声明、类、JSX、eval、switch、计算方法名等复杂绑定只参与 exact；Go 局部类型和匿名结构体只参与 exact。',
  'Go 覆盖函数和方法声明（包括 Ent schema），不单独报告匿名函数；其结构仍参与外层函数比较。',
];
function compare(a, b) {
  return a < b ? -1 : a > b ? 1 : 0;
}
export function collectSources(root) {
  const sources = { javascript: [], go: [] };
  const statistics = {
    scannedFiles: { javascript: 0, go: 0 },
    excludedFiles: {},
    excludedDirectories: 0,
  };
  const errors = [];
  const exclude = (reason) => {
    statistics.excludedFiles[reason] =
      (statistics.excludedFiles[reason] ?? 0) + 1;
  };
  function walk(directory, language) {
    for (const entry of readdirSync(directory, { withFileTypes: true }).sort(
      (a, b) => compare(a.name, b.name),
    )) {
      const absolute = join(directory, entry.name);
      const path = relative(root, absolute).replaceAll('\\', '/');
      if (entry.isDirectory()) {
        if (
          excludedDirectories.has(entry.name) ||
          entry.name.startsWith('.umi') ||
          path === 'web/src/services/roncin'
        ) {
          statistics.excludedDirectories++;
          continue;
        }
        walk(absolute, language);
        continue;
      }
      if (!entry.isFile()) {
        exclude('非普通文件');
        continue;
      }
      if (!(language === 'go' ? /\.go$/ : /\.[jt]sx?$/).test(entry.name)) {
        exclude('非目标源码');
        continue;
      }
      if (/(?:\.(?:test|spec)\.[jt]sx?|_test\.go)$/.test(entry.name)) {
        exclude('测试');
        continue;
      }
      if (
        /(?:\.d\.ts|\.generated\.[jt]sx?|\.pb\.go|_grpc\.pb\.go|_http\.pb\.go|wire_gen\.go)$/.test(
          entry.name,
        )
      ) {
        exclude('生成后缀或类型声明');
        continue;
      }
      try {
        const source = readFileSync(absolute, 'utf8');
        if (generated.test(source.slice(0, 4096))) {
          exclude('生成标记');
          continue;
        }
        sources[language].push({ path, source });
        statistics.scannedFiles[language]++;
      } catch (error) {
        errors.push({ path, message: error.message });
      }
    }
  }
  for (const [directory, language] of [
    ['web/src', 'javascript'],
    ['server', 'go'],
  ]) {
    try {
      walk(join(root, directory), language);
    } catch (error) {
      errors.push({ path: directory, message: error.message });
    }
  }
  return { sources, statistics, errors };
}
function publicRecord({ exact, renamed, ...record }) {
  return record;
}
export function groupFunctions(records, minNodes = 60) {
  const groups = [];
  const candidates = records.filter((record) => record.nodes >= minNodes);
  // 哈希只负责分桶，桶内用完整规范化内容再次分组，避免哈希碰撞。
  for (const mode of ['exact', 'renamed']) {
    const buckets = new Map();
    for (const record of candidates) {
      if (record[mode] == null) continue;
      const hash = createHash('sha256').update(record[mode]).digest('hex');
      const key = `${record.language}:${hash}`;
      if (!buckets.has(key)) buckets.set(key, new Map());
      const bucket = buckets.get(key);
      if (!bucket.has(record[mode])) bucket.set(record[mode], []);
      bucket.get(record[mode]).push(record);
    }
    for (const [key, bucket] of buckets)
      for (const members of bucket.values()) {
        if (
          members.length < 2 ||
          (mode === 'renamed' &&
            new Set(members.map((item) => item.exact)).size === 1)
        )
          continue;
        const sorted = members.sort(
          (a, b) => compare(a.path, b.path) || a.start - b.start,
        );
        groups.push({
          language: members[0].language,
          mode,
          fingerprint: key.split(':')[1],
          nodes: members[0].nodes,
          coveredNodes: members.reduce((sum, item) => sum + item.nodes, 0),
          fileCount: new Set(members.map((item) => item.path)).size,
          evidence:
            mode === 'exact'
              ? '忽略位置、格式、注释及声明名称后的 AST 相同'
              : '词法绑定参数与局部变量改名后的 AST 相同（保留外部符号及属性）',
          members: sorted.map(publicRecord),
        });
      }
  }
  groups.sort(
    (a, b) =>
      b.coveredNodes - a.coveredNodes ||
      b.fileCount - a.fileCount ||
      compare(a.members[0].path, b.members[0].path) ||
      a.members[0].start - b.members[0].start ||
      compare(a.mode, b.mode),
  );
  // 外层完整函数已经覆盖所有成员时，省略其内的重复回调组。
  return groups.filter(
    (group, index) =>
      !groups.some(
        (outer, outerIndex) =>
          outerIndex !== index &&
          outer.nodes > group.nodes &&
          group.members.every((member) =>
            outer.members.some(
              (parent) =>
                parent.path === member.path &&
                parent.start <= member.start &&
                parent.end >= member.end,
            ),
          ),
      ),
  );
}
export function renderText(report) {
  const lines = [
    `疑似重复函数报告（最少 ${report.minNodes} 个 AST 节点）`,
    `覆盖：JS/TS ${report.statistics.scannedFiles.javascript} 文件，Go ${report.statistics.scannedFiles.go} 文件；${report.statistics.functions} 个函数，${report.statistics.eligibleFunctions} 个达到阈值；${report.groups.length} 组候选。`,
    `排除：${JSON.stringify(report.statistics.excludedFiles)}；跳过 ${report.statistics.excludedDirectories} 个目录。`,
    `仅 exact 的函数：${report.statistics.exactOnlyFunctions}。`,
    ...report.limitations,
  ];
  for (const [index, group] of report.groups.entries()) {
    lines.push(
      `\n${index + 1}. [${group.language}/${group.mode}] ${group.nodes} 节点 × ${group.members.length} 处：${group.evidence}`,
    );
    for (const member of group.members)
      lines.push(
        `  ${member.path}:${member.startLine}-${member.endLine} ${member.symbol}`,
      );
  }
  for (const error of report.errors)
    lines.push(`错误 ${error.path}: ${error.message}`);
  return `${lines.join('\n')}\n`;
}
export function scan(root = repository, minNodes = 60) {
  const { sources, statistics, errors } = collectSources(root);
  const records = [];
  for (const input of sources.javascript) {
    try {
      records.push(...extractFunctions(input.source, input.path));
    } catch (error) {
      errors.push({ path: input.path, message: error.message });
    }
  }
  if (sources.go.length) {
    const child = spawnSync(
      'go',
      ['-C', join(repository, 'scripts/duplicate-scan-go'), 'run', '.'],
      {
        input: JSON.stringify(sources.go),
        encoding: 'utf8',
        maxBuffer: 256 * 1024 * 1024,
      },
    );
    if (child.stdout) {
      try {
        const output = JSON.parse(child.stdout);
        records.push(...output.records);
        errors.push(...output.errors);
      } catch (error) {
        errors.push({
          path: 'Go 扫描器',
          message: `输出无效：${error.message}`,
        });
      }
    }
    if (child.error || child.status !== 0)
      errors.push({
        path: 'Go 扫描器',
        message:
          child.error?.message ??
          child.stderr.trim() ??
          `退出码 ${child.status}`,
      });
  }
  statistics.functions = records.length;
  statistics.functionsByLanguage = {
    javascript: records.filter((item) => item.language === 'javascript').length,
    go: records.filter((item) => item.language === 'go').length,
  };
  statistics.eligibleFunctions = records.filter(
    (item) => item.nodes >= minNodes,
  ).length;
  statistics.exactOnlyFunctions = records.filter(
    (item) => item.renameUnsupported,
  ).length;
  return {
    version: 1,
    modes: ['exact', 'renamed'],
    minNodes,
    roots: ['web/src', 'server'],
    statistics,
    limitations,
    exactOnly: records
      .filter((item) => item.renameUnsupported)
      .map(publicRecord),
    groups: groupFunctions(records, minNodes),
    errors,
  };
}
export function main(args = process.argv.slice(2)) {
  let format = 'text';
  let minNodes = 60;
  let output;
  for (let index = 0; index < args.length; index++) {
    const argument = args[index];
    if (argument === '--format') format = args[++index];
    else if (argument === '--min-nodes') minNodes = Number(args[++index]);
    else if (argument === '--output') {
      output = args[++index];
      if (!output) throw new Error('--output 缺少路径');
    } else if (argument === '--help') {
      process.stdout.write(
        'pnpm run report:duplicates [--format text|json] [--min-nodes 60] [--output 路径]\n',
      );
      return;
    } else throw new Error(`未知参数：${argument}`);
  }
  if (
    !['text', 'json'].includes(format) ||
    !Number.isSafeInteger(minNodes) ||
    minNodes < 1
  )
    throw new Error('format 必须为 text/json，min-nodes 必须为正整数');
  const report = scan(repository, minNodes);
  const rendered =
    format === 'json'
      ? `${JSON.stringify(report, null, 2)}\n`
      : renderText(report);
  if (output) writeFileSync(resolve(output), rendered);
  else process.stdout.write(rendered);
  if (report.errors.length) {
    console.error(
      `重复扫描失败：${report.errors.length} 个错误${output ? `，详情见 ${output}` : ''}`,
    );
    process.exitCode = 1;
  }
}
if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  try {
    main();
  } catch (error) {
    console.error(`重复扫描失败：${error.message}`);
    process.exitCode = 1;
  }
}
