import { Table } from 'antd';
import { describe, expect, it } from 'vitest';
import {
  applyColumnFreeze,
  deriveColumnFreezeDefaults,
  orderColumnKeys,
  resolveColumnFreeze,
} from './freeze';

describe('逐字段多列冻结', () => {
  it('所有默认固定字段均保留，辅助列不写入偏好', () => {
    expect(
      deriveColumnFreezeDefaults([
        { key: 'index', fixed: 'left' },
        { key: 'a', fixed: 'left' },
        { key: 'b', fixed: true },
        { key: 'option', fixed: 'end' },
      ]),
    ).toEqual({ left: ['a', 'b'], right: ['option'] });
  });
  it('按身份冻结多列并排序，保留宽度、回调和原对象', () => {
    const onHeaderCell = () => ({ width: 180 });
    const columns = [
      { key: 'a', fixed: 'left' as const },
      { key: 'b', width: 180, onHeaderCell },
      { key: 'c' },
      { key: 'option', fixed: 'right' as const },
    ];
    const result = applyColumnFreeze(columns, {
      left: ['b', 'c'],
      right: ['option'],
    });
    expect(result.map((column) => column.key)).toEqual([
      'b',
      'c',
      'a',
      'option',
    ]);
    expect(result.map((column) => column.fixed)).toEqual([
      'left',
      'left',
      false,
      'right',
    ]);
    expect(result[0].onHeaderCell).toBe(onHeaderCell);
    expect(result[0].width).toBe(180);
    expect(columns[0].fixed).toBe('left');
  });
  it('哨兵、序号和显式结构锚点保持，操作列可解除固定', () => {
    const index = { key: 'index', valueType: 'index', fixed: 'left' as const };
    const result = applyColumnFreeze(
      [
        Table.SELECTION_COLUMN,
        Table.EXPAND_COLUMN,
        index,
        { key: 'a' },
        { key: 'b' },
        { key: 'custom' },
      ],
      { left: ['b'], right: [] },
      { structuralKeys: ['custom'] },
    );
    expect(result.slice(0, 3)).toEqual([
      Table.SELECTION_COLUMN,
      Table.EXPAND_COLUMN,
      index,
    ]);
    expect(
      result.slice(3).map((column) => ('key' in column ? column.key : '')),
    ).toEqual(['b', 'a', 'custom']);
  });
  it('校验空数组、非法值、未知字段和双侧重复，不转换旧开关', () => {
    const defaults = { left: ['a'], right: ['b'] };
    expect(resolveColumnFreeze({ left: [], right: [] }, defaults)).toEqual({
      left: [],
      right: [],
    });
    expect(
      resolveColumnFreeze(
        { left: ['a', 'a', 'unknown'], right: ['a', 'b'] },
        defaults,
        ['a', 'b'],
      ),
    ).toEqual(defaults);
    expect(resolveColumnFreeze({ left: null, right: [] }, defaults)).toEqual({
      left: ['a'],
      right: [],
    });
    expect(resolveColumnFreeze({ first: true, last: false }, defaults)).toEqual(
      defaults,
    );
    expect(
      orderColumnKeys(['a', 'b', 'c'], { left: ['c'], right: ['a'] }),
    ).toEqual(['c', 'b', 'a']);
  });
  it('自定义标识操作列与仅搜索列保持原锚点', () => {
    const result = applyColumnFreeze(
      [
        { key: 'a' },
        { key: 'commands', valueType: 'option' },
        { key: 'keyword', hideInTable: true },
        { key: 'b' },
      ],
      { left: ['b'], right: ['commands'] },
    );
    expect(result.map((column) => column.key)).toEqual([
      'b',
      'commands',
      'keyword',
      'a',
    ]);
    expect(result[1]).toMatchObject({ fixed: 'right' });
  });
  it('分组叶列支持多列冻结，无稳定标识结构列不被猜测', () => {
    const columns = [
      { title: '资料', children: [{ key: 'a' }, { key: 'b' }] },
      { valueType: 'option', fixed: 'right' as const },
    ];
    expect(
      applyColumnFreeze(columns, { left: ['a', 'b'], right: [] })[0],
    ).toMatchObject({
      children: [
        { key: 'a', fixed: 'left' },
        { key: 'b', fixed: 'left' },
      ],
    });
  });
});
