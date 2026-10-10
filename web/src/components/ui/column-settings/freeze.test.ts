import { Table } from 'antd';
import { describe, expect, it } from 'vitest';
import {
  applyColumnFreeze,
  deriveColumnFreezeDefaults,
  getColumnFreezeTargets,
  resolveColumnFreeze,
} from './freeze';

describe('首尾冻结规则', () => {
  it('首尾按可见顺序计算，并取消中间列的原固定设置', () => {
    const headerCell = () => ({ width: 180 });
    const columns = [
      {
        key: 'name',
        title: '姓名',
        width: 180,
        fixed: 'left' as const,
        onHeaderCell: headerCell,
      },
      { key: 'email', title: '邮箱', fixed: 'left' as const },
      { key: 'action', title: '操作', fixed: 'right' as const },
    ];
    const frozen = applyColumnFreeze(columns, { first: true, last: true });
    expect(frozen.map((column) => column.fixed)).toEqual([
      'left',
      false,
      'right',
    ]);
    expect(frozen[0].onHeaderCell).toBe(headerCell);
    expect(frozen[0].width).toBe(180);
    expect(columns[1].fixed).toBe('left');
    expect(
      applyColumnFreeze(columns, { first: false, last: false }).map(
        (column) => column.fixed,
      ),
    ).toEqual([false, false, false]);
  });

  it('选择、展开、序号保持结构规则，不作为首尾数据目标', () => {
    const index = { key: 'index', valueType: 'index', fixed: 'left' as const };
    const name = { key: 'name', title: '姓名' };
    const action = { key: 'action', title: '操作' };
    const columns = [
      Table.SELECTION_COLUMN,
      Table.EXPAND_COLUMN,
      index,
      name,
      action,
    ];
    expect(getColumnFreezeTargets(columns)).toEqual({
      first: name,
      last: action,
    });
    const frozen = applyColumnFreeze(columns, { first: true, last: true });
    expect(frozen[0]).toBe(Table.SELECTION_COLUMN);
    expect(frozen[1]).toBe(Table.EXPAND_COLUMN);
    expect(frozen[2]).toBe(index);
    expect(frozen[3]).toMatchObject({ fixed: 'left' });
    expect(frozen[4]).toMatchObject({ fixed: 'right' });
  });

  it('无稳定key的操作列仍是合法尾列，辅助固定不产生默认开关', () => {
    const columns = [
      { valueType: 'index', fixed: 'left' as const },
      { key: 'name' },
      { title: '操作', valueType: 'option', fixed: 'right' as const },
    ];
    expect(deriveColumnFreezeDefaults(columns)).toEqual({
      first: false,
      last: true,
    });
    expect(
      applyColumnFreeze(columns, { first: true, last: false }).map(
        (column) => column.fixed,
      ),
    ).toEqual(['left', 'left', false]);
    expect(getColumnFreezeTargets([{ key: 'action' }])).toEqual({
      first: undefined,
      last: { key: 'action' },
    });
  });

  it('首尾重合时左侧优先，不改写两个开关', () => {
    const columns: { key: string; fixed?: 'left' | 'right' | boolean }[] = [
      { key: 'name' },
    ];
    expect(
      applyColumnFreeze(columns, { first: true, last: true })[0].fixed,
    ).toBe('left');
    expect(
      applyColumnFreeze(columns, { first: false, last: true })[0].fixed,
    ).toBe('right');
    expect(getColumnFreezeTargets([])).toEqual({
      first: undefined,
      last: undefined,
    });
  });

  it('隐藏与搜索列不参与首尾，列组使用真实可见叶列', () => {
    const name = { key: 'name' };
    const action = { key: 'option' };
    const columns = [
      { key: 'hidden', hidden: true },
      { key: 'search', hideInTable: true },
      { title: '资料', children: [name, { key: 'email' }] },
      action,
    ];
    expect(getColumnFreezeTargets(columns)).toEqual({
      first: name,
      last: action,
    });
    const frozen = applyColumnFreeze(columns, { first: true, last: true });
    expect(frozen[2]).toMatchObject({
      children: [
        { key: 'name', fixed: 'left' },
        { key: 'email', fixed: false },
      ],
    });
  });

  it('显式结构字段排除首列，但末尾结构操作仍可以冻结', () => {
    const columns = [
      { key: 'direction' },
      { key: 'feeName' },
      { key: 'customAction' },
    ];
    expect(
      getColumnFreezeTargets(columns, {
        firstExcludedKeys: ['direction', 'customAction'],
      }),
    ).toEqual({ first: columns[1], last: columns[2] });
  });

  it('非法偏好使用业务默认，false不被默认覆盖', () => {
    expect(
      resolveColumnFreeze(
        { first: false, last: false },
        { first: true, last: true },
      ),
    ).toEqual({ first: false, last: false });
    expect(
      resolveColumnFreeze(
        { first: 'false', last: true },
        { first: false, last: false },
      ),
    ).toEqual({ first: false, last: true });
    expect(resolveColumnFreeze(null, { first: true, last: false })).toEqual({
      first: true,
      last: false,
    });
  });
});
