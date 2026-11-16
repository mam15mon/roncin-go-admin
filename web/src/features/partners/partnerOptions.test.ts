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

  it('筛选条件和传输选项分离，失败信封不落入 data TypeError', async () => {
    const options = {
      signal: new AbortController().signal,
      skipErrorHandler: true,
    };
    listPartners.mockResolvedValueOnce({ success: true });
    expect(await searchPartnerOptions('客户', { role: 1 }, options)).toEqual(
      [],
    );
    expect(listPartners).toHaveBeenCalledWith(
      expect.objectContaining({ keyword: '客户', role: 1 }),
      options,
    );
    listPartners.mockResolvedValueOnce(undefined);
    await expect(searchPartnerOptions('客户', {}, options)).rejects.toThrow(
      '往来单位候选加载失败',
    );
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

    expect(listPartners.mock.calls[0][0]).toEqual({
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
