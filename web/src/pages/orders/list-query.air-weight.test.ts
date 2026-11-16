import { describe, expect, it } from 'vitest';
import { mapOrderListItems } from './list-query';
import { airExportDefinition } from './order-kinds/air-export/definition';
import { seaExportDefinition } from './order-kinds/sea-export/definition';

const context = {
  organizationId: 'company-1',
  ports: [],
  airports: [],
  customerMap: {},
};

describe('航空列表真实wire映射', () => {
  it('HOUSE关系与两种来源独立，省略false和显式零保留语义', async () => {
    const [row] = await mapOrderListItems(
      [
        {
          id: 'order-1',
          airExport: {
            documentMode: 'HOUSE',
            house: { houseNo: 'H001', status: 'RELEASED' },
            master: {
              content: { masterNo: '78112345675', status: 'DRAFT' },
              memberCount: 2,
            },
            weightSummaries: [
              { source: 'PLANNED', confirmedWeightKg: '0' },
              {
                source: 'ACTUAL',
                confirmedWeightKg: '125',
                confirmationStale: true,
              },
            ],
          },
        },
      ],
      airExportDefinition,
      context,
    );
    expect(row.billRelation).toEqual({
      kind: 'house',
      houseNo: 'H001',
      memberCount: 2,
    });
    expect(row.airWeightSummaries).toMatchObject([
      { source: 'PLANNED', confirmedWeightKg: '0', confirmationStale: false },
      { source: 'ACTUAL', confirmedWeightKg: '125', confirmationStale: true },
    ]);
    expect(row.airMasterStatus).toBe('资料待完善');
    expect(row.airHouseStatus).toBe('已签发');
  });
  it('DIRECT不生成分单，无确认的省略集合保持为空', async () => {
    const [row] = await mapOrderListItems(
      [{ id: 'order-2', airExport: { documentMode: 'DIRECT' } }],
      airExportDefinition,
      context,
    );
    expect(row.billRelation).toEqual({
      kind: 'direct',
      houseNo: undefined,
      memberCount: 0,
    });
    expect(row.airWeightSummaries).toEqual([]);
    expect(row.airHouseStatus).toBe('直单 · 无分单');
    const [sea] = await mapOrderListItems(
      [{ id: 'sea' }],
      seaExportDefinition,
      context,
    );
    expect(sea.airWeightSummaries).toBeUndefined();
  });
  it.each([
    { source: 'PLANNED' },
    { source: 'OTHER', confirmedWeightKg: '100' },
    { source: 'ACTUAL', confirmedWeightKg: '-1' },
  ])('损坏摘要显式失败 %j', async (summary) => {
    await expect(
      mapOrderListItems(
        [{ id: 'bad', airExport: { weightSummaries: [summary] } }],
        airExportDefinition,
        context,
      ),
    ).rejects.toThrow('订单计费重量摘要不完整');
  });
});
