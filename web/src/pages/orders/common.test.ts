import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  isUnimplementedTransportMode,
  MASTER_DATA_KINDS,
  searchOrderLocations,
  searchPartnersByRole,
  searchShippingAgentOptions,
} from './common';

const listItems = vi.hoisted(() => vi.fn());
const listPorts = vi.hoisted(() => vi.fn());
const listAirports = vi.hoisted(() => vi.fn());
const listPartners = vi.hoisted(() => vi.fn());
const searchOptions = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListItems: listItems,
  masterDataServiceListPorts: listPorts,
  masterDataServiceListAirports: listAirports,
  masterDataServiceListCurrencies: vi.fn(),
  masterDataServiceListOptions: vi.fn(),
  masterDataServiceSearchOptions: searchOptions,
}));

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartners: listPartners,
}));

describe('订单远程候选项', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listItems.mockResolvedValue({
      data: [{ id: 'region-1', code: 'SHA', name: '上海' }],
    });
  });

  it('按关键字从服务端搜索地点与海运港口', async () => {
    listPorts.mockResolvedValue({
      data: [
        {
          id: 'port-1',
          unLocode: 'CNSHG',
          nameZh: '上海港',
          nameEn: 'Shanghai',
        },
      ],
    });

    const options = await searchOrderLocations('sea', 'shang');

    expect(listItems.mock.calls[0][0]).toEqual({
      kind: MASTER_DATA_KINDS.REGION,
      keyword: 'shang',
      enabled: true,
      page: 1,
      pageSize: 50,
    });
    expect(listPorts.mock.calls[0][0]).toEqual({
      keyword: 'shang',
      enabled: true,
      page: 1,
      pageSize: 50,
    });
    expect(options).toEqual([
      { label: '上海 (SHA)', value: 'region-1' },
      { label: '上海港 / Shanghai (CNSHG)', value: 'port-1' },
    ]);
  });

  it.each([undefined, '客户'])(
    '委托单位候选在服务端排除黑名单和停用角色（关键词：%s）',
    async (keyword) => {
      listPartners.mockResolvedValue({ data: [] });

      await searchPartnersByRole(1, keyword);

      expect(listPartners.mock.calls[0][0]).toEqual({
        role: 1,
        enabled: true,
        blacklisted: false,
        roleEnabled: true,
        keyword,
        page: 1,
        pageSize: 50,
      });
    },
  );

  it.each([2, 3])('其他合作角色 %s 保留原有候选范围', async (role) => {
    listPartners.mockResolvedValue({ data: [] });
    await searchPartnersByRole(role, '合作方');
    expect(listPartners.mock.calls[0][0]).toEqual({
      role,
      enabled: true,
      keyword: '合作方',
      page: 1,
      pageSize: 50,
    });
  });

  it('船代按拼音关键词查询公共字典，不读取公司供应商', async () => {
    searchOptions.mockResolvedValue({
      data: [{ id: 'agent-1', name: '上海船代', code: 'HGJ-1' }],
    });
    expect(await searchShippingAgentOptions('shcd')).toEqual([
      { label: '上海船代', value: 'agent-1', code: 'HGJ-1' },
    ]);
    expect(searchOptions).toHaveBeenCalledWith({
      kind: 11,
      keyword: 'shcd',
      page: 1,
      pageSize: 50,
    });
    expect(listPartners).not.toHaveBeenCalled();
  });

  it.each([undefined, { success: false }])(
    '船代搜索返回失败 %s 时显式报错',
    async (response) => {
      searchOptions.mockResolvedValue(response);
      await expect(searchShippingAgentOptions('shcd')).rejects.toThrow(
        '船代搜索失败，请稍后重试',
      );
    },
  );

  it.each(['land', 'rail'] as const)(
    '未开放的 %s 运输方式显式抛错，不落入机场分支',
    async (transportMode) => {
      expect(isUnimplementedTransportMode(transportMode)).toBe(true);
      expect(isUnimplementedTransportMode('sea')).toBe(false);
      expect(isUnimplementedTransportMode('air')).toBe(false);

      await expect(searchOrderLocations(transportMode, 'x')).rejects.toThrow(
        '陆运与铁路订单的地点主数据尚未开放',
      );
      expect(listPorts).not.toHaveBeenCalled();
      expect(listAirports).not.toHaveBeenCalled();
    },
  );

  it('港口/机场请求被全局错误处理器消费后 resolve undefined 时，转成明确业务错误而非 TypeError', async () => {
    listPorts.mockResolvedValueOnce(undefined as unknown as any);

    await expect(searchOrderLocations('sea', 'shang')).rejects.toThrow(
      '运输地点主数据搜索失败，请稍后重试',
    );
  });

  it('地区主数据请求 resolve undefined 时转成明确业务错误', async () => {
    listItems.mockResolvedValueOnce(undefined as unknown as any);
    listPorts.mockResolvedValue({ data: [] });

    await expect(searchOrderLocations('sea', 'shang')).rejects.toThrow(
      '地区主数据搜索失败，请稍后重试',
    );
  });
});
