import { beforeEach, describe, expect, it } from 'vitest';
import {
  clearFeeColumnPreference,
  DEFAULT_FEE_COLUMN_DEFS,
  defaultFeeColumnPreference,
  effectiveFeeColumnDefs,
  type FeeColumnPreference,
  isDefaultFeeColumnPreference,
  loadFeeColumnPreference,
  resolveFeeColumnPreference,
  saveFeeColumnPreference,
} from './feeColumnPreference';

const lockedKeys = DEFAULT_FEE_COLUMN_DEFS.filter((def) => def.lockVisible).map(
  (def) => def.key,
);

describe('feeColumnPreference 列定义与默认值', () => {
  it('录入必备列与操作列必显，税额四列默认可见，日期、标签与关联账单列默认隐藏', () => {
    const defaults = defaultFeeColumnPreference(false);
    expect(lockedKeys).toEqual(
      expect.arrayContaining([
        'feeSettingId',
        'settlementPartyId',
        'currency',
        'unitPrice',
        'quantity',
        'billingUnitId',
        'option',
      ]),
    );
    for (const key of lockedKeys) {
      expect(defaults.hidden).not.toContain(key);
    }
    expect(lockedKeys).not.toContain('createdAt');
    expect(defaults.hidden).toEqual(
      expect.arrayContaining(['createdAt', 'tags']),
    );
    for (const key of [
      'netUnitPrice',
      'taxRate',
      'taxAmount',
      'netAmount',
      'baseCurrencyAmount',
    ] as const) {
      expect(defaults.hidden).not.toContain(key);
    }
    expect(defaults.hidden).not.toContain('feeCode');
  });

  it('不含税单价列默认可见且默认顺序紧跟单价之后，偏好未提及按默认可见兜底', () => {
    const defaults = defaultFeeColumnPreference(false);
    expect(defaults.order[defaults.order.indexOf('unitPrice') + 1]).toBe(
      'netUnitPrice',
    );
    // 老用户偏好存储未提及新列时，按 defaultVisible 兜底自动出现且不进 hidden
    const stored: FeeColumnPreference = {
      order: ['feeCode', 'feeSettingId'],
      hidden: [],
    };
    const resolved = resolveFeeColumnPreference(stored, false);
    expect(resolved.order).toContain('netUnitPrice');
    expect(resolved.hidden).not.toContain('netUnitPrice');
  });

  it('无财务权限时裁剪账单关联列，有权限时提供且默认隐藏', () => {
    expect(effectiveFeeColumnDefs(false).map((def) => def.key)).not.toContain(
      'billNo',
    );
    const withFinance = effectiveFeeColumnDefs(true);
    expect(withFinance.map((def) => def.key)).toContain('billNo');
    expect(withFinance.map((def) => def.key)).toContain('financialProgress');
    expect(defaultFeeColumnPreference(true).hidden).toEqual(
      expect.arrayContaining(['billNo', 'financialProgress']),
    );
  });
});

describe('resolveFeeColumnPreference 存储合并', () => {
  it('忽略未知 key，并按默认可见性兜底偏好未提及的新列', () => {
    // 模拟历史版本或损坏存储写入的未知列 key
    const stored = {
      order: ['note', 'ghost-column', 'feeSettingId'],
      hidden: ['unknown-hidden'],
    } as unknown as FeeColumnPreference;
    const resolved = resolveFeeColumnPreference(stored, false);
    expect(resolved.order).not.toContain('ghost-column' as never);
    expect(resolved.hidden).not.toContain('unknown-hidden' as never);
    // 未提及的可选列（如 tags）按默认隐藏兜底，不会因升级突然出现
    expect(resolved.hidden).toContain('tags');
    expect(resolved.hidden).not.toContain('taxRate');
  });

  it('必显列无论存储内容如何都不可隐藏', () => {
    const stored = {
      order: defaultFeeColumnPreference(false).order,
      hidden: [...lockedKeys, 'taxRate'],
    } as FeeColumnPreference;
    const resolved = resolveFeeColumnPreference(stored, false);
    for (const key of lockedKeys) {
      expect(resolved.hidden).not.toContain(key);
    }
    expect(resolved.hidden).toContain('taxRate');
  });

  it('空存储返回默认偏好；存储顺序缺列时按默认顺序补齐', () => {
    expect(resolveFeeColumnPreference(null, false)).toEqual(
      defaultFeeColumnPreference(false),
    );
    const partial: FeeColumnPreference = {
      order: ['note', 'feeCode'],
      hidden: ['note'],
    };
    const resolved = resolveFeeColumnPreference(partial, false);
    expect(resolved.order.slice(0, 2)).toEqual(['note', 'feeCode']);
    for (const key of defaultFeeColumnPreference(false).order) {
      expect(resolved.order).toContain(key);
    }
  });

  it('isDefaultFeeColumnPreference 识别等价默认与差异偏好', () => {
    const defaults = defaultFeeColumnPreference(false);
    expect(isDefaultFeeColumnPreference(defaults, false)).toBe(true);
    expect(
      isDefaultFeeColumnPreference(
        { order: defaults.order, hidden: [] },
        false,
      ),
    ).toBe(false);
  });
});

describe('feeColumnPreference 本地存储', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('按用户与组织隔离：不同 scope 互不串读', () => {
    const preference: FeeColumnPreference = {
      order: defaultFeeColumnPreference(false).order,
      hidden: ['taxRate'],
      freeze: { left: ['feeCode'], right: [] },
    };
    expect(
      saveFeeColumnPreference(
        { userId: 'user-1', organizationId: 'org-1' },
        preference,
      ),
    ).toBe(true);
    expect(
      loadFeeColumnPreference({ userId: 'user-1', organizationId: 'org-2' }),
    ).toBeNull();
    expect(
      loadFeeColumnPreference({ userId: 'user-2', organizationId: 'org-1' }),
    ).toBeNull();
    expect(
      loadFeeColumnPreference({ userId: 'user-1', organizationId: 'org-1' }),
    ).toEqual(preference);
  });

  it('缺省 scope 段落到 anonymous/default 键，避免误读他人偏好', () => {
    expect(
      saveFeeColumnPreference({}, { order: ['feeCode'], hidden: [] }),
    ).toBe(true);
    expect(
      window.localStorage.getItem(
        'roncin:order-fee-columns:v1:anonymous:default',
      ),
    ).not.toBeNull();
  });

  it('损坏的 JSON 或错误结构回退 null', () => {
    window.localStorage.setItem(
      'roncin:order-fee-columns:v1:user-1:org-1',
      '{bad json',
    );
    expect(
      loadFeeColumnPreference({ userId: 'user-1', organizationId: 'org-1' }),
    ).toBeNull();
    window.localStorage.setItem(
      'roncin:order-fee-columns:v1:user-1:org-1',
      JSON.stringify({ order: 'not-array' }),
    );
    expect(
      loadFeeColumnPreference({ userId: 'user-1', organizationId: 'org-1' }),
    ).toBeNull();
  });

  it('存储不可用时保存与清除返回 false，不抛异常', () => {
    const originalSet = window.localStorage.setItem;
    const originalRemove = window.localStorage.removeItem;
    window.localStorage.setItem = () => {
      throw new Error('quota exceeded');
    };
    window.localStorage.removeItem = () => {
      throw new Error('storage disabled');
    };
    try {
      expect(
        saveFeeColumnPreference(
          { userId: 'user-1', organizationId: 'org-1' },
          { order: ['feeCode'], hidden: [] },
        ),
      ).toBe(false);
      expect(
        clearFeeColumnPreference({ userId: 'user-1', organizationId: 'org-1' }),
      ).toBe(false);
    } finally {
      window.localStorage.setItem = originalSet;
      window.localStorage.removeItem = originalRemove;
    }
  });

  it('恢复默认清除场景偏好键', () => {
    const scope = { userId: 'user-1', organizationId: 'org-1' };
    saveFeeColumnPreference(scope, { order: ['feeCode'], hidden: [] });
    expect(clearFeeColumnPreference(scope)).toBe(true);
    expect(
      window.localStorage.getItem('roncin:order-fee-columns:v1:user-1:org-1'),
    ).toBeNull();
  });
});

describe('feeColumnPreference 字段冻结', () => {
  it('默认只冻结末尾操作列；未配置冻结时保留已有顺序与显隐', () => {
    const defaults = defaultFeeColumnPreference(false);
    expect(defaults.freeze).toEqual({ left: [], right: ['option'] });
    const resolved = resolveFeeColumnPreference(
      { order: ['note', 'feeCode'], hidden: ['taxRate'] },
      false,
    );
    expect(resolved.order.slice(0, 2)).toEqual(['note', 'feeCode']);
    expect(resolved.hidden).toContain('taxRate');
    expect(resolved.freeze).toEqual(defaults.freeze);
  });

  it('保存指定冻结字段与空数组，冻结变化不被误判为恢复默认', () => {
    const defaults = defaultFeeColumnPreference(false);
    const preference = {
      ...defaults,
      freeze: { left: ['feeCode'], right: [] },
    };
    expect(isDefaultFeeColumnPreference(preference, false)).toBe(false);
    const scope = { userId: 'freeze-user', organizationId: 'freeze-org' };
    expect(saveFeeColumnPreference(scope, preference)).toBe(true);
    expect(loadFeeColumnPreference(scope)).toEqual(preference);
    expect(
      isDefaultFeeColumnPreference(
        { ...defaults, freeze: { left: [], right: [] } },
        false,
      ),
    ).toBe(false);
  });

  it('非法冻结值按字段默认解析，不丢弃有效的显隐或顺序', () => {
    const scope = { userId: 'invalid-user', organizationId: 'invalid-org' };
    window.localStorage.setItem(
      'roncin:order-fee-columns:v1:invalid-user:invalid-org',
      JSON.stringify({
        order: ['note'],
        hidden: ['taxAmount'],
        freeze: { left: 'bad', right: [] },
      }),
    );
    const resolved = resolveFeeColumnPreference(
      loadFeeColumnPreference(scope),
      false,
    );
    expect(resolved.freeze).toEqual({ left: [], right: [] });
    expect(resolved.order[0]).toBe('note');
    expect(resolved.hidden).toContain('taxAmount');
  });

  it('操作列始终必显且在末尾，重复排序键去重，财务列继续按权限裁剪', () => {
    const resolved = resolveFeeColumnPreference(
      {
        order: ['option', 'note', 'billNo', 'note'],
        hidden: ['option', 'note'],
        freeze: { left: ['feeCode'], right: ['option'] },
      },
      false,
    );
    expect(resolved.order[0]).toBe('note');
    expect(resolved.order.at(-1)).toBe('option');
    expect(resolved.order.filter((key) => key === 'note')).toHaveLength(1);
    expect(resolved.order).not.toContain('billNo');
    expect(resolved.hidden).not.toContain('option');
    expect(resolved.hidden).toContain('note');
    expect(resolved.freeze).toEqual({ left: ['feeCode'], right: ['option'] });
  });
});
