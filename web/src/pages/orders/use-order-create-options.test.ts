import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  clearOrderMasterDataCache,
  getOrderPersonnelOptions,
} from '@/utils/order-options-cache';
import { fetchOrderMasterData, searchOrderLocations } from './common';
import { seaExportDefinition } from './order-kinds/sea-export/definition';
import { useOrderCreateOptions } from './use-order-create-options';

let mockCurrentUser: any = {
  id: 'user-1',
  currentOrganization: { id: 'org-1', name: '组织1' },
};

vi.mock('@umijs/max', () => ({
  useModel: (model: string) => {
    if (model === '@@initialState') {
      return {
        initialState: { currentUser: mockCurrentUser },
      };
    }
    return {};
  },
}));

vi.mock('@/utils/order-options-cache', () => ({
  clearOrderMasterDataCache: vi.fn(),
  getOrderPersonnelOptions: vi.fn(),
}));

vi.mock('./common', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./common')>();
  return {
    ...actual,
    fetchOrderMasterData: vi.fn(),
    searchOrderLocations: vi.fn(),
  };
});

const mockFetchMasterData = vi.mocked(fetchOrderMasterData);
const mockSearchLocations = vi.mocked(searchOrderLocations);
const mockGetPersonnel = vi.mocked(getOrderPersonnelOptions);
const mockClearCache = vi.mocked(clearOrderMasterDataCache);

const seaConfig = seaExportDefinition;

/**
 * React Query 迁移后的 hook 测试包装：每个用例独立 QueryClient，
 * 防止缓存串味（与 renderWithClient 同策略，但以 wrapper 形式供 renderHook 使用）。
 */
function createHookWrapper() {
  const queryClient = createTestQueryClient();
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(
      QueryClientProvider,
      { client: queryClient },
      React.createElement(App, null, children),
    );
  return { queryClient, wrapper };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: any) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

const mockMasterDataSuccess = {
  serviceTypeOptions: [
    { code: 'BOOKING', label: '订舱', value: 'st-booking' },
    { code: 'TRUCKING', label: '拖车', value: 'st-trucking' },
    { code: 'STUFFING', label: '内装', value: 'st-stuffing' },
    { code: 'CUSTOMS_EXPORT', label: '报关', value: 'st-customs' },
    { code: 'CUSTOMS_IMPORT', label: '清关', value: 'st-clearance' },
    { code: 'OVERSEA_SEGMENT', label: '海外段', value: 'st-oversea' },
    { code: 'INSURANCE', label: '保险', value: 'st-insurance' },
    { code: 'PALLET_CHARTER', label: '包板', value: 'st-pallet' },
    { code: 'CONTAINER_LEASE', label: '租箱', value: 'st-container' },
    { code: 'FUMIGATION', label: '熏蒸', value: 'st-fumigation' },
    { code: 'DOC_BUY', label: '买单', value: 'st-doc-buy' },
    { code: 'CERTIFICATE', label: '办证', value: 'st-certificate' },
    { code: 'DOC_PREP', label: '制单', value: 'st-doc-prep' },
    { code: 'DANGEROUS_SERVICE', label: '危险品', value: 'st-dangerous' },
    { code: 'OVERWEIGHT_SERVICE', label: '超重', value: 'st-overweight' },
    { code: 'DOCUMENT_EXCHANGE', label: '换单', value: 'st-doc-exchange' },
    { code: 'WAREHOUSING', label: '仓储', value: 'st-warehousing' },
    { code: 'INSPECTION', label: '报检', value: 'st-inspection' },
    { code: 'CONTAINER_PURCHASE', label: '买箱', value: 'st-container-buy' },
  ],
  cargoCategoryOptions: [
    { code: 'GENERAL', label: '普货', value: 'cargo-gen' },
  ],
  seaLocationOptions: [{ label: '上海港 (CNSHA)', value: 'loc-1' }],
  airLocationOptions: [{ label: '上海浦东 (PVG)', value: 'loc-2' }],
  currencyOptions: [{ label: 'CNY - 人民币', value: 'CNY' }],
  masterOptions: [],
  ports: [],
  airports: [],
  currencies: [],
};

describe('useOrderCreateOptions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-1', name: '组织1' },
    };
    mockFetchMasterData.mockResolvedValue(mockMasterDataSuccess);
    mockSearchLocations.mockResolvedValue([]);
    mockGetPersonnel.mockResolvedValue([
      { userId: 'u-1', displayName: '张三', organizationId: 'org-1' } as any,
    ]);
  });

  it('config 为空时，loading 为 false 且无错误', () => {
    const { result } = renderHook(() => useOrderCreateOptions(undefined), {
      wrapper: createHookWrapper().wrapper,
    });
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it('用户已登录但无有效组织时，不发起请求并展示明确错误', async () => {
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: null,
    };

    const { result } = renderHook(() => useOrderCreateOptions(seaConfig), {
      wrapper: createHookWrapper().wrapper,
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error?.message).toBe(
      '缺少当前组织，无法加载订单主数据',
    );
    expect(mockFetchMasterData).not.toHaveBeenCalled();
  });

  it('具备有效组织时正常加载海运主数据与人员选项', async () => {
    const { result } = renderHook(() => useOrderCreateOptions(seaConfig), {
      wrapper: createHookWrapper().wrapper,
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBeNull();
    expect(result.current.serviceTypeOptions).toHaveLength(19);
    expect(result.current.cargoCategoryOptions).toEqual([
      { code: 'GENERAL', label: '普货', value: 'cargo-gen' },
    ]);
    expect(result.current.personnelOptions).toEqual([
      expect.objectContaining({ userId: 'u-1', displayName: '张三' }),
    ]);
    expect(mockFetchMasterData).toHaveBeenCalledWith('org-1', 'sea');
    expect(mockGetPersonnel).toHaveBeenCalledWith(
      'org-1',
      seaConfig.businessType,
    );
  });

  it('接口加载失败时设置 error 状态', async () => {
    mockFetchMasterData.mockRejectedValueOnce(new Error('数据字典加载超时'));

    const { result } = renderHook(() => useOrderCreateOptions(seaConfig), {
      wrapper: createHookWrapper().wrapper,
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error?.message).toBe('数据字典加载超时');
  });

  it('重试操作先失效当前组织缓存，再重新发起请求并成功恢复', async () => {
    mockFetchMasterData
      .mockRejectedValueOnce(new Error('主数据拉取失败'))
      .mockResolvedValueOnce(mockMasterDataSuccess);

    const { result } = renderHook(() => useOrderCreateOptions(seaConfig), {
      wrapper: createHookWrapper().wrapper,
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBeTruthy();

    // 点击重试
    act(() => {
      result.current.retry();
    });

    expect(mockClearCache).toHaveBeenCalledWith('org-1');
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBeNull();
    expect(result.current.serviceTypeOptions).toHaveLength(19);
  });

  it('组织切换竞态：组织 A 请求慢，组织 B 请求快，组织 A 的迟到响应不得覆盖组织 B 的状态', async () => {
    const deferOrgA = deferred<any>();
    const deferOrgB = deferred<any>();

    mockFetchMasterData
      .mockImplementationOnce(() => deferOrgA.promise)
      .mockImplementationOnce(() => deferOrgB.promise);

    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };

    const { result, rerender } = renderHook(
      () => useOrderCreateOptions(seaConfig),
      { wrapper: createHookWrapper().wrapper },
    );

    expect(result.current.loading).toBe(true);

    // 切换至组织 B
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();

    // 组织 B 率先返回
    deferOrgB.resolve({
      ...mockMasterDataSuccess,
      cargoCategoryOptions: [
        { code: 'GENERAL', label: '普货-组织B', value: 'cargo-B' },
      ],
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.cargoCategoryOptions[0].value).toBe('cargo-B');

    // 随后组织 A 迟到的响应到达
    deferOrgA.resolve({
      ...mockMasterDataSuccess,
      cargoCategoryOptions: [
        { code: 'GENERAL', label: '普货-组织A', value: 'cargo-A' },
      ],
    });

    // 验证状态保持为组织 B，A 的响应已被成功丢弃
    await waitFor(() =>
      expect(result.current.cargoCategoryOptions[0].value).toBe('cargo-B'),
    );
  });

  it('组织切换时，已落入 React state 的旧组织数据立即隐藏，绝不在新组织渲染中暴露', async () => {
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };
    mockFetchMasterData.mockResolvedValueOnce({
      ...mockMasterDataSuccess,
      cargoCategoryOptions: [
        { code: 'GENERAL', label: '普货-组织A', value: 'cargo-A' },
      ],
    });

    const { result, rerender } = renderHook(
      () => useOrderCreateOptions(seaConfig),
      { wrapper: createHookWrapper().wrapper },
    );

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.cargoCategoryOptions[0].value).toBe('cargo-A');

    // 组织 A 已经渲染，此时切换到组织 B，组织 B 处于加载中
    const deferOrgB = deferred<any>();
    mockFetchMasterData.mockImplementationOnce(() => deferOrgB.promise);

    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();

    // 在组织 B 响应前，必须立即进入 loading 态，且选项必须为空，严防组织 A 数据闪现
    expect(result.current.loading).toBe(true);
    expect(result.current.cargoCategoryOptions).toEqual([]);
    expect(result.current.serviceTypeOptions).toEqual([]);

    // 组织 B 响应后，正常展示组织 B 数据
    deferOrgB.resolve({
      ...mockMasterDataSuccess,
      cargoCategoryOptions: [
        { code: 'GENERAL', label: '普货-组织B', value: 'cargo-B' },
      ],
    });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.cargoCategoryOptions[0].value).toBe('cargo-B');
  });

  it('地点搜索为空或仅含空白时复用当前组织首批候选项，不发起远程请求', async () => {
    const { result } = renderHook(() => useOrderCreateOptions(seaConfig), {
      wrapper: createHookWrapper().wrapper,
    });

    await waitFor(() => expect(result.current.loading).toBe(false));

    await expect(result.current.searchLocations()).resolves.toEqual(
      mockMasterDataSuccess.seaLocationOptions,
    );
    await expect(result.current.searchLocations('   ')).resolves.toEqual(
      mockMasterDataSuccess.seaLocationOptions,
    );
    expect(mockSearchLocations).not.toHaveBeenCalled();
  });

  it('地点搜索有真实关键字时继续调用服务端联想', async () => {
    mockSearchLocations.mockResolvedValueOnce([
      { label: '上海港远程结果', value: 'remote-port' },
    ]);
    const { result } = renderHook(() => useOrderCreateOptions(seaConfig), {
      wrapper: createHookWrapper().wrapper,
    });

    await waitFor(() => expect(result.current.loading).toBe(false));

    await expect(result.current.searchLocations(' shang ')).resolves.toEqual([
      { label: '上海港远程结果', value: 'remote-port' },
    ]);
    expect(mockSearchLocations).toHaveBeenCalledWith('sea', ' shang ');
  });

  it('组织切换后，旧组织迟到的地点搜索结果对调用方返回空数组', async () => {
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };
    const { result, rerender } = renderHook(
      () => useOrderCreateOptions(seaConfig),
      { wrapper: createHookWrapper().wrapper },
    );
    await waitFor(() => expect(result.current.loading).toBe(false));

    const delayedSearch = deferred<{ label: string; value: string }[]>();
    mockSearchLocations.mockImplementationOnce(() => delayedSearch.promise);
    const searchPromise = result.current.searchLocations('旧组织港口');

    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();
    // 组织切换触发的主数据重载在 act 内落地
    await act(async () => {});

    delayedSearch.resolve([{ label: '旧组织港口', value: 'old-port' }]);
    // 迟到搜索结果回调在 act 内收敛，避免用例结束后迟到更新
    await act(async () => {
      await searchPromise;
    });

    await expect(searchPromise).resolves.toEqual([]);
  });

  it('同组织切换业务配置时，空查询和旧配置迟到搜索均不得返回旧候选项', async () => {
    let currentConfig = seaConfig;
    const { result, rerender } = renderHook(
      () => useOrderCreateOptions(currentConfig),
      { wrapper: createHookWrapper().wrapper },
    );
    await waitFor(() => expect(result.current.loading).toBe(false));

    const delayedSearch = deferred<{ label: string; value: string }[]>();
    mockSearchLocations.mockImplementationOnce(() => delayedSearch.promise);
    const searchPromise = result.current.searchLocations('旧业务地点');

    const nextLoad = deferred<any>();
    mockFetchMasterData.mockImplementationOnce(() => nextLoad.promise);
    currentConfig = {
      ...seaConfig,
      businessType: (seaConfig.businessType +
        1) as typeof seaConfig.businessType,
    };
    rerender();

    await expect(result.current.searchLocations('   ')).resolves.toEqual([]);
    delayedSearch.resolve([{ label: '旧业务地点', value: 'old-location' }]);
    await expect(searchPromise).resolves.toEqual([]);

    nextLoad.resolve(mockMasterDataSuccess);
    await waitFor(() => expect(result.current.loading).toBe(false));
  });

  it('组织 A 加载失败后切到 B，B 首次渲染立即隐藏 A 的错误并进入加载态', async () => {
    const orgBRequest = deferred<any>();
    mockFetchMasterData
      .mockRejectedValueOnce(new Error('组织 A 主数据失败'))
      .mockImplementationOnce(() => orgBRequest.promise);
    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-A', name: '组织A' },
    };
    const renderSnapshots: Array<{
      organizationId?: string;
      loading: boolean;
      error: Error | null;
    }> = [];
    const { result, rerender } = renderHook(
      () => {
        const state = useOrderCreateOptions(seaConfig);
        renderSnapshots.push({
          organizationId: mockCurrentUser.currentOrganization?.id,
          loading: state.loading,
          error: state.error,
        });
        return state;
      },
      { wrapper: createHookWrapper().wrapper },
    );

    await waitFor(() =>
      expect(result.current.error?.message).toBe('组织 A 主数据失败'),
    );

    mockCurrentUser = {
      id: 'user-1',
      currentOrganization: { id: 'org-B', name: '组织B' },
    };
    rerender();

    const firstOrgBSnapshot = renderSnapshots.find(
      (snapshot) => snapshot.organizationId === 'org-B',
    );
    expect(firstOrgBSnapshot).toEqual(
      expect.objectContaining({ loading: true, error: null }),
    );

    orgBRequest.resolve(mockMasterDataSuccess);
    await waitFor(() => expect(result.current.loading).toBe(false));
  });
});
