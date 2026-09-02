import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  MASTER_DATA_KINDS,
  searchOrderLocations,
  searchPartnersByRole,
} from './common';

const listItems = vi.hoisted(() => vi.fn());
const listPorts = vi.hoisted(() => vi.fn());
const listAirports = vi.hoisted(() => vi.fn());
const listPartners = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListItems: listItems,
  masterDataServiceListPorts: listPorts,
  masterDataServiceListAirports: listAirports,
  masterDataServiceListCurrencies: vi.fn(),
  masterDataServiceListOptions: vi.fn(),
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

    expect(listItems).toHaveBeenCalledWith({
      kind: MASTER_DATA_KINDS.REGION,
      keyword: 'shang',
      enabled: true,
      page: 1,
      pageSize: 50,
    });
    expect(listPorts).toHaveBeenCalledWith({
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

  it('合作方联想查询只读取首批匹配结果', async () => {
    listPartners.mockResolvedValue({ data: [] });

    await searchPartnersByRole(1, '客户');

    expect(listPartners).toHaveBeenCalledWith({
      role: 1,
      enabled: true,
      keyword: '客户',
      page: 1,
      pageSize: 50,
    });
  });
});
