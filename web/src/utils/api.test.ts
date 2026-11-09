import { describe, expect, it } from 'vitest';
import { isWriteResponse, toTableRequest, unwrapList, unwrapPage } from './api';

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
    expect(
      toTableRequest({ data: [{ id: '1' }], success: false, total: '3' }),
    ).toEqual({
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
  it('写入必须取得成功信封或有效实体，不将空对象视为成功', () => {
    expect(isWriteResponse({})).toBe(false);
    expect(isWriteResponse({ success: true })).toBe(true);
    expect(isWriteResponse({ success: false, data: { id: 'id' } }, true)).toBe(
      false,
    );
    expect(isWriteResponse({ data: { id: 'id' } }, true)).toBe(true);
    expect(isWriteResponse({ data: {} }, true)).toBe(false);
  });
});
