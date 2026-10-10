import { describe, expect, it } from 'vitest';
import {
  releaseFixedColumns,
  shouldReleaseFixedColumns,
  visibleColumnsWidth,
} from './layout';

describe('表格容器固定列取舍', () => {
  it('有效列宽决定滚动宽度，未设宽的可换行列不被max-content拉成巨宽', () => {
    expect(
      visibleColumnsWidth([
        { width: 200 },
        { width: 140 },
        { fixed: false },
        { width: 100 },
        { width: 170 },
      ]),
    ).toBe(770);
    expect(
      visibleColumnsWidth([
        { width: 900, hidden: true },
        { width: 200 },
        { children: [{ width: '120px' }, { fixed: false }] },
      ]),
    ).toBe(480);
    expect(visibleColumnsWidth([{ width: '28%' }])).toBe('max-content');
  });

  it('释放固定保留官方选择和展开哨兵对象身份', () => {
    const sentinel = {};
    expect(releaseFixedColumns([sentinel])[0]).toBe(sentinel);
    expect(visibleColumnsWidth([sentinel])).toBe(0);
  });
  it('实际窄容器取消固定区，宽容器保留合理固定区', () => {
    const columns = [
      { width: 220, fixed: 'left' as const },
      { width: 120, fixed: 'right' as const },
    ];
    expect(shouldReleaseFixedColumns(columns, 299, 32)).toBe(true);
    expect(shouldReleaseFixedColumns(columns, 815, 32)).toBe(false);
  });

  it('固定列过多时即便桌面容器也释放中间阅读空间', () => {
    expect(shouldReleaseFixedColumns([{ width: 700, fixed: true }], 1000)).toBe(
      true,
    );
    expect(shouldReleaseFixedColumns([{ width: 780, fixed: true }], 1000)).toBe(
      true,
    );
  });

  it('隐藏列不占固定区，支持分组继承固定和像素字符串', () => {
    const columns = [
      { width: 900, fixed: true, hidden: true },
      { width: 900, fixed: true, hideInTable: true },
      {
        fixed: 'left' as const,
        children: [{ width: '160px' }, { width: 160 }],
      },
    ];
    expect(shouldReleaseFixedColumns(columns, 815)).toBe(false);
    expect(shouldReleaseFixedColumns(columns, 650, 100)).toBe(true);
  });

  it('释放固定不修改原列定义、显隐、顺序和分组内容', () => {
    const render = () => '完整内容';
    const columns = [
      {
        key: 'group',
        fixed: 'left' as const,
        children: [{ key: 'a', width: 180, fixed: true, render }],
      },
      { key: 'b', hidden: true },
    ];
    const result = releaseFixedColumns(columns);
    expect(result.map((column) => column.key)).toEqual(['group', 'b']);
    expect(result[0].fixed).toBe(false);
    expect(result[0].children?.[0].fixed).toBe(false);
    expect(result[0].children?.[0].render).toBe(render);
    expect(result[1].hidden).toBe(true);
    expect(columns[0].fixed).toBe('left');
    expect(columns[0].children?.[0].fixed).toBe(true);
  });
});
