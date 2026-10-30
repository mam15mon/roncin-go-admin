import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import { queryOrderList } from './list-query';
import { airExportDefinition } from './order-kinds/air-export/definition';
import { seaExportDefinition } from './order-kinds/sea-export/definition';

const listOrdersMock = vi.hoisted(() => vi.fn());
const masterOptionsMock = vi.hoisted(() => vi.fn());

vi.mock('@/features/orders/options', () => ({
  getMasterDataOptions: masterOptionsMock,
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceListOrders: listOrdersMock,
}));

describe('queryOrderList', () => {
  beforeEach(() => {
    listOrdersMock.mockReset();
    masterOptionsMock.mockReset().mockResolvedValue([]);
  });

  it.each([
    ['customer_reference', 4],
    ['booking', 5],
  ] as const)('映射新增号码筛选类型 %s', async (numberType, expectedType) => {
    listOrdersMock.mockResolvedValue({ data: [], total: 0, success: true });

    await queryOrderList(
      { numberType, numberKeyword: 'REF-001' },
      seaExportDefinition,
      { ports: [], airports: [], customerMap: {}, organizationId: 'org-1' },
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
      { ports: [], airports: [], customerMap: {}, organizationId: 'org-1' },
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
    expect(listOrdersMock.mock.calls[0][0]).not.toHaveProperty('closureStatus');
  });

  it.each([
    {
      name: '正常运作',
      terminationStatus: OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
      closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
      stage: ['正常运作'],
    },
    {
      name: '退关中',
      terminationStatus:
        OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING,
      closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
      stage: ['退关中'],
    },
    {
      name: '已退关',
      terminationStatus:
        OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
      closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
      stage: ['已退关'],
    },
    {
      name: '正常完结',
      terminationStatus: OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
      closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
      stage: ['已完结'],
    },
    {
      name: '退关后完结',
      terminationStatus:
        OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
      closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
      stage: ['已退关', '已完结'],
    },
  ])(
    '$name 的进程独立于异常信息',
    async ({ terminationStatus, closureStatus, stage }) => {
      listOrdersMock.mockResolvedValue({
        data: [false, true].map((hasActiveException, index) => ({
          id: `order-${index}`,
          terminationStatus,
          closureStatus,
          flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED,
          hasActiveException,
        })),
        total: 2,
        success: true,
      });

      const result = await queryOrderList({}, seaExportDefinition, {
        ports: [],
        airports: [],
        customerMap: {},
        organizationId: 'org-1',
      });

      expect(
        result.data.map((row) => ({
          stage: row.stage,
          statusName: row.statusName,
          abnormalLevel: row.abnormalLevel,
        })),
      ).toEqual([
        { stage, statusName: '已订舱', abnormalLevel: 'normal' },
        { stage, statusName: '已订舱', abnormalLevel: 'high' },
      ]);
    },
  );

  it.each([
    ['海运', seaExportDefinition, '已放单'],
    ['空运', airExportDefinition, '已签单'],
  ] as const)(
    '%s 操作节点不推断订单完结',
    async (_name, definition, statusName) => {
      listOrdersMock.mockResolvedValue({
        data: [
          {
            id: 'order-1',
            businessType: definition.businessType,
            flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_DOCUMENT_RELEASED,
            terminationStatus:
              OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
            closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
          },
        ],
        total: 1,
        success: true,
      });

      const result = await queryOrderList({}, definition, {
        ports: [],
        airports: [],
        customerMap: {},
        organizationId: 'org-1',
      });

      expect(result.data[0]).toEqual(
        expect.objectContaining({
          statusName,
          stage: ['正常运作'],
        }),
      );
    },
  );

  it('使用港口、机场、客户和箱型主数据构造列表行', async () => {
    masterOptionsMock.mockResolvedValue([
      { id: 'container-spec-1', kind: 7, code: '40HQ', name: '40尺高柜' },
    ]);
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
        organizationId: 'org-1',
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
          stage: ['正常运作'],
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
        organizationId: 'org-1',
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

  it('订单先返回时等待同组织箱型字典，冷启动不把 UUID 固化为列表文字', async () => {
    const masterData = Promise.withResolvers<API.MasterDataItem[]>();
    masterOptionsMock.mockReturnValue(masterData.promise);
    listOrdersMock.mockResolvedValue({
      data: [
        {
          id: 'order-cold-start',
          containerRequests: [
            {
              quantity: 1,
              containerSpecId: '31c5d297-1456-454b-8e26-e489a83afce9',
            },
            { quantity: 2, containerSpecId: 'container-spec-40hq' },
          ],
        },
      ],
      total: 1,
    });
    let resolved = false;
    const result = queryOrderList({}, seaExportDefinition, {
      organizationId: 'org-cold-start',
      ports: [],
      airports: [],
      customerMap: {},
    }).then((value) => {
      resolved = true;
      return value;
    });
    await vi.waitFor(() =>
      expect(masterOptionsMock).toHaveBeenCalledWith('org-cold-start'),
    );
    expect(resolved).toBe(false);
    masterData.resolve([
      {
        id: '31c5d297-1456-454b-8e26-e489a83afce9',
        kind: 7,
        code: '20GP',
        name: '20尺普柜',
      },
      {
        id: 'container-spec-40hq',
        kind: 7,
        code: '40HQ',
        name: '40尺高柜',
        enabled: false,
      },
    ]);
    expect((await result).data[0].containerSummary).toBe('1×20GP, 2×40HQ');
  });

  it('无代码时使用箱型名称，缺少箱型时保留数量并明确标记，不显示 ID', async () => {
    masterOptionsMock.mockResolvedValue([
      { id: 'named-spec', kind: 7, name: '特种箱' },
      { id: 'wrong-kind', kind: 9, code: '非箱型字典' },
    ]);
    listOrdersMock.mockResolvedValue({
      data: [
        {
          id: 'order-1',
          containerRequests: [
            { quantity: 1, containerSpecId: 'named-spec' },
            { quantity: 3, containerSpecId: 'missing-spec' },
            { quantity: 2, containerSpecId: 'wrong-kind' },
          ],
        },
      ],
      total: 1,
    });
    const result = await queryOrderList({}, seaExportDefinition, {
      organizationId: 'org-1',
      ports: [],
      airports: [],
      customerMap: {},
    });
    expect(result.data[0].containerSummary).toBe(
      '1×特种箱, 3×箱型缺失, 2×箱型缺失',
    );
  });

  it.each([seaExportDefinition, airExportDefinition])(
    '$kind 无箱计划时不为摘要加载字典',
    async (definition) => {
      listOrdersMock.mockResolvedValue({ data: [{ id: 'order-1' }], total: 1 });
      const result = await queryOrderList({}, definition, {
        organizationId: 'org-1',
        ports: [],
        airports: [],
        customerMap: {},
      });
      expect(result.data[0].containerSummary).toBeUndefined();
      expect(masterOptionsMock).not.toHaveBeenCalled();
    },
  );

  it('箱型字典加载失败时外传错误，不返回带原始 ID 的列表行', async () => {
    masterOptionsMock.mockRejectedValue(new Error('主数据选项加载失败'));
    listOrdersMock.mockResolvedValue({
      data: [
        {
          id: 'order-1',
          containerRequests: [{ quantity: 1, containerSpecId: 'missing-spec' }],
        },
      ],
      total: 1,
    });
    await expect(
      queryOrderList({}, seaExportDefinition, {
        organizationId: 'org-1',
        ports: [],
        airports: [],
        customerMap: {},
      }),
    ).rejects.toThrow('主数据选项加载失败');
  });

  it('人员三列映射服务端投影名称，未分配时为空由模板渲染占位', async () => {
    listOrdersMock.mockResolvedValue({
      data: [
        {
          id: 'order-1',
          orderNo: 'SE-001',
          flowStatus: 2,
          operatorName: '张强',
          operatorBranch: '上海分公司',
          salesName: '王丽',
          creatorName: '李明',
        },
        { id: 'order-2', orderNo: 'SE-002', flowStatus: 2 },
      ],
      total: 2,
      success: true,
    });

    const result = await queryOrderList(
      { page: 1, pageSize: 20 },
      seaExportDefinition,
      { ports: [], airports: [], customerMap: {}, organizationId: 'org-1' },
    );

    const [assigned, unassigned] = result.data;
    expect(assigned.operatorName).toBe('张强');
    expect(assigned.operatorBranch).toBe('上海分公司');
    expect(assigned.salesName).toBe('王丽');
    expect(assigned.creatorName).toBe('李明');
    expect(unassigned.operatorName).toBeUndefined();
    expect(unassigned.salesName).toBeUndefined();
    expect(unassigned.creatorName).toBeUndefined();
  });

  it('业务类型列展示注册定义的导航标题而非页面主标题', async () => {
    listOrdersMock.mockResolvedValue({ data: [], total: 0, success: true });

    const result = await queryOrderList(
      { page: 1, pageSize: 20 },
      seaExportDefinition,
      { ports: [], airports: [], customerMap: {}, organizationId: 'org-1' },
    );

    expect(result.data).toEqual([]);
    expect(seaExportDefinition.title).toBe('海运出口订单');
    expect(seaExportDefinition.navigationTitle).toBe('海运出口');
  });
});
