import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { enterpriseResourceServiceListEnterpriseResourceRegionOptions } from '@/services/roncin/enterpriseResourceService';
import { useRegionOptions } from './useRegionOptions';

vi.mock('@/services/roncin/enterpriseResourceService', () => ({
  enterpriseResourceServiceListEnterpriseResourceRegionOptions: vi.fn(),
}));

const listRegions = vi.mocked(
  enterpriseResourceServiceListEnterpriseResourceRegionOptions,
);

function renderRegions(initialProps: Parameters<typeof useRegionOptions>[0]) {
  const client = createTestQueryClient();
  const wrapper = ({ children }: PropsWithChildren) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return { ...renderHook(useRegionOptions, { initialProps, wrapper }), client };
}

beforeEach(() => {
  listRegions.mockReset();
  listRegions.mockResolvedValue({ success: true });
});

describe('企业资源地区候选', () => {
  it('关闭、非地址类型或非中国地址不加载，打开后按层级及父级查询', async () => {
    const props = {
      editorOpen: false,
      isAddressTab: true,
      countryCode: 'CN',
      provinceCode: '31',
      cityCode: '3101',
    };
    const { result, rerender } = renderRegions(props);
    await act(async () => {});
    expect(listRegions).not.toHaveBeenCalled();
    rerender({ ...props, editorOpen: true, isAddressTab: false });
    await act(async () => {});
    expect(listRegions).not.toHaveBeenCalled();
    rerender({ ...props, editorOpen: true, countryCode: 'US' });
    await act(async () => {});
    expect(listRegions).not.toHaveBeenCalled();
    rerender({ ...props, editorOpen: true });
    await waitFor(() => expect(listRegions).toHaveBeenCalledTimes(3));
    expect(listRegions).toHaveBeenCalledWith(
      { level: 2, parentCode: '31', page: 1, pageSize: 200 },
      expect.objectContaining({ skipErrorHandler: true }),
    );
    expect(listRegions).toHaveBeenCalledWith(
      { level: 3, parentCode: '3101', page: 1, pageSize: 200 },
      expect.anything(),
    );
    expect(result.current.cityOptions).toEqual([]);
  });

  it('换省立即隔离旧城市候选，旧响应迟到也不覆盖当前省份', async () => {
    let resolveOld:
      | ((response: API.ListEnterpriseResourceRegionOptionsResponse) => void)
      | undefined;
    listRegions.mockImplementation(async (params) => {
      if (params.level !== 2) return { success: true };
      if (params.parentCode === '31')
        return new Promise((resolve) => {
          resolveOld = resolve;
        });
      return { success: true, data: [{ code: '4401', name: '广州市' }] };
    });
    const props = {
      editorOpen: true,
      isAddressTab: true,
      countryCode: 'CN',
      provinceCode: '31',
    };
    const { result, rerender } = renderRegions(props);
    await waitFor(() => expect(resolveOld).toBeDefined());
    rerender({ ...props, provinceCode: '44' });
    expect(result.current.cityOptions).toEqual([]);
    await waitFor(() =>
      expect(result.current.cityOptions).toEqual([
        { value: '4401', label: '广州市' },
      ]),
    );
    await act(async () => {
      resolveOld?.({ success: true, data: [{ code: '3101', name: '上海市' }] });
    });
    expect(result.current.cityOptions).toEqual([
      { value: '4401', label: '广州市' },
    ]);
  });

  it('失败响应进入查询错误，明确成功的空 repeated 可正常重试', async () => {
    listRegions.mockResolvedValueOnce(undefined as never);
    const { client, result } = renderRegions({
      editorOpen: true,
      isAddressTab: true,
      countryCode: 'CN',
    });
    await waitFor(() =>
      expect(
        client.getQueryCache().find({
          queryKey: ['enterprise-resources', 'region-options'],
          exact: false,
        })?.state.status,
      ).toBe('error'),
    );
    expect(result.current.provinceOptions).toEqual([]);
    await act(async () => {
      await client.refetchQueries({
        queryKey: ['enterprise-resources', 'region-options'],
      });
    });
    expect(
      client.getQueryCache().find({
        queryKey: ['enterprise-resources', 'region-options'],
        exact: false,
      })?.state.status,
    ).toBe('success');
    expect(result.current.provinceOptions).toEqual([]);
  });
});
