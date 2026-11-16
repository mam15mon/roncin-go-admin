import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  masterDataServiceListAirports,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
} from '@/services/roncin/masterDataService';
import { orderServiceListPersonnelOptions } from '@/services/roncin/orderService';
import {
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

const listOptions = vi.mocked(masterDataServiceListOptions);
const listPorts = vi.mocked(masterDataServiceListPorts);
const listAirports = vi.mocked(masterDataServiceListAirports);
const listPersonnel = vi.mocked(orderServiceListPersonnelOptions);

beforeEach(() => vi.resetAllMocks());

describe('订单候选纯取数入口', () => {
  it('身份缺失时拒绝请求', async () => {
    await expect(getMasterDataOptions('')).rejects.toThrow('缺少当前组织');
    await expect(getCachedPorts('')).rejects.toThrow('缺少当前组织');
    await expect(getCachedAirports('')).rejects.toThrow('缺少当前组织');
    await expect(getOrderPersonnelOptions('', 1)).rejects.toThrow(
      '缺少当前组织',
    );
    expect(listOptions).not.toHaveBeenCalled();
    expect(listPorts).not.toHaveBeenCalled();
    expect(listAirports).not.toHaveBeenCalled();
    expect(listPersonnel).not.toHaveBeenCalled();
  });

  it('助手不持有模块缓存，重新读取可取得新数据', async () => {
    listOptions
      .mockResolvedValueOnce({ data: [{ id: 'old' }] })
      .mockResolvedValueOnce({ data: [{ id: 'new' }] });
    expect(await getMasterDataOptions('org')).toEqual([{ id: 'old' }]);
    expect(await getMasterDataOptions('org')).toEqual([{ id: 'new' }]);
    expect(listOptions).toHaveBeenCalledTimes(2);
  });

  it('工作区客户端负责同键在途去重、失效重读和独立缓存', async () => {
    listOptions
      .mockResolvedValueOnce({ data: [{ id: 'first' }] })
      .mockResolvedValueOnce({ data: [{ id: 'fresh' }] })
      .mockResolvedValueOnce({ data: [{ id: 'other-workspace' }] });
    const firstClient = createTestQueryClient();
    const secondClient = createTestQueryClient();
    const options = {
      queryKey: ['orders', 'master-options', { organizationId: 'org' }],
      queryFn: () => getMasterDataOptions('org'),
      staleTime: Infinity,
    };
    const [first, duplicate] = await Promise.all([
      firstClient.fetchQuery(options),
      firstClient.fetchQuery(options),
    ]);
    expect(first).toEqual([{ id: 'first' }]);
    expect(duplicate).toBe(first);
    expect(listOptions).toHaveBeenCalledOnce();
    await firstClient.invalidateQueries({ queryKey: ['orders'] });
    expect(await firstClient.fetchQuery(options)).toEqual([{ id: 'fresh' }]);
    expect(await secondClient.fetchQuery(options)).toEqual([
      { id: 'other-workspace' },
    ]);
    firstClient.clear();
    secondClient.clear();
  });

  it('保留首批港口、机场及业务人员的请求参数', async () => {
    listPorts.mockResolvedValue({ success: true });
    listAirports.mockResolvedValue({ success: true });
    listPersonnel.mockResolvedValue({ success: true });
    expect(await getCachedPorts('org')).toEqual([]);
    expect(await getCachedAirports('org')).toEqual([]);
    expect(await getOrderPersonnelOptions('org', 2)).toEqual([]);
    expect(listPorts).toHaveBeenCalledWith({
      page: 1,
      pageSize: 50,
      enabled: true,
    });
    expect(listAirports).toHaveBeenCalledWith({
      page: 1,
      pageSize: 50,
      enabled: true,
    });
    expect(listPersonnel).toHaveBeenCalledWith({
      businessType: 2,
      page: 1,
      pageSize: 200,
    });
  });

  it.each([
    undefined,
    { success: false, data: [] },
    { success: true, data: {} },
    {},
  ])('非法列表信封 %j 拒绝为业务错误', async (response) => {
    listOptions.mockResolvedValueOnce(response as API.ListOptionsResponse);
    await expect(getMasterDataOptions('org')).rejects.toThrow(
      '主数据选项加载失败',
    );
  });

  it('失败不污染下一次读取', async () => {
    listAirports
      .mockRejectedValueOnce(new Error('网络超时'))
      .mockResolvedValueOnce({ data: [{ id: 'airport' }] });
    await expect(getCachedAirports('org')).rejects.toThrow('网络超时');
    await expect(getCachedAirports('org')).resolves.toEqual([
      { id: 'airport' },
    ]);
  });
});
