import { describe, expect, it } from 'vitest';
import { toTableRequest, unwrapList, unwrapPage } from './api';

describe('API 响应解包', () => {
  it('将缺失的列表数据解包为空数组', () => {
    expect(unwrapList<{ id: string }>({})).toEqual([]);
  });

  it('将字符串分页总数转换为数字', () => {
    expect(unwrapPage({ data: [{ id: '1' }], total: '12' })).toEqual({
      data: [{ id: '1' }],
      total: 12,
    });
  });

  it('保留表格请求的失败状态和分页总数', () => {
    expect(toTableRequest({ data: [{ id: '1' }], success: false, total: '3' })).toEqual({
      data: [{ id: '1' }],
      success: false,
      total: 3,
    });
  });

  it('非分页响应不虚构总数', () => {
    expect(toTableRequest({ data: [{ id: '1' }] })).toEqual({
      data: [{ id: '1' }],
      success: true,
    });
  });
});
