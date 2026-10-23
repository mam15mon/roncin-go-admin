import type { ProColumns } from '@ant-design/pro-components';
import { describe, expect, it, vi } from 'vitest';
import { orderColumnsByPreference } from './orderFeeOptionalColumns';

describe('订单费用可见列的字段冻结应用', () => {
  const onHeaderCell = vi.fn(() => ({ style: { width: 180 } }));
  const columns: ProColumns<API.OrderFee>[] = [
    { title: '费用代码', dataIndex: 'feeCode', width: 100 },
    { title: '费用名称', dataIndex: 'feeSettingId', width: 180, onHeaderCell },
    { title: '备注', dataIndex: 'note', width: 140 },
    { title: '操作', valueType: 'option', width: 120, fixed: 'right' },
  ];

  it('隐藏其他字段后保留指定字段冻结、列宽与表头回调', () => {
    const result = orderColumnsByPreference(columns, {
      order: ['feeCode', 'feeSettingId', 'note', 'option'],
      hidden: ['feeCode'],
      freeze: { left: ['feeSettingId'], right: ['option'] },
    });
    expect(result.map((column) => column.dataIndex ?? 'option')).toEqual([
      'feeSettingId',
      'note',
      'option',
    ]);
    expect(result.map((column) => column.fixed)).toEqual([
      'left',
      false,
      'right',
    ]);
    expect(result[0].width).toBe(180);
    expect(result[0].onHeaderCell).toBe(onHeaderCell);
    expect(columns[1].fixed).toBeUndefined();
  });

  it('指定字段按区域排序；操作列不能被偏好隐藏或排到中间', () => {
    const result = orderColumnsByPreference(columns, {
      order: ['option', 'note', 'feeCode', 'note', 'feeSettingId'],
      hidden: ['option'],
      freeze: { left: ['note'], right: [] },
    });
    expect(result.map((column) => column.dataIndex ?? 'option')).toEqual([
      'note',
      'feeCode',
      'feeSettingId',
      'option',
    ]);
    expect(result.map((column) => column.fixed)).toEqual([
      'left',
      false,
      false,
      false,
    ]);
    expect(columns.at(-1)?.fixed).toBe('right');
  });

  it('空冻结数组解除固定，未配置时从业务列推导默认操作列冻结', () => {
    const order = ['feeCode', 'feeSettingId', 'note', 'option'] as const;
    const unfreeze = orderColumnsByPreference(columns, {
      order: [...order],
      hidden: [],
      freeze: { left: [], right: [] },
    });
    expect(unfreeze.every((column) => !column.fixed)).toBe(true);
    const defaults = orderColumnsByPreference(columns, {
      order: [...order],
      hidden: [],
    });
    expect(defaults.at(-1)?.fixed).toBe('right');
    expect(defaults[0].fixed).toBe(false);
  });

  it('仅一个业务列时按字段冻结且不改写偏好', () => {
    const freeze = { left: ['feeCode'], right: [] };
    const result = orderColumnsByPreference([columns[0]], {
      order: ['feeCode'],
      hidden: [],
      freeze,
    });
    expect(result).toHaveLength(1);
    expect(result[0].fixed).toBe('left');
    expect(freeze).toEqual({ left: ['feeCode'], right: [] });
  });
});
