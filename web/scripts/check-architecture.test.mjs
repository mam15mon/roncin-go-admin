// check-architecture.mjs 的针对性正反例测试（node:test）。
// 夹具在运行时构建到临时目录，覆盖 design.md 第 4 节全部规则：
// 别名/相对路径两类形态的每类违规、深层 index 非公共入口、重导出链穿透、
// import type、字面量动态 import()/require()、注释与字符串不误报、
// 合法同模块引用、合法公开 API 引用、路由精确授权清单、features 领域环、
// 无法解析/越界路径报错。每条违规断言输出含文件、行号与原因。

import assert from 'node:assert/strict';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { after, test } from 'node:test';
import {
  PAGES_ASSEMBLY_ALLOWLIST,
  RULE,
  findFeatureCycles,
  pageModuleKey,
  runScan,
} from './check-architecture.mjs';

const createdWebRoots = [];

after(async () => {
  await Promise.all(
    createdWebRoots.map((webRoot) => rm(webRoot, { recursive: true, force: true })),
  );
});

/** 在临时 web 根下按 { 相对路径: 内容 } 构建夹具并执行真实扫描。 */
async function scanFixture(files) {
  const webRoot = await mkdtemp(join(tmpdir(), 'arch-check-'));
  createdWebRoots.push(webRoot);
  for (const [relPath, content] of Object.entries(files)) {
    const absPath = join(webRoot, relPath);
    await mkdir(dirname(absPath), { recursive: true });
    await writeFile(absPath, content, 'utf8');
  }
  return runScan({ webRoot });
}

/** 断言存在指定文件、行号、规则的违规，且原因非空（AC3 输出要求）。 */
function expectFinding(findings, file, line, rule) {
  const hit = findings.find(
    (item) => item.file === file && item.line === line && item.rule === rule,
  );
  assert.ok(
    hit,
    `应报告 ${file}:${line} [${rule}]，实际：${JSON.stringify(findings, null, 2)}`,
  );
  assert.equal(typeof hit.file, 'string');
  assert.equal(typeof hit.line, 'number');
  assert.ok(hit.reason.length > 0, '违规必须输出原因');
  return hit;
}

function expectClean(findings) {
  assert.deepEqual(findings, []);
}

test('路由页面装配授权清单当前仅含 adaptRoutes.tsx（精确授权，不豁免 router 目录）', () => {
  assert.deepEqual(PAGES_ASSEMBLY_ALLOWLIST, ['router/adaptRoutes.tsx']);
});

test('跨页面模块导入：别名与相对路径两种形态均违规', async () => {
  const result = await scanFixture({
    'src/pages/orders/index.tsx':
      "import { FeeLedgerTable } from '@/pages/finance/bills';",
    'src/pages/finance/fees/index.tsx':
      "import { billHelper } from '../bills/constants';",
    'src/pages/finance/bills/index.tsx': 'export const anchor = 1;',
    'src/pages/finance/bills/constants.ts': 'export const billHelper = 1;',
  });
  assert.equal(result.findings.length, 2);
  const byAlias = expectFinding(
    result.findings,
    'src/pages/orders/index.tsx',
    1,
    RULE.PAGE_CROSS,
  );
  assert.ok(
    byAlias.reason.includes('pages/orders') && byAlias.reason.includes('pages/finance/bills'),
    `原因应包含两个模块名：${byAlias.reason}`,
  );
  expectFinding(result.findings, 'src/pages/finance/fees/index.tsx', 1, RULE.PAGE_CROSS);
});

test('同模块内部引用合法：别名与相对路径均放行（含 finance 子模块内部）', async () => {
  const result = await scanFixture({
    'src/pages/orders/index.tsx':
      "import { common } from '@/pages/orders/common';\nimport { types } from './types';",
    'src/pages/orders/common.ts': 'export const common = 1;',
    'src/pages/orders/types.ts': 'export const types = 1;',
    'src/pages/finance/bills/index.tsx': "import { c } from './constants';",
    'src/pages/finance/bills/constants.ts': 'export const c = 1;',
  });
  expectClean(result.findings);
});

test('页面模块边界：finance 顶层单文件页面与 finance 子模块互为独立模块', () => {
  assert.equal(pageModuleKey('pages/finance/settlement-placeholder.tsx'), 'pages/finance/settlement-placeholder.tsx');
  assert.equal(pageModuleKey('pages/finance/bills/index.tsx'), 'pages/finance/bills');
  assert.equal(pageModuleKey('pages/orders/detail.tsx'), 'pages/orders');
  assert.equal(pageModuleKey('pages/NotFound.tsx'), 'pages/NotFound.tsx');
});

test('features 反向依赖 pages：别名与相对路径均违规', async () => {
  const result = await scanFixture({
    'src/features/audit/presenter.ts':
      "import { adminAudit } from '@/pages/admin/audit';",
    'src/features/finance/bill-creation/panel.tsx':
      "import { legacy } from '../../../pages/admin/legacy';",
    'src/pages/admin/audit.tsx': 'export const adminAudit = 1;',
    'src/pages/admin/legacy.tsx': 'export const legacy = 1;',
  });
  assert.equal(result.findings.length, 2);
  expectFinding(result.findings, 'src/features/audit/presenter.ts', 1, RULE.FEATURE_TO_PAGES);
  expectFinding(
    result.findings,
    'src/features/finance/bill-creation/panel.tsx',
    1,
    RULE.FEATURE_TO_PAGES,
  );
});

test('外部绕过 feature 公开入口：别名与相对路径深导入均违规，入口引用放行', async () => {
  const result = await scanFixture({
    'src/pages/demo/index.tsx':
      "import { auditActorName } from '@/features/audit/audit-presentation';",
    'src/pages/demo/other.tsx':
      "import { again } from '../../features/audit/audit-presentation';",
    'src/pages/demo/entry.tsx': "import { viaEntry } from '@/features/audit';",
    'src/features/audit/index.ts':
      "export { auditActorName, again, viaEntry } from './audit-presentation';",
    'src/features/audit/audit-presentation.ts':
      'export const auditActorName = 1;\nexport const again = 2;\nexport const viaEntry = 3;',
  });
  assert.equal(result.findings.length, 2);
  expectFinding(result.findings, 'src/pages/demo/index.tsx', 1, RULE.FEATURE_PRIVATE);
  expectFinding(result.findings, 'src/pages/demo/other.tsx', 1, RULE.FEATURE_PRIVATE);
});

test('深层目录 index 不自动成为公共 API：外部导入能力内部更深层 index 报违规', async () => {
  const result = await scanFixture({
    'src/pages/demo/index.tsx':
      "import { inner } from '@/features/audit/sub/inner';",
    'src/features/audit/index.ts': 'export const anchor = 1;',
    'src/features/audit/sub/index.ts': 'export const sub = 1;',
    'src/features/audit/sub/inner/index.ts': 'export const inner = 1;',
  });
  const hit = expectFinding(result.findings, 'src/pages/demo/index.tsx', 1, RULE.FEATURE_PRIVATE);
  assert.ok(
    hit.reason.includes('features/audit/sub/index.ts'),
    `原因应指出真正的公共入口：${hit.reason}`,
  );
});

test('重导出链穿透：export from 与 export * from 均按依赖判定', async () => {
  const result = await scanFixture({
    'src/pages/demo/index.tsx':
      "export { hidden } from '@/features/audit/internal';\nexport * from '../../features/audit/deeper';",
    'src/features/audit/index.ts': "export * from './audit-presentation';",
    'src/features/audit/audit-presentation.ts': 'export const visible = 1;',
    'src/features/audit/internal.ts': 'export const hidden = 1;',
    'src/features/audit/deeper.ts': 'export const deep = 1;',
  });
  assert.equal(result.findings.length, 2);
  expectFinding(result.findings, 'src/pages/demo/index.tsx', 1, RULE.FEATURE_PRIVATE);
  expectFinding(result.findings, 'src/pages/demo/index.tsx', 2, RULE.FEATURE_PRIVATE);
});

test('import type 违规同样计入：页面跨模块类型导入与绕过 feature 入口的类型导入', async () => {
  const result = await scanFixture({
    'src/pages/finance/bills/index.tsx':
      "import type { FeeValues } from '@/pages/finance/fees/types';",
    'src/pages/finance/fees/types.ts': 'export type FeeValues = { feeId: string };',
    'src/pages/demo/index.tsx':
      "import type { AuditActor } from '@/features/audit/audit-presentation';",
    'src/features/audit/audit-presentation.ts': 'export type AuditActor = string;',
  });
  assert.equal(result.findings.length, 2);
  expectFinding(
    result.findings,
    'src/pages/finance/bills/index.tsx',
    1,
    RULE.PAGE_CROSS,
  );
  expectFinding(result.findings, 'src/pages/demo/index.tsx', 1, RULE.FEATURE_PRIVATE);
});

test('字面量动态 import() 与 require() 违规同样计入', async () => {
  const result = await scanFixture({
    'src/pages/orders/index.tsx': [
      "export const loadHeavy = () => import('@/pages/admin/heavy');",
      "export const loadLegacy = () => import('../finance/fees');",
      "export const legacyRequire = () => require('@/pages/partners/tags');",
    ].join('\n'),
    'src/pages/admin/heavy.tsx': 'export const heavy = 1;',
    'src/pages/finance/fees/index.tsx': 'export const fees = 1;',
    'src/pages/partners/tags.tsx': 'export const tags = 1;',
  });
  assert.equal(result.findings.length, 3);
  expectFinding(result.findings, 'src/pages/orders/index.tsx', 1, RULE.PAGE_CROSS);
  expectFinding(result.findings, 'src/pages/orders/index.tsx', 2, RULE.PAGE_CROSS);
  expectFinding(result.findings, 'src/pages/orders/index.tsx', 3, RULE.PAGE_CROSS);
});

test('注释、普通字符串、模板字符串与非字面量动态导入不误报', async () => {
  const result = await scanFixture({
    'src/pages/orders/comment-trap.ts': [
      "// import '@/pages/finance/bills';",
      '/* ../../pages/finance/bills */',
      "const label = '@/pages/finance/bills';",
      'const hint = "import x from \'../../pages/finance/bills\'";',
      'const tpl = `see ../../pages/finance/bills`;',
      'const dynamic = (path: string) => import(path);',
      "const glob = import.meta.glob('../pages/**/*.tsx');",
      'export { dynamic, glob, hint, label, tpl };',
    ].join('\n'),
    'src/pages/finance/bills/index.tsx': 'export const bills = 1;',
  });
  expectClean(result.findings);
});

test('合法公开 API 引用：页面消费 feature 入口、feature 间跨能力入口引用', async () => {
  const result = await scanFixture({
    'src/pages/finance/bills/index.tsx':
      "import { BillCreationWorkbench } from '@/features/finance/bill-creation';\nimport { billStatusMeta } from '@/features/finance/bill-status';",
    'src/pages/orders/fees.tsx':
      "import { PartnerSelectOptionTags } from '@/features/partners';",
    'src/features/finance/bill-creation/result.tsx':
      "import { billStatusMeta } from '@/features/finance/bill-status';",
    'src/features/finance/bill-creation/index.ts':
      "export { ResultTable } from './result';",
    'src/features/finance/bill-status/index.ts':
      'export const billStatusMeta = {};',
    'src/features/partners/index.ts': 'export const PartnerSelectOptionTags = 1;',
  });
  expectClean(result.findings);
  assert.deepEqual(result.featureEdges, [
    {
      from: 'features/finance/bill-creation',
      to: 'features/finance/bill-status',
      file: 'src/features/finance/bill-creation/result.tsx',
      line: 1,
    },
  ]);
});

test('路由页面装配授权清单：清单内放行，清单外 router 文件与其他业务代码导入 pages 均违规', async () => {
  const result = await scanFixture({
    'src/router/adaptRoutes.tsx': "import OrdersPage from '@/pages/orders';",
    'src/router/other.ts': "import AdminPage from '@/pages/admin';",
    'src/app/portal.tsx': "import PartnerPage from '@/pages/partners';",
    'src/pages/orders/index.tsx': 'export const orders = 1;',
    'src/pages/admin/index.tsx': 'export const admin = 1;',
    'src/pages/partners/index.tsx': 'export const partners = 1;',
  });
  assert.equal(result.findings.length, 2);
  expectFinding(result.findings, 'src/router/other.ts', 1, RULE.OTHER_TO_PAGES);
  expectFinding(result.findings, 'src/app/portal.tsx', 1, RULE.OTHER_TO_PAGES);
});

test('通用层反向依赖：components/ui、hooks、utils、constants 导入 pages 与 features 均违规', async () => {
  const result = await scanFixture({
    'src/components/ui/box.tsx':
      "import { page } from '@/pages/admin';\nimport { audit } from '@/features/audit';",
    'src/hooks/useThing.ts': "import { page } from '@/pages/admin';",
    'src/utils/calc.ts':
      "import { page } from '@/pages/admin';\nimport { audit } from '@/features/audit';",
    'src/constants/lookup.ts': "import { audit } from '@/features/audit';",
    'src/pages/admin/index.tsx': 'export const admin = 1;',
    'src/features/audit/index.ts': 'export const audit = 1;',
  });
  assert.equal(result.findings.length, 6);
  expectFinding(result.findings, 'src/components/ui/box.tsx', 1, RULE.GENERIC_TO_PAGES);
  expectFinding(result.findings, 'src/components/ui/box.tsx', 2, RULE.GENERIC_TO_FEATURES);
  expectFinding(result.findings, 'src/hooks/useThing.ts', 1, RULE.GENERIC_TO_PAGES);
  expectFinding(result.findings, 'src/utils/calc.ts', 1, RULE.GENERIC_TO_PAGES);
  expectFinding(result.findings, 'src/utils/calc.ts', 2, RULE.GENERIC_TO_FEATURES);
  expectFinding(result.findings, 'src/constants/lookup.ts', 1, RULE.GENERIC_TO_FEATURES);
});

test('features 能力依赖环：两能力互相经公开入口引用时报环', async () => {
  const result = await scanFixture({
    'src/features/finance/bill-creation/index.ts':
      "import { billStatusMeta } from '@/features/finance/bill-status';",
    'src/features/finance/bill-status/index.ts':
      "import { workbench } from '@/features/finance/bill-creation';",
  });
  assert.equal(result.findings.length, 1);
  const hit = expectFinding(
    result.findings,
    'src/features/finance/bill-creation/index.ts',
    1,
    RULE.FEATURE_CYCLE,
  );
  assert.ok(
    hit.reason.includes('features/finance/bill-creation → features/finance/bill-status'),
    `环原因应包含依赖链：${hit.reason}`,
  );
});

test('findFeatureCycles：三节点环规范化为单一结果', () => {
  const cycles = findFeatureCycles([
    { from: 'features/a', to: 'features/b', file: 'f-a', line: 1 },
    { from: 'features/b', to: 'features/c', file: 'f-b', line: 2 },
    { from: 'features/c', to: 'features/a', file: 'f-c', line: 3 },
  ]);
  assert.equal(cycles.length, 1);
  assert.deepEqual(cycles[0].units, ['features/a', 'features/b', 'features/c', 'features/a']);
});

test('无法解析的本地路径与越出 web 根目录的路径均报错', async () => {
  const result = await scanFixture({
    'src/pages/orders/index.tsx':
      "import './no-such-module';\nimport escape from '../../../../../../outside';",
  });
  assert.equal(result.findings.length, 2);
  expectFinding(result.findings, 'src/pages/orders/index.tsx', 1, RULE.UNRESOLVED);
  expectFinding(result.findings, 'src/pages/orders/index.tsx', 2, RULE.ESCAPED);
});

test('测试文件与生成物不参与产品边界检查，也不建立任何豁免基线', async () => {
  const result = await scanFixture({
    'src/pages/orders/index.test.ts':
      "import { x } from '@/pages/finance/bills';",
    'src/pages/orders/other.spec.tsx':
      "import { y } from '@/pages/finance/bills';",
    'src/services/roncin/billService.ts':
      "import { z } from '@/pages/finance/bills';",
    'src/enums.generated.ts': "import { w } from '@/pages/finance/bills';",
    'src/pages/finance/bills/index.tsx': 'export const bills = 1;',
  });
  expectClean(result.findings);
  assert.equal(result.scannedFiles, 1);
});
