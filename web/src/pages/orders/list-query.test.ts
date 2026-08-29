import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ORDER_KIND_CONFIGS } from './common';
import { queryOrderList } from './list-query';

const listOrdersMock = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceListOrders: listOrdersMock,
}));

describe('queryOrderList', () => {
  beforeEach(() => {
    listOrdersMock.mockReset();
  });

  it('将页面筛选条件映射为订单查询参数', async () => {
    listOrdersMock.mockResolvedValue({ data: [], total: 0, success: true });

    await queryOrderList(
      {
        page: 2,
        pageSize: 50,
        stage: 'abnormal',
        numberType: 'consolidated_master',
        numberKeyword: 'MBL-001',
        isLocked: 'locked',
        shareStatus: 'unshared',
        tagIds: ['tag-1'],
      },
      ORDER_KIND_CONFIGS['sea-export'],
      { ports: [], airports: [], customerMap: {}, containerSpecMap: {} },
    );

    expect(listOrdersMock).toHaveBeenCalledWith(
      expect.objectContaining({
        page: 2,
        pageSize: 50,
        businessType: 1,
        hasActiveException: true,
        numberType: 3,
        numberKeyword: 'MBL-001',
        isLocked: true,
        isShared: false,
        tagIds: ['tag-1'],
      }),
    );
  });

  it('使用港口、机场、客户和箱型主数据构造列表行', async () => {
    listOrdersMock.mockResolvedValue({
      data: [
        {
          id: 'order-1',
          orderNo: 'SE-001',
          customerId: 'customer-1',
          originLocationId: 'port-1',
          destinationLocationId: 'airport-1',
          flowStatus: 2,
          paymentTerm: 1,
          tradeTerm: 3,
          containerRequests: [
            { quantity: 2, containerSpecId: 'container-spec-1' },
          ],
        },
      ],
      total: 1,
      success: true,
    });

    const result = await queryOrderList(
      { page: 1, pageSize: 20 },
      ORDER_KIND_CONFIGS['sea-export'],
      {
        ports: [
          {
            id: 'port-1',
            nameZh: '上海港',
            nameEn: 'Shanghai',
            unLocode: 'CNSHA',
          },
        ],
        airports: [
          {
            id: 'airport-1',
            nameZh: '洛杉矶机场',
            nameEn: 'Los Angeles',
            iataCode: 'LAX',
          },
        ],
        customerMap: { 'customer-1': '示例客户 (CUS001)' },
        containerSpecMap: { 'container-spec-1': '40HQ' },
      },
    );

    expect(result).toEqual({
      data: [
        expect.objectContaining({
          id: 'order-1',
          orderNo: 'SE-001',
          customerName: '示例客户 (CUS001)',
          originPortName: '上海港',
          originPortCode: 'CNSHA',
          destinationPortName: '洛杉矶机场',
          destinationPortCode: 'LAX',
          containerSummary: '2×40HQ',
          paymentTerm: '预付 (PP)',
          tradeTerm: 'FOB',
          statusName: '已订舱',
          stage: '正常运作',
        }),
      ],
      total: 1,
      success: true,
    });
  });
});
