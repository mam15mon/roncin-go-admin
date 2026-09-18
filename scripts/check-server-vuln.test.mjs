import assert from 'node:assert/strict';
import test from 'node:test';
import { classifyVulnerabilityReport } from './check-server-vuln.mjs';

const excelize = 'github.com/xuri/excelize/v2';
const fixedVersion = 'v2.11.1-0.20260728235842-f98df08a8f6a';
const affected = (name) => ({ status: 'affected', vulnerability: { name } });

test('仅豁免已审计的精确 Excelize 版本', () => {
  const result = classifyVulnerabilityReport(
    { statements: [affected('GO-2026-6452')] },
    (module) => module === excelize ? fixedVersion : null,
  );
  assert.deepEqual(result.unresolved, []);
  assert.equal(result.accepted.length, 1);
});

test('新增可达漏洞仍使门禁失败', () => {
  const result = classifyVulnerabilityReport(
    { statements: [affected('GO-2026-6452'), affected('GO-2026-9999')] },
    () => fixedVersion,
  );
  assert.equal(result.unresolved.length, 1);
  assert.equal(result.unresolved[0].vulnerability.name, 'GO-2026-9999');
});

test('Excelize 版本变化或无法读取时不继承豁免', () => {
  for (const version of ['v2.11.0', null]) {
    const result = classifyVulnerabilityReport(
      { statements: [affected('GO-2026-6452')] },
      () => version,
    );
    assert.equal(result.unresolved.length, 1);
    assert.equal(result.accepted.length, 0);
  }
});

test('扫描输出格式异常时失败，不视作零漏洞', () => {
  assert.throws(() => classifyVulnerabilityReport({}, () => fixedVersion));
  assert.throws(() => classifyVulnerabilityReport({ statements: [{}] }, () => fixedVersion));
});
