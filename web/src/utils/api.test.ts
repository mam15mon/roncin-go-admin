import { describe, expect, it } from 'vitest';
import {
  ensureListResponse,
  isWriteResponse,
  toTableRequest,
  unwrapList,
  unwrapPage,
} from './api';

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

describe('列表响应信封校验', () => {
  it.each([
    undefined,
    {},
    { success: false, data: [] },
    { success: true, data: null },
    { success: true, data: {} },
  ])('拒绝失败或非法响应 %j', (response) => {
    expect(() =>
      ensureListResponse(
        response as Parameters<typeof ensureListResponse>[0],
        '加载失败',
      ),
    ).toThrow('加载失败');
  });
  it('成功省略 repeated 保留合法空值，并保留分页元数据', () => {
    const response = { success: true, total: 0 };
    expect(ensureListResponse(response, '失败')).toBe(response);
    expect(unwrapList(ensureListResponse(response, '失败'))).toEqual([]);
    const page = { data: [{ id: 'one' }], total: 9 };
    expect(unwrapPage(ensureListResponse(page, '失败')).total).toBe(9);
  });
});
