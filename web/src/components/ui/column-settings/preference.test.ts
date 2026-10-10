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

describe('冻结偏好存取与解析', () => {
  afterEach(() => window.localStorage.clear());

  it('增加冻结后保留已有显隐与顺序，并补业务默认开关', () => {
    const stored = { order: ['email', 'name', 'phone'], hidden: ['email'] };
    const defaults = defaultColumnSettingsValue(fields, [], {
      first: true,
      last: true,
    });
    expect(resolveColumnSettingsValue(fields, defaults, stored)).toEqual({
      ...stored,
      freeze: { first: true, last: true },
    });
  });

  it('冻结关闭写入并读取，用户/组织/表格分别隔离', () => {
    const value = {
      order: ['name', 'email'],
      hidden: ['email'],
      freeze: { first: false, last: false },
    };
    const scope = { userId: 'u1', organizationId: 'o1' };
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

  it('非法冻结内容不改变显隐、必显与未知字段过滤', () => {
    window.localStorage.setItem(
      'roncin:column-settings:v1:users:u1:o1',
      JSON.stringify({
        order: ['email', 'unknown', 'name', 'phone'],
        hidden: ['email', 'phone'],
        freeze: { first: 'yes', last: null },
      }),
    );
    const stored = loadColumnSettingsPreference('users', {
      userId: 'u1',
      organizationId: 'o1',
    });
    expect(
      resolveColumnSettingsValue(
        fields,
        defaultColumnSettingsValue(fields),
        stored,
      ),
    ).toEqual({
      order: ['email', 'name', 'phone'],
      hidden: ['email'],
      freeze: { first: true, last: false },
    });
  });

  it('全隐藏兜底仍保留显式冻结选择', () => {
    expect(
      resolveColumnSettingsValue(
        fields.slice(0, 2),
        defaultColumnSettingsValue(fields.slice(0, 2)),
        {
          order: ['name', 'email'],
          hidden: ['name', 'email'],
          freeze: { first: false, last: true },
        },
      ),
    ).toEqual({
      order: ['name', 'email'],
      hidden: [],
      freeze: { first: false, last: true },
    });
  });

  it('逐侧读取合法false，非法另一侧使用业务默认', () => {
    const scope = { userId: 'u1', organizationId: 'o1' };
    const defaults = defaultColumnSettingsValue(fields, [], {
      first: true,
      last: true,
    });
    for (const [freeze, expected] of [
      [
        { first: false, last: 'bad' },
        { first: false, last: true },
      ],
      [
        { first: null, last: false },
        { first: true, last: false },
      ],
    ] as const) {
      window.localStorage.setItem(
        'roncin:column-settings:v1:users:u1:o1',
        JSON.stringify({
          order: ['name', 'email', 'phone'],
          hidden: ['email'],
          freeze,
        }),
      );
      const stored = loadColumnSettingsPreference('users', scope);
      expect(
        resolveColumnSettingsValue(fields, defaults, stored),
      ).toMatchObject({ hidden: ['email'], freeze: expected });
    }
  });
});
