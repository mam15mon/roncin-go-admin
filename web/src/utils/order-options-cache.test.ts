import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  masterDataServiceListAirports,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
} from '@/services/roncin/masterDataService';
import { orderServiceListPersonnelOptions } from '@/services/roncin/orderService';
import {
  clearOrderMasterDataCache,
  getCachedAirports,
  getCachedPorts,
  getMasterDataOptions,
  getOrderPersonnelOptions,
} from './order-options-cache';

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListOptions: vi.fn(),
  masterDataServiceListPorts: vi.fn(),
  masterDataServiceListAirports: vi.fn(),
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceListPersonnelOptions: vi.fn(),
}));

const mockListOptions = vi.mocked(masterDataServiceListOptions);
const mockListPorts = vi.mocked(masterDataServiceListPorts);
const mockListAirports = vi.mocked(masterDataServiceListAirports);
const mockListPersonnel = vi.mocked(orderServiceListPersonnelOptions);

describe('order-options-cache', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clearOrderMasterDataCache();
  });

  it('缺少 organizationId 时应直接 reject 业务错误', async () => {
    await expect(getMasterDataOptions('')).rejects.toThrow(
      '缺少当前组织，无法加载订单主数据',
    );
    await expect(getCachedPorts('')).rejects.toThrow(
      '缺少当前组织，无法加载港口主数据',
    );
    await expect(getCachedAirports('')).rejects.toThrow(
      '缺少当前组织，无法加载机场主数据',
    );
    await expect(getOrderPersonnelOptions('', 1)).rejects.toThrow(
      '缺少当前组织，无法加载订单人员选项',
    );
  });

  it('同组织并发调用去重，只产生 1 次网络请求', async () => {
    mockListOptions.mockResolvedValue({
      data: [{ id: 'opt-1', name: '选项1' } as any],
    });

    const [res1, res2] = await Promise.all([
      getMasterDataOptions('org-1'),
      getMasterDataOptions('org-1'),
    ]);

    expect(res1).toEqual([{ id: 'opt-1', name: '选项1' }]);
    expect(res2).toEqual([{ id: 'opt-1', name: '选项1' }]);
    expect(mockListOptions).toHaveBeenCalledTimes(1);
  });

  it('同组织顺序调用命中缓存，不重复发起请求', async () => {
    mockListPorts.mockResolvedValue({
      data: [{ id: 'port-1', unLocode: 'CNSHA' } as any],
    });

    const first = await getCachedPorts('org-1');
    const second = await getCachedPorts('org-1');

    expect(first).toEqual([{ id: 'port-1', unLocode: 'CNSHA' }]);
    expect(second).toEqual([{ id: 'port-1', unLocode: 'CNSHA' }]);
    expect(mockListPorts).toHaveBeenCalledTimes(1);
  });

  it('请求失败自动移出缓存，下次调用重新发起', async () => {
    mockListAirports
      .mockRejectedValueOnce(new Error('网络超时'))
      .mockResolvedValueOnce({
        data: [{ id: 'air-1', iataCode: 'PVG' } as any],
      });

    await expect(getCachedAirports('org-1')).rejects.toThrow('网络超时');
    expect(mockListAirports).toHaveBeenCalledTimes(1);

    const retryRes = await getCachedAirports('org-1');
    expect(retryRes).toEqual([{ id: 'air-1', iataCode: 'PVG' }]);
    expect(mockListAirports).toHaveBeenCalledTimes(2);
  });

  it('不同组织隔离不共享缓存', async () => {
    mockListOptions
      .mockResolvedValueOnce({
        data: [{ id: 'opt-org1', name: '组织1选项' } as any],
      })
      .mockResolvedValueOnce({
        data: [{ id: 'opt-org2', name: '组织2选项' } as any],
      });

    const org1Data = await getMasterDataOptions('org-1');
    const org2Data = await getMasterDataOptions('org-2');

    expect(org1Data[0].id).toBe('opt-org1');
    expect(org2Data[0].id).toBe('opt-org2');
    expect(mockListOptions).toHaveBeenCalledTimes(2);
  });

  it('人员候选项按组织与业务类型分别缓存', async () => {
    mockListPersonnel
      .mockResolvedValueOnce({
        data: [{ userId: 'u1', displayName: '销售' } as any],
      })
      .mockResolvedValueOnce({
        data: [{ userId: 'u2', displayName: '空运销售' } as any],
      });

    const sePersonnel = await getOrderPersonnelOptions('org-1', 1);
    const aePersonnel = await getOrderPersonnelOptions('org-1', 2);
    const seCached = await getOrderPersonnelOptions('org-1', 1);

    expect(sePersonnel).toEqual([{ userId: 'u1', displayName: '销售' }]);
    expect(aePersonnel).toEqual([{ userId: 'u2', displayName: '空运销售' }]);
    expect(seCached).toEqual([{ userId: 'u1', displayName: '销售' }]);
    expect(mockListPersonnel).toHaveBeenCalledTimes(2);
  });

  it('clearOrderMasterDataCache(orgId) 定向失效目标组织缓存，其他组织保持命中', async () => {
    mockListOptions
      .mockResolvedValueOnce({
        data: [{ id: 'opt-1' } as any],
      })
      .mockResolvedValueOnce({
        data: [{ id: 'opt-2' } as any],
      })
      .mockResolvedValueOnce({
        data: [{ id: 'opt-1-refetched' } as any],
      });

    await getMasterDataOptions('org-1');
    await getMasterDataOptions('org-2');
    expect(mockListOptions).toHaveBeenCalledTimes(2);

    // 仅失效 org-1
    clearOrderMasterDataCache('org-1');

    // org-2 依然命中缓存
    const org2Data = await getMasterDataOptions('org-2');
    expect(org2Data[0].id).toBe('opt-2');
    expect(mockListOptions).toHaveBeenCalledTimes(2);

    // org-1 重新发起
    const org1Data = await getMasterDataOptions('org-1');
    expect(org1Data[0].id).toBe('opt-1-refetched');
    expect(mockListOptions).toHaveBeenCalledTimes(3);
  });

  it('clearOrderMasterDataCache() 全量清理所有组织的缓存', async () => {
    mockListPorts
      .mockResolvedValueOnce({ data: [{ id: 'p1' } as any] })
      .mockResolvedValueOnce({ data: [{ id: 'p2' } as any] })
      .mockResolvedValueOnce({ data: [{ id: 'p1-new' } as any] })
      .mockResolvedValueOnce({ data: [{ id: 'p2-new' } as any] });

    await getCachedPorts('org-1');
    await getCachedPorts('org-2');
    expect(mockListPorts).toHaveBeenCalledTimes(2);

    clearOrderMasterDataCache();

    await getCachedPorts('org-1');
    await getCachedPorts('org-2');
    expect(mockListPorts).toHaveBeenCalledTimes(4);
  });
});
