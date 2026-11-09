import assert from 'node:assert/strict';
import { test } from 'node:test';
import { checkMapperSource } from './check-master-data-mappers.mjs';

const sharedImport =
  "import { mapPersistedMasterDataFields as fields } from './masterDataMapper';";
const check = (source) => checkMapperSource(source, 'PortsPanel.tsx');

test('正确共享映射与专有 optional 判断通过，注释和字符串不误报', () => {
  assert.deepEqual(
    check(`${sharedImport}
    const mapPort = (item: API.Port) => ({
      ...fields(item), name: item.nameZh === undefined ? '' : item.nameZh,
      note: 'item.enabled === undefined', // item.sortOrder ?? 100
    });`),
    [],
  );
});

test('重复缺失拒绝和默认值处理均拒绝，包含反向比较及计算属性', () => {
  for (const expression of [
    'item.enabled === undefined',
    'undefined === item.sortOrder',
    'item.enabled ?? false',
    "item['sortOrder'] || 0",
  ]) {
    const findings = check(`${sharedImport}
      const mapPort = (item: API.Port) => { const value = ${expression}; return fields(item); };`);
    assert.equal(findings.length, 1);
    assert.match(findings[0].reason, /公共主数据映射/);
    assert.ok(findings[0].line > 0);
  }
});

test('解构重写公共字段被拒绝', () => {
  const findings = check(`${sharedImport}
    function mapPort(item: API.Port) { const { enabled = false } = item; return fields(item); }`);
  assert.equal(findings.length, 1);
  assert.match(findings[0].reason, /解构/);
});

test('缺少共享入口、仅导入却不调用、伪造同名本地函数均拒绝', () => {
  for (const prefix of ['', sharedImport, 'const fields = (value) => value;']) {
    const findings = check(
      `${prefix}\nconst mapPort = (item: API.Port) => ({ id: item.id });`,
    );
    assert.equal(findings.length, 1);
    assert.match(findings[0].reason, /必须调用/);
  }
});

test('请求 optional 筛选与非主数据实体不在门禁范围内', () => {
  assert.deepEqual(
    check(`
    const request = (item: API.MasterDataServiceListPortsParams) => item.enabled === undefined;
    const order = (item: API.Order) => item.enabled ?? false;
    const form = (item: FormValues) => item.sortOrder ?? 100;
  `),
    [],
  );
});

test('普通 DTO helper 与 mapper 内层回调的同名参数不误报', () => {
  assert.deepEqual(
    check(`${sharedImport}
      const isPortEnabled = (item: API.Port) => item.enabled === true;
      function findPort(item: API.Port) { return item.id; }
      const mapPort = (item: API.Port) => ({
        ...fields(item), labels: values.map((item) => item.enabled),
      });`),
    [],
  );
});

test('嵌套函数中调用公共映射不代替适配器实际调用', () => {
  const findings = check(`${sharedImport}
    const mapPort = (item: API.Port) => {
      const unused = (item) => fields(item);
      return { id: item.id };
    };`);
  assert.equal(findings.length, 1);
  assert.match(findings[0].reason, /必须调用/);
});
