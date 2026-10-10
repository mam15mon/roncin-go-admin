import type { ProColumns } from '@ant-design/pro-components';
import { describe, expect, it, vi } from 'vitest';
import { getColumnFreezeTargets } from '@/components/ui/column-settings';
import { orderColumnsByPreference } from './orderFeeOptionalColumns';

describe('订单费用可见列的首尾冻结应用', () => {
  const onHeaderCell = vi.fn(() => ({ style: { width: 180 } }));
  const columns: ProColumns<API.OrderFee>[] = [
    { title: '费用代码', dataIndex: 'feeCode', width: 100 },
    { title: '费用名称', dataIndex: 'feeSettingId', width: 180, onHeaderCell },
    { title: '备注', dataIndex: 'note', width: 140 },
    { title: '操作', valueType: 'option', width: 120, fixed: 'right' },
  ];

  it('在当前显示排序后冻结首列，隐藏原首列后自动跟随；保留列宽与表头回调', () => {
    const result = orderColumnsByPreference(columns, {
      order: ['feeCode', 'feeSettingId', 'note', 'option'],
      hidden: ['feeCode'],
      freeze: { first: true, last: true },
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

  it('排序后的首列为冻结目标；操作列不能被偏好隐藏或排到中间', () => {
    const result = orderColumnsByPreference(columns, {
      order: ['option', 'note', 'feeCode', 'note', 'feeSettingId'],
      hidden: ['option'],
      freeze: { first: true, last: false },
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
    const targets = getColumnFreezeTargets(result);
    expect(targets.first?.title).toBe('备注');
    expect(targets.last?.title).toBe('操作');
    expect(columns.at(-1)?.fixed).toBe('right');
  });

  it('关闭首尾后解除普通列原固定，未配置时从业务列推导默认尾列冻结', () => {
    const order = ['feeCode', 'feeSettingId', 'note', 'option'] as const;
    const unfreeze = orderColumnsByPreference(columns, {
      order: [...order],
      hidden: [],
      freeze: { first: false, last: false },
    });
    expect(unfreeze.every((column) => !column.fixed)).toBe(true);
    const defaults = orderColumnsByPreference(columns, {
      order: [...order],
      hidden: [],
    });
    expect(defaults.at(-1)?.fixed).toBe('right');
    expect(defaults[0].fixed).toBe(false);
  });

  it('仅一个业务列时首尾重合左侧优先，不改写两个开关', () => {
    const freeze = { first: true, last: true };
    const result = orderColumnsByPreference([columns[0]], {
      order: ['feeCode'],
      hidden: [],
      freeze,
    });
    expect(result).toHaveLength(1);
    expect(result[0].fixed).toBe('left');
    expect(freeze).toEqual({ first: true, last: true });
  });
});
