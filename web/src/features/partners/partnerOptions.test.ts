import { beforeEach, describe, expect, it, vi } from 'vitest';
import { searchPartnerOptions } from './partnerOptions';

const listPartners = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceListPartners: listPartners,
}));

describe('往来单位候选搜索', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('按角色和启用状态搜索合作方并统一标签', async () => {
    listPartners.mockResolvedValue({
      data: [
        { id: 'p1', code: 'CUS001', legalName: '示例客户', isCasual: true },
        { id: 'p2', code: 'CUS002', isCasual: false },
      ],
    });

    const result = await searchPartnerOptions('示例', {
      role: 1,
      enabled: true,
    });

    expect(listPartners).toHaveBeenCalledWith({
      page: 1,
      pageSize: 50,
      keyword: '示例',
      role: 1,
      enabled: true,
    });
    expect(result).toEqual([
      {
        label: '示例客户 (CUS001)',
        value: 'p1',
        code: 'CUS001',
        name: '示例客户',
        isCasual: true,
      },
      {
        label: 'CUS002',
        value: 'p2',
        code: 'CUS002',
        name: undefined,
        isCasual: false,
      },
    ]);
  });
});
