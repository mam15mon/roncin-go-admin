import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderTerminationStatus } from '@/enums.generated';
import { queryOrderList } from './list-query';
import { seaExportDefinition } from './order-kinds/sea-export/definition';

const listOrdersMock = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceListOrders: listOrdersMock,
}));

describe('queryOrderList', () => {
  beforeEach(() => {
    listOrdersMock.mockReset();
  });

  it.each([
    ['customer_reference', 4],
    ['booking', 5],
  ] as const)('映射新增号码筛选类型 %s', async (numberType, expectedType) => {
    listOrdersMock.mockResolvedValue({ data: [], total: 0, success: true });

    await queryOrderList(
      { numberType, numberKeyword: 'REF-001' },
      seaExportDefinition,
      { ports: [], airports: [], customerMap: {}, containerSpecMap: {} },
    );

    expect(listOrdersMock).toHaveBeenCalledWith(
      expect.objectContaining({
        numberType: expectedType,
        numberKeyword: 'REF-001',
      }),
    );
  });

  it('将页面筛选条件映射为订单查询参数', async () => {
    listOrdersMock.mockResolvedValue({ data: [], total: 0, success: true });

    await queryOrderList(
      {
        page: 2,
        pageSize: 50,
        stage: 'returned',
        numberType: 'consolidated_master',
        numberKeyword: 'MBL-001',
        isLocked: 'locked',
        shareStatus: 'unshared',
        tagIds: ['tag-1'],
      },
      seaExportDefinition,
      { ports: [], airports: [], customerMap: {}, containerSpecMap: {} },
    );

    expect(listOrdersMock).toHaveBeenCalledWith(
      expect.objectContaining({
        page: 2,
        pageSize: 50,
        businessType: 1,
        terminationStatus:
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
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
      seaExportDefinition,
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
          orderKind: 'sea-export',
          businessType: '海运出口',
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

  it('服务端名称投影优先，本地缓存缺失时兜底且任何情况不回退原始 ID', async () => {
    listOrdersMock.mockResolvedValue({
      data: [
        {
          id: 'order-1',
          orderNo: 'SE-001',
          customerId: 'customer-1',
          customerName: '服务端客户名',
          originLocationId: 'port-1',
          originLocationName: '上海港 (CNSHA)',
          destinationLocationId: 'port-2',
          flowStatus: 2,
        },
        {
          id: 'order-2',
          orderNo: 'SE-002',
          customerId: 'customer-2',
          destinationLocationId: 'port-2',
        },
      ],
      total: 2,
      success: true,
    });

    const result = await queryOrderList(
      { page: 1, pageSize: 20 },
      seaExportDefinition,
      {
        ports: [{ id: 'port-2', nameZh: '青岛港', unLocode: 'CNTAO' }],
        airports: [],
        customerMap: { 'customer-2': '本地缓存客户' },
        containerSpecMap: {},
      },
    );

    const [serverRow, fallbackRow] = result.data;
    expect(serverRow.customerName).toBe('服务端客户名');
    expect(serverRow.originPortName).toBe('上海港 (CNSHA)');
    // 服务端名称已含代码，不得再拼第二段代码。
    expect(serverRow.originPortCode).toBeUndefined();
    expect(fallbackRow.customerName).toBe('本地缓存客户');
    expect(fallbackRow.destinationPortName).toBe('青岛港');
    expect(fallbackRow.destinationPortCode).toBe('CNTAO');
    // 全部行均不得出现原始 UUID。
    expect(
      result.data.some(
        (row) =>
          row.customerName === 'customer-2' ||
          row.destinationPortName === 'port-2',
      ),
    ).toBe(false);
  });

  it('业务类型列展示注册定义的导航标题而非页面主标题', async () => {
    listOrdersMock.mockResolvedValue({ data: [], total: 0, success: true });

    const result = await queryOrderList(
      { page: 1, pageSize: 20 },
      seaExportDefinition,
      { ports: [], airports: [], customerMap: {}, containerSpecMap: {} },
    );

    expect(result.data).toEqual([]);
    expect(seaExportDefinition.title).toBe('海运出口订单');
    expect(seaExportDefinition.navigationTitle).toBe('海运出口');
  });
});
