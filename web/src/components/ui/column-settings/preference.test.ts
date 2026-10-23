import { afterEach, describe, expect, it } from 'vitest';
import {
  defaultColumnSettingsValue,
  loadColumnSettingsPreference,
  resolveColumnSettingsValue,
  saveColumnSettingsPreference,
} from './preference';

const fields = [
  { key: 'name', title: '姓名', fixed: 'left' as const },
  { key: 'email', title: '邮箱' },
  { key: 'phone', title: '电话', lockVisible: true },
];
const scope = { userId: 'u1', organizationId: 'o1' };
describe('字段冻结偏好存取', () => {
  afterEach(() => window.localStorage.clear());
  it('保留已有显隐顺序，新冻结缺失时使用完整业务默认', () => {
    const stored = { order: ['email', 'name', 'phone'], hidden: ['email'] };
    expect(
      resolveColumnSettingsValue(
        fields,
        defaultColumnSettingsValue(fields),
        stored,
      ),
    ).toEqual({ ...stored, freeze: { left: ['name'], right: [] } });
  });
  it('空数组关闭冻结并按用户组织表格隔离', () => {
    const value = {
      order: ['name', 'email'],
      hidden: ['email'],
      freeze: { left: [], right: [] },
    };
    expect(saveColumnSettingsPreference('users', scope, value)).toBe(true);
    expect(loadColumnSettingsPreference('users', scope)).toEqual(value);
    expect(
      loadColumnSettingsPreference('users', { ...scope, userId: 'u2' }),
    ).toBeNull();
    expect(
      loadColumnSettingsPreference('users', { ...scope, organizationId: 'o2' }),
    ).toBeNull();
    expect(loadColumnSettingsPreference('orders', scope)).toBeNull();
  });
  it('过滤未知字段并保留必显，非法一侧不覆盖另一侧', () => {
    window.localStorage.setItem(
      'roncin:column-settings:v1:users:u1:o1',
      JSON.stringify({
        order: ['email', 'unknown', 'name', 'phone'],
        hidden: ['email', 'phone'],
        freeze: { left: 'bad', right: ['email', 'unknown'] },
      }),
    );
    expect(
      resolveColumnSettingsValue(
        fields,
        defaultColumnSettingsValue(fields),
        loadColumnSettingsPreference('users', scope),
      ),
    ).toEqual({
      order: ['email', 'name', 'phone'],
      hidden: ['email'],
      freeze: { left: ['name'], right: ['email'] },
    });
  });
  it('全隐藏恢复默认显隐，仍保留冻结字段身份', () => {
    expect(
      resolveColumnSettingsValue(
        fields.slice(0, 2),
        defaultColumnSettingsValue(fields.slice(0, 2)),
        {
          order: ['name', 'email'],
          hidden: ['name', 'email'],
          freeze: { left: [], right: ['email'] },
        },
      ),
    ).toEqual({
      order: ['name', 'email'],
      hidden: [],
      freeze: { left: [], right: ['email'] },
    });
  });
  it('结构操作列即使业务默认不固定也能保存新冻结', () => {
    expect(
      resolveColumnSettingsValue(
        fields,
        defaultColumnSettingsValue(fields),
        {
          order: ['name', 'email'],
          hidden: [],
          freeze: { left: [], right: ['action'] },
        },
        ['name', 'email', 'phone', 'action'],
      ).freeze,
    ).toEqual({ left: [], right: ['action'] });
  });
});
