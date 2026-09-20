// 前端依赖边界扫描器（任务 design.md 第 4 节的规范实现）。
//
// 解析方式：@babel/parser 逐文件解析 AST，覆盖静态 import、export ... from、
// import type / export type、字面量 import('...') 与 require('...')、以及
// TS 类型级 import('...') 引用。注释与普通字符串不构成依赖；非字面量动态
// 导入（import.meta.glob、变量拼接）不判定、不误报。
//
// 边界规则：
// 1. 不同页面模块禁止互相导入。页面模块 = pages/<领域>/；财务独立工作区细分为
//    pages/finance/<模块>/；pages 顶层单文件页面自成模块。
// 2. features 禁止导入 pages。
// 3. 外部引用 feature 只能走公共入口 features/<领域>/index.ts 或
//    features/<领域>/<能力>/index.ts；更深层目录的 index.ts 不自动成为公共
//    API；feature 内部模块互相引用不限制。
// 4. components/ui、hooks、utils、constants 通用层禁止导入 pages 与 features；
//    其他非页面业务代码禁止直接导入 pages；路由页面装载使用精确授权清单
//    （PAGES_ASSEMBLY_ALLOWLIST），不对 router 目录整体豁免。
// 5. features 领域/能力公共入口之间的依赖图存在有向环时报错。
// 6. node_modules 包依赖与指向 src 之外（web/config、web/tests 等平台/测试
//    区域）的引用不判定；越出 web 根目录的本地路径视为违规。
//
// 退出码：发现违规或执行失败返回非 0，通过返回 0。

import { readdirSync, readFileSync, statSync } from 'node:fs';
import { basename, dirname, extname, join, relative, resolve, sep } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import process from 'node:process';
import { parse } from '@babel/parser';

/** 参与扫描的手写源码扩展名（存在手写 JS 时同样处理）。 */
const SCAN_EXTENSIONS = new Set(['.ts', '.tsx', '.js', '.jsx']);

/** 目录名排除清单：生成缓存、依赖、构建产物与测试目录。 */
const EXCLUDED_DIR_NAMES = new Set([
  'node_modules',
  'dist',
  '.umi',
  '.umi-production',
  '.umi-test',
  '.umi-test-production',
  '__tests__',
  'tests',
]);

/** 解析无扩展名/目录引用时尝试的扩展名顺序。 */
const RESOLVE_EXTENSIONS = ['.ts', '.tsx', '.js', '.jsx'];

/** 通用基础层目录（相对 src，带斜杠前缀匹配），禁止反向导入 pages/features。 */
const GENERIC_DIR_PREFIXES = ['components/ui/', 'hooks/', 'utils/', 'constants/'];

/**
 * 路由页面装配的精确授权清单（相对 src 的 POSIX 路径）。
 * 清单外文件导入 pages 一律违规；新增装配入口必须显式追加到这里。
 */
export const PAGES_ASSEMBLY_ALLOWLIST = ['router/adaptRoutes.tsx'];

/** 违规规则标识（中文，供报告与测试断言使用）。 */
export const RULE = {
  PAGE_CROSS: '跨页面模块导入',
  FEATURE_TO_PAGES: 'features 反向依赖 pages',
  FEATURE_PRIVATE: '绕过 feature 公开入口',
  GENERIC_TO_PAGES: '通用层反向依赖 pages',
  GENERIC_TO_FEATURES: '通用层反向依赖 features',
  OTHER_TO_PAGES: '非页面业务代码导入 pages',
  FEATURE_CYCLE: 'features 能力依赖环',
  UNRESOLVED: '本地路径无法解析',
  ESCAPED: '本地路径越界',
  PARSE: '文件解析失败',
};

/** 统一为 POSIX 分隔符的相对路径。 */
export function toPosix(value) {
  return value.split(sep).join('/');
}

/** 测试文件不参与产品边界检查（可访问被测私有实现，也不建立任何豁免基线）。 */
function isTestFile(basenameValue) {
  return /(\.test|\.spec)\.[cm]?[jt]sx?$/.test(basenameValue);
}

/** 生成物不参与扫描：OpenAPI 生成客户端目录与 *.generated.* 文件。 */
function isGeneratedFile(relFromSrc) {
  if (relFromSrc.startsWith('services/roncin/')) return true;
  return /\.generated\.[cm]?[jt]sx?$/.test(basename(relFromSrc));
}

/** 是否为参与产品边界检查的文件。 */
export function isProductFile(relFromSrc) {
  if (isTestFile(basename(relFromSrc))) return false;
  return !isGeneratedFile(relFromSrc);
}

/** 递归收集 src 下参与扫描的源码文件（绝对路径，已排序）。 */
export function collectSourceFiles(srcRoot) {
  const files = [];
  const walk = (dir) => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      if (entry.isDirectory()) {
        if (EXCLUDED_DIR_NAMES.has(entry.name)) continue;
        walk(join(dir, entry.name));
        continue;
      }
      if (!entry.isFile() || !SCAN_EXTENSIONS.has(extname(entry.name))) continue;
      files.push(join(dir, entry.name));
    }
  };
  walk(srcRoot);
  return files.sort();
}

/**
 * 页面模块归属：pages/<领域>/ 为一个模块；财务细分为 pages/finance/<模块>/；
 * pages 顶层单文件页面自成模块（文件名即模块键）。
 */
export function pageModuleKey(relFromSrc) {
  const parts = relFromSrc.split('/');
  if (parts[0] !== 'pages') return relFromSrc;
  if (parts.length === 2) return relFromSrc;
  if (parts[1] === 'finance') return `pages/finance/${parts[2]}`;
  return `pages/${parts[1]}`;
}

/**
 * feature 公共单元归属：features/<领域>/ 下的文件（含领域入口 index.ts）属于
 * 领域单元；features/<领域>/<能力>/ 及其更深层的文件属于能力单元。公共入口
 * 固定为单元根目录的 index.ts；能力内部更深层目录的 index.ts 不自动成为
 * 公共 API（如 features/<领域>/<能力>/<子目录>/index.ts）。
 */
export function featureOwnerUnit(relFromSrc) {
  const parts = relFromSrc.split('/');
  if (parts[0] !== 'features' || parts.length < 3) return null;
  if (parts.length === 3) return `features/${parts[1]}`;
  return `features/${parts[1]}/${parts[2]}`;
}

function isGenericDir(relFromSrc) {
  return GENERIC_DIR_PREFIXES.some((prefix) => relFromSrc.startsWith(prefix));
}

/**
 * 创建本地依赖解析器。返回值分类：
 * - external：node_modules 包或内建模块，不判定；
 * - escaped：正规化后越出 web 根目录，违规；
 * - unresolved：无法解析到真实文件，违规；
 * - out-of-scope：解析到 web 根内但 src 之外（web/config、web/tests 等
 *   平台/测试区域），不判定；
 * - resolved：src 内真实文件，参与边界判定。
 */
export function createDependencyResolver({ webRoot, srcRoot }) {
  const statCache = new Map();
  const isFile = (candidate) => {
    const cached = statCache.get(candidate);
    if (cached !== undefined) return cached;
    let result = false;
    try {
      result = statSync(candidate).isFile();
    } catch {
      result = false;
    }
    statCache.set(candidate, result);
    return result;
  };
  const findByExtensions = (base) => {
    for (const ext of RESOLVE_EXTENSIONS) {
      if (isFile(base + ext)) return base + ext;
    }
    return null;
  };
  const findIndexFile = (dir) => {
    for (const ext of RESOLVE_EXTENSIONS) {
      const candidate = join(dir, `index${ext}`);
      if (isFile(candidate)) return candidate;
    }
    return null;
  };
  // 无扩展名引用按扩展名序补全；显式 .js/.jsx 后缀按 TS 惯例映射到 .ts/.tsx。
  const resolveToFile = (candidate) => {
    if (isFile(candidate)) return candidate;
    const byExtension = findByExtensions(candidate);
    if (byExtension) return byExtension;
    const jsLike = /\.jsx?$/.exec(candidate);
    if (jsLike) {
      const tsCandidate = candidate.replace(/\.jsx?$/, jsLike[0] === '.jsx' ? '.tsx' : '.ts');
      if (isFile(tsCandidate)) return tsCandidate;
    }
    return findIndexFile(candidate);
  };
  return (specifier, importerFile) => {
    let candidate;
    if (specifier.startsWith('@/')) {
      candidate = join(srcRoot, specifier.slice(2));
    } else if (specifier === '@root') {
      candidate = webRoot;
    } else if (specifier.startsWith('@root/')) {
      candidate = join(webRoot, specifier.slice('@root/'.length));
    } else if (specifier.startsWith('./') || specifier.startsWith('../')) {
      candidate = resolve(dirname(importerFile), specifier);
    } else {
      return { status: 'external' };
    }
    if (candidate !== webRoot && !candidate.startsWith(webRoot + sep)) {
      return { status: 'escaped', candidate };
    }
    const file = resolveToFile(candidate);
    if (!file) return { status: 'unresolved', candidate };
    if (file !== srcRoot && !file.startsWith(srcRoot + sep)) {
      return { status: 'out-of-scope' };
    }
    return { status: 'resolved', file };
  };
}

const WALK_SKIP_KEYS = new Set([
  'loc',
  'start',
  'end',
  'range',
  'extra',
  'leadingComments',
  'trailingComments',
  'innerComments',
]);

/** .tsx/.jsx 启用 jsx 插件；.ts 仅启用 typescript，避免类型断言语法歧义。 */
function parserPluginsFor(file) {
  return /\.tsx$|\.jsx$/.test(file) ? ['typescript', 'jsx'] : ['typescript'];
}

/**
 * 从单个文件源码提取全部依赖引用，返回 { imports, parseError }。
 * imports 元素形如 { specifier, line }；只收录字面量字符串依赖。
 */
export function extractImports(code, file) {
  let program;
  try {
    ({ program } = parse(code, { sourceType: 'module', plugins: parserPluginsFor(file) }));
  } catch (error) {
    return { imports: [], parseError: error instanceof Error ? error.message : String(error) };
  }
  const imports = [];
  const record = (node) => {
    if (node.source?.type === 'StringLiteral') {
      imports.push({ specifier: node.source.value, line: node.loc.start.line });
    }
  };
  const visit = (node) => {
    if (!node || typeof node.type !== 'string') return;
    switch (node.type) {
      case 'ImportDeclaration':
      case 'ExportAllDeclaration':
      case 'ExportNamedDeclaration':
        record(node);
        break;
      case 'CallExpression': {
        // 字面量 import('...') 与 require('...')；变量拼接等非字面量不判定。
        const isDynamicImport = node.callee?.type === 'Import';
        const isRequire = node.callee?.type === 'Identifier' && node.callee.name === 'require';
        if ((isDynamicImport || isRequire) && node.arguments[0]?.type === 'StringLiteral') {
          imports.push({ specifier: node.arguments[0].value, line: node.loc.start.line });
        }
        break;
      }
      case 'TSImportType':
        // TS 类型级 import('...').Type 引用。
        if (node.argument?.type === 'StringLiteral') {
          imports.push({ specifier: node.argument.value, line: node.loc.start.line });
        }
        break;
      default:
        break;
    }
    for (const key of Object.keys(node)) {
      if (WALK_SKIP_KEYS.has(key)) continue;
      const value = node[key];
      if (Array.isArray(value)) {
        for (const item of value) {
          if (item && typeof item === 'object') visit(item);
        }
      } else if (value && typeof value === 'object') {
        visit(value);
      }
    }
  };
  visit(program);
  return { imports, parseError: null };
}

const NO_VIOLATION = { violations: [], featureEdge: null };
const violation = (rule, reason) => ({ violations: [{ rule, reason }], featureEdge: null });

/**
 * 纯函数边界判定：给定已正规化到 src 的导入方/目标相对路径，返回违规列表与
 * 需要记入 features 依赖图的边（from/to 为 feature 单元）。
 */
export function judgeEdge({ importerRel, targetRel, specifier }) {
  if (targetRel.startsWith('pages/')) {
    if (importerRel.startsWith('features/')) {
      return violation(RULE.FEATURE_TO_PAGES, `features 禁止导入 pages：${specifier}`);
    }
    // 路由页面装配的精确授权：清单内放行，清单外绝不整体豁免 router 目录。
    if (PAGES_ASSEMBLY_ALLOWLIST.includes(importerRel)) return NO_VIOLATION;
    if (importerRel.startsWith('pages/')) {
      const importerModule = pageModuleKey(importerRel);
      const targetModule = pageModuleKey(targetRel);
      if (importerModule === targetModule) return NO_VIOLATION;
      return violation(
        RULE.PAGE_CROSS,
        `页面模块 ${importerModule} 禁止导入其他页面模块 ${targetModule}：${specifier}`,
      );
    }
    if (isGenericDir(importerRel)) {
      return violation(RULE.GENERIC_TO_PAGES, `components/ui、hooks、utils、constants 通用层禁止导入 pages：${specifier}`);
    }
    return violation(
      RULE.OTHER_TO_PAGES,
      `非页面业务代码禁止直接导入 pages（路由页面装配须加入精确授权清单）：${specifier}`,
    );
  }
  if (targetRel.startsWith('features/')) {
    const targetUnit = featureOwnerUnit(targetRel);
    if (!targetUnit) {
      return violation(RULE.FEATURE_PRIVATE, `features 顶层散文件不属于任何领域/能力单元，禁止导入：${specifier}`);
    }
    const targetEntry = `${targetUnit}/index.ts`;
    if (importerRel.startsWith('features/')) {
      const importerUnit = featureOwnerUnit(importerRel);
      if (importerUnit === targetUnit) return NO_VIOLATION;
      if (targetRel !== targetEntry) {
        return violation(
          RULE.FEATURE_PRIVATE,
          `外部只能从公共入口 ${targetEntry} 导入 feature，更深层目录的 index.ts 不自动成为公共 API：${specifier}`,
        );
      }
      return { violations: [], featureEdge: { from: importerUnit, to: targetUnit } };
    }
    if (isGenericDir(importerRel)) {
      return violation(RULE.GENERIC_TO_FEATURES, `components/ui、hooks、utils、constants 通用层禁止导入 features：${specifier}`);
    }
    if (targetRel !== targetEntry) {
      return violation(
        RULE.FEATURE_PRIVATE,
        `外部只能从公共入口 ${targetEntry} 导入 feature：${specifier}`,
      );
    }
    return NO_VIOLATION;
  }
  // src 内其余引用（平台层 app/router/components/layout 等）本期不做强约束。
  return NO_VIOLATION;
}

/**
 * 在 features 公共入口依赖图中查找有向环，返回去重后的环列表。
 * 每个环形如 { units, file, line }，units 首尾相接（末元素指回首元素）。
 */
export function findFeatureCycles(edges) {
  const adjacency = new Map();
  for (const edge of edges) {
    if (edge.from === edge.to) continue;
    const list = adjacency.get(edge.from) ?? [];
    if (!list.some((item) => item.to === edge.to)) list.push(edge);
    adjacency.set(edge.from, list);
  }
  const cycles = [];
  const seenCycleKeys = new Set();
  const color = new Map();
  const stack = [];
  const visit = (unit) => {
    color.set(unit, 1);
    stack.push(unit);
    for (const edge of adjacency.get(unit) ?? []) {
      const state = color.get(edge.to) ?? 0;
      if (state === 0) {
        visit(edge.to);
      } else if (state === 1) {
        const cycleUnits = stack.slice(stack.indexOf(edge.to));
        // 环的规范化键：旋转到字典序最小的单元开头，避免同一环重复上报。
        let smallestIndex = 0;
        for (let index = 1; index < cycleUnits.length; index += 1) {
          if (cycleUnits[index] < cycleUnits[smallestIndex]) smallestIndex = index;
        }
        const canonical = [
          ...cycleUnits.slice(smallestIndex),
          ...cycleUnits.slice(0, smallestIndex),
        ];
        const key = canonical.join('|');
        if (!seenCycleKeys.has(key)) {
          seenCycleKeys.add(key);
          // 归属到规范化环首边（canonical[0] → canonical[1]），保证输出稳定可读。
          const firstEdge = (adjacency.get(canonical[0]) ?? []).find(
            (item) => item.to === canonical[1],
          );
          cycles.push({
            units: [...canonical, canonical[0]],
            file: firstEdge?.file ?? edge.file,
            line: firstEdge?.line ?? edge.line,
          });
        }
      }
    }
    stack.pop();
    color.set(unit, 2);
  };
  for (const unit of Array.from(adjacency.keys()).sort()) {
    if (!color.get(unit)) visit(unit);
  }
  return cycles;
}

/**
 * 对给定文件集合执行完整边界分析，返回：
 * { scannedFiles, dependencyCount, featureEdges, findings }
 * findings 元素：{ file, line, rule, specifier, reason }（file 相对 web 根）。
 */
export function analyze({ webRoot, srcRoot, files }) {
  const resolveSpecifier = createDependencyResolver({ webRoot, srcRoot });
  const findings = [];
  const featureEdges = [];
  const pushFinding = (file, line, rule, specifier, reason) => {
    findings.push({ file: toPosix(relative(webRoot, file)), line, rule, specifier, reason });
  };
  let scannedFiles = 0;
  let dependencyCount = 0;
  for (const file of [...files].sort()) {
    const relFromSrc = toPosix(relative(srcRoot, file));
    if (!isProductFile(relFromSrc)) continue;
    scannedFiles += 1;
    const { imports, parseError } = extractImports(readFileSync(file, 'utf8'), file);
    if (parseError) {
      pushFinding(file, 0, RULE.PARSE, '', `AST 解析失败：${parseError}`);
      continue;
    }
    for (const { specifier, line } of imports) {
      const resolved = resolveSpecifier(specifier, file);
      if (resolved.status === 'external' || resolved.status === 'out-of-scope') continue;
      if (resolved.status === 'escaped') {
        pushFinding(file, line, RULE.ESCAPED, specifier, `本地路径越界（越出 web 根目录）：${specifier}`);
        continue;
      }
      if (resolved.status === 'unresolved') {
        pushFinding(file, line, RULE.UNRESOLVED, specifier, `本地路径无法解析到真实文件：${specifier}`);
        continue;
      }
      dependencyCount += 1;
      const targetRel = toPosix(relative(srcRoot, resolved.file));
      const { violations, featureEdge } = judgeEdge({ importerRel: relFromSrc, targetRel, specifier });
      for (const item of violations) {
        pushFinding(file, line, item.rule, specifier, item.reason);
      }
      if (featureEdge) {
        featureEdges.push({ ...featureEdge, file: toPosix(relative(webRoot, file)), line });
      }
    }
  }
  for (const cycle of findFeatureCycles(featureEdges)) {
    findings.push({
      file: cycle.file,
      line: cycle.line,
      rule: RULE.FEATURE_CYCLE,
      specifier: '',
      reason: `features 能力依赖环：${cycle.units.join(' → ')}`,
    });
  }
  findings.sort((left, right) =>
    left.file.localeCompare(right.file) || left.line - right.line || left.rule.localeCompare(right.rule));
  return { scannedFiles, dependencyCount, featureEdges, findings };
}

/** 对真实 web 目录执行扫描（srcRoot 取 webRoot/src）。 */
export function runScan({ webRoot }) {
  const srcRoot = join(webRoot, 'src');
  return analyze({ webRoot, srcRoot, files: collectSourceFiles(srcRoot) });
}

function main() {
  const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
  const result = runScan({ webRoot });
  const lines = [
    `架构边界扫描：扫描产品文件 ${result.scannedFiles} 个，src 内本地依赖 ${result.dependencyCount} 条，features 能力依赖边 ${result.featureEdges.length} 条。`,
  ];
  if (result.findings.length === 0) {
    lines.push('未发现边界违规，前端依赖边界检查通过。');
    process.stdout.write(`${lines.join('\n')}\n`);
    return 0;
  }
  lines.push(`发现 ${result.findings.length} 处边界违规：`);
  for (const item of result.findings) {
    const specifier = item.specifier ? `（${item.specifier}）` : '';
    lines.push(`- ${item.file}:${item.line} [${item.rule}] ${specifier}${item.reason}`);
  }
  process.stdout.write(`${lines.join('\n')}\n`);
  return 1;
}

// 直接执行本脚本时运行真实扫描；被测试导入时只暴露函数。
const invokedDirectly =
  process.argv[1] !== undefined && import.meta.url === pathToFileURL(resolve(process.argv[1])).href;
if (invokedDirectly) {
  process.exitCode = main();
}
