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
} from './orderOptionsCache';

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

describe('订单候选缓存', () => {
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

  it('港口与机场缓存同组织并发去重，不同组织互不共享', async () => {
    mockListPorts
      .mockResolvedValueOnce({ data: [{ id: 'port-org1' } as any] })
      .mockResolvedValueOnce({ data: [{ id: 'port-org2' } as any] });
    mockListAirports
      .mockResolvedValueOnce({ data: [{ id: 'air-org1' } as any] })
      .mockResolvedValueOnce({ data: [{ id: 'air-org2' } as any] });

    const [ports1, ports1Again, ports2] = await Promise.all([
      getCachedPorts('org-1'),
      getCachedPorts('org-1'),
      getCachedPorts('org-2'),
    ]);
    const [airports1, airports1Again, airports2] = await Promise.all([
      getCachedAirports('org-1'),
      getCachedAirports('org-1'),
      getCachedAirports('org-2'),
    ]);

    expect(ports1).toEqual([{ id: 'port-org1' }]);
    expect(ports1Again).toEqual([{ id: 'port-org1' }]);
    expect(ports2).toEqual([{ id: 'port-org2' }]);
    expect(airports1).toEqual([{ id: 'air-org1' }]);
    expect(airports1Again).toEqual([{ id: 'air-org1' }]);
    expect(airports2).toEqual([{ id: 'air-org2' }]);
    expect(mockListPorts).toHaveBeenCalledTimes(2);
    expect(mockListAirports).toHaveBeenCalledTimes(2);
  });

  it('人员候选项同键并发调用去重，只产生 1 次请求', async () => {
    mockListPersonnel.mockResolvedValue({
      data: [{ userId: 'u1', displayName: '销售' } as any],
    });

    const [first, second] = await Promise.all([
      getOrderPersonnelOptions('org-1', 1),
      getOrderPersonnelOptions('org-1', 1),
    ]);

    expect(first).toEqual([{ userId: 'u1', displayName: '销售' }]);
    expect(second).toEqual(first);
    expect(mockListPersonnel).toHaveBeenCalledTimes(1);
  });

  it('请求层错误被全局处理后 resolve undefined 时，转成明确业务错误并移出缓存', async () => {
    mockListOptions
      .mockResolvedValueOnce(undefined as unknown as API.ListOptionsResponse)
      .mockResolvedValueOnce({
        data: [{ id: 'opt-retry', name: '重试选项' } as any],
      });

    await expect(getMasterDataOptions('org-1')).rejects.toThrow(
      '主数据选项加载失败，请稍后重试',
    );

    // 失败后缓存条目已移除，下次调用重新发起请求而不是命中坏缓存
    const retryRes = await getMasterDataOptions('org-1');
    expect(retryRes).toEqual([{ id: 'opt-retry', name: '重试选项' }]);
    expect(mockListOptions).toHaveBeenCalledTimes(2);
  });

  it('人员选项请求 resolve undefined 时转成明确业务错误，不得抛 TypeError', async () => {
    mockListPersonnel.mockResolvedValueOnce(
      undefined as unknown as API.ListPersonnelOptionsResponse,
    );

    await expect(getOrderPersonnelOptions('org-1', 1)).rejects.toThrow(
      '订单人员选项加载失败，请稍后重试',
    );
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

  it('旧失败请求迟到 reject 时，不得误删已写入的新请求缓存', async () => {
    let rejectReq1!: (err: Error) => void;
    const p1 = new Promise<any>((_, reject) => {
      rejectReq1 = reject;
    });
    mockListOptions.mockImplementationOnce(() => p1);

    // 1. 发起请求 1，进入 pending
    const req1Promise = getMasterDataOptions('org-1');

    // 2. 在请求 1 尚未 reject 前，清理 org-1 缓存并启动请求 2
    clearOrderMasterDataCache('org-1');
    mockListOptions.mockResolvedValueOnce({
      data: [{ id: 'opt-2', name: '新选项' } as any],
    });
    const req2Promise = getMasterDataOptions('org-1');

    // 3. 请求 1 发生迟到错误
    rejectReq1(new Error('旧网络超时'));
    await expect(req1Promise).rejects.toThrow('旧网络超时');

    // 4. 请求 2 成功完成
    const res2 = await req2Promise;
    expect(res2).toEqual([{ id: 'opt-2', name: '新选项' }]);

    // 5. 再次调用 getMasterDataOptions('org-1')，应继续命中请求 2 缓存，而不会因旧请求的 catch 误删缓存重新触发第 3 次网络请求
    const resCached = await getMasterDataOptions('org-1');
    expect(resCached).toEqual([{ id: 'opt-2', name: '新选项' }]);
    expect(mockListOptions).toHaveBeenCalledTimes(2);
  });

  it('港口旧失败请求迟到 reject 时，不得误删按组织清理后的新缓存', async () => {
    let rejectReq1!: (err: Error) => void;
    mockListPorts.mockImplementationOnce(
      () =>
        new Promise<any>((_, reject) => {
          rejectReq1 = reject;
        }),
    );

    const stalePromise = getCachedPorts('org-1');

    // 旧请求仍在途时定向清理 org-1 并重新发起请求。
    clearOrderMasterDataCache('org-1');
    mockListPorts.mockResolvedValueOnce({ data: [{ id: 'port-new' } as any] });
    const freshPromise = getCachedPorts('org-1');

    rejectReq1(new Error('旧网络超时'));
    await expect(stalePromise).rejects.toThrow('旧网络超时');

    expect(await freshPromise).toEqual([{ id: 'port-new' }]);
    expect(await getCachedPorts('org-1')).toEqual([{ id: 'port-new' }]);
    expect(mockListPorts).toHaveBeenCalledTimes(2);
  });

  it('机场旧失败请求迟到 reject 时，不得误删全量清理后的新缓存', async () => {
    let rejectReq1!: (err: Error) => void;
    mockListAirports.mockImplementationOnce(
      () =>
        new Promise<any>((_, reject) => {
          rejectReq1 = reject;
        }),
    );

    const stalePromise = getCachedAirports('org-1');

    // 退出登录等全量清理后，新请求缓存不被旧失败清掉。
    clearOrderMasterDataCache();
    mockListAirports.mockResolvedValueOnce({
      data: [{ id: 'air-new' } as any],
    });
    const freshPromise = getCachedAirports('org-1');

    rejectReq1(new Error('旧网络超时'));
    await expect(stalePromise).rejects.toThrow('旧网络超时');

    expect(await freshPromise).toEqual([{ id: 'air-new' }]);
    expect(await getCachedAirports('org-1')).toEqual([{ id: 'air-new' }]);
    expect(mockListAirports).toHaveBeenCalledTimes(2);
  });

  it('人员旧失败请求迟到 reject 时，不得误删同键新缓存', async () => {
    let rejectReq1!: (err: Error) => void;
    mockListPersonnel.mockImplementationOnce(
      () =>
        new Promise<any>((_, reject) => {
          rejectReq1 = reject;
        }),
    );

    const stalePromise = getOrderPersonnelOptions('org-1', 1);

    clearOrderMasterDataCache('org-1');
    mockListPersonnel.mockResolvedValueOnce({
      data: [{ userId: 'u2', displayName: '新销售' } as any],
    });
    const freshPromise = getOrderPersonnelOptions('org-1', 1);

    rejectReq1(new Error('旧网络超时'));
    await expect(stalePromise).rejects.toThrow('旧网络超时');

    expect(await freshPromise).toEqual([
      { userId: 'u2', displayName: '新销售' },
    ]);
    expect(await getOrderPersonnelOptions('org-1', 1)).toEqual([
      { userId: 'u2', displayName: '新销售' },
    ]);
    expect(mockListPersonnel).toHaveBeenCalledTimes(2);
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
    mockListOptions.mockResolvedValueOnce({
      data: [{ id: 'opt-1-refetched' } as any],
    });
    const org1Data = await getMasterDataOptions('org-1');
    expect(org1Data[0].id).toBe('opt-1-refetched');
    expect(mockListOptions).toHaveBeenCalledTimes(3);
  });

  it('clearOrderMasterDataCache(orgId) 定向失效港口、机场与全部业务类型人员缓存，其他组织保持命中', async () => {
    mockListPorts
      .mockResolvedValueOnce({ data: [{ id: 'port-1' } as any] })
      .mockResolvedValueOnce({ data: [{ id: 'port-2' } as any] });
    mockListAirports.mockResolvedValueOnce({ data: [{ id: 'air-1' } as any] });
    mockListPersonnel
      .mockResolvedValueOnce({ data: [{ userId: 'u1' } as any] })
      .mockResolvedValueOnce({ data: [{ userId: 'u2' } as any] });

    await getCachedPorts('org-1');
    await getCachedPorts('org-2');
    await getCachedAirports('org-1');
    await getOrderPersonnelOptions('org-1', 1);
    await getOrderPersonnelOptions('org-1', 2);
    expect(mockListPorts).toHaveBeenCalledTimes(2);
    expect(mockListAirports).toHaveBeenCalledTimes(1);
    expect(mockListPersonnel).toHaveBeenCalledTimes(2);

    // 仅失效 org-1：该组织的港口、机场与两个业务类型的人员缓存全部重建；org-2 不受牵连。
    clearOrderMasterDataCache('org-1');

    mockListAirports.mockResolvedValueOnce({
      data: [{ id: 'air-1-new' } as any],
    });
    mockListPorts.mockResolvedValueOnce({
      data: [{ id: 'port-1-new' } as any],
    });
    mockListPersonnel
      .mockResolvedValueOnce({ data: [{ userId: 'u1-new' } as any] })
      .mockResolvedValueOnce({ data: [{ userId: 'u2-new' } as any] });

    // org-2 港口依然命中缓存。
    expect(await getCachedPorts('org-2')).toEqual([{ id: 'port-2' }]);

    // org-1 的港口、机场与人员(1/2) 重新发起。
    expect(await getCachedAirports('org-1')).toEqual([{ id: 'air-1-new' }]);
    expect(await getCachedPorts('org-1')).toEqual([{ id: 'port-1-new' }]);
    expect(await getOrderPersonnelOptions('org-1', 1)).toEqual([
      { userId: 'u1-new' },
    ]);
    expect(await getOrderPersonnelOptions('org-1', 2)).toEqual([
      { userId: 'u2-new' },
    ]);

    expect(mockListPorts).toHaveBeenCalledTimes(3);
    expect(mockListAirports).toHaveBeenCalledTimes(2);
    expect(mockListPersonnel).toHaveBeenCalledTimes(4);
  });

  it('clearOrderMasterDataCache() 全量清理所有组织的缓存', async () => {
    mockListPorts
      .mockResolvedValueOnce({ data: [{ id: 'p1' } as any] })
      .mockResolvedValueOnce({ data: [{ id: 'p2' } as any] })
      .mockResolvedValueOnce({ data: [{ id: 'p1-new' } as any] })
      .mockResolvedValueOnce({ data: [{ id: 'p2-new' } as any] });
    mockListOptions
      .mockResolvedValueOnce({ data: [{ id: 'opt-1' } as any] })
      .mockResolvedValueOnce({ data: [{ id: 'opt-1-new' } as any] });

    await getCachedPorts('org-1');
    await getCachedPorts('org-2');
    await getMasterDataOptions('org-1');
    expect(mockListPorts).toHaveBeenCalledTimes(2);
    expect(mockListOptions).toHaveBeenCalledTimes(1);

    clearOrderMasterDataCache();

    await getCachedPorts('org-1');
    await getCachedPorts('org-2');
    await getMasterDataOptions('org-1');
    expect(mockListPorts).toHaveBeenCalledTimes(4);
    expect(mockListOptions).toHaveBeenCalledTimes(2);
  });
});
