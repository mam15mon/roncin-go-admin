import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  OrderBusinessType,
  OrderClosureStatus,
  OrderFlowStatus,
  OrderTerminationStatus,
  SeaDocumentStructure,
} from '@/enums.generated';
import { queryOrderList } from './list-query';
import { airExportDefinition } from './order-kinds/air-export/definition';
import { airImportDefinition } from './order-kinds/air-import/definition';
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

  it.each([undefined, false])(
    '成功标志%s不是true时拒绝夹带旧data的列表响应',
    async (success) => {
      listOrdersMock.mockResolvedValue({
        success,
        data: [{ id: 'old-order', orderNo: '旧资料' }],
        total: 1,
      });
      await expect(
        queryOrderList({}, seaExportDefinition, {
          organizationId: 'org-1',
          ports: [],
          airports: [],
          customerMap: {},
        }),
      ).rejects.toThrow('订单列表加载失败');
    },
  );

  it('AI列表查询携带真实类型并保留权威ETA/航空号码投影', async () => {
    listOrdersMock.mockResolvedValue({
      success: true,
      total: 1,
      data: [
        {
          id: 'order-ai',
          businessType: OrderBusinessType.BUSINESS_TYPE_AI,
          organizationId: 'org-1',
          orderNo: 'AI001',
          eta: '2026-11-16T16:30:00Z',
          airExport: {
            master: {
              id: 'master-ai',
              businessType: 'AI',
              organizationId: 'org-1',
              version: '2',
              content: { masterNo: '78112345675' },
              originAirportName: '洛杉矶 (LAX)',
              destinationAirportName: '上海浦东 (PVG)',
            },
          },
        },
      ],
    });
    const result = await queryOrderList(
      { page: 1, pageSize: 20 },
      airImportDefinition,
      { ports: [], airports: [], customerMap: {}, organizationId: 'org-1' },
    );
    expect(listOrdersMock).toHaveBeenCalledWith(
      expect.objectContaining({
        businessType: OrderBusinessType.BUSINESS_TYPE_AI,
      }),
    );
    expect(result.data[0]).toMatchObject({
      orderKind: 'air-import',
      masterBlNo: '78112345675',
      eta: '2026-11-16T16:30:00Z',
    });
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
      }),
    );
    expect(listOrdersMock.mock.calls[0][0]).not.toHaveProperty('closureStatus');
    expect(listOrdersMock.mock.calls[0][0]).not.toHaveProperty('tagIds');
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

  it('使用港口、机场、客户和服务端箱型投影构造列表行', async () => {
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
            {
              quantity: 2,
              containerSpecId: 'container-spec-1',
              containerSpecCode: '40HQ',
              containerSpecName: '40尺高柜',
            },
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

  it('列表箱计划直接使用同次响应投影，不等待或读取另一次字典', async () => {
    masterOptionsMock.mockRejectedValue(new Error('不得另外加载变化中的字典'));
    listOrdersMock.mockResolvedValue({
      success: true,
      data: [
        {
          id: 'order-cold-start',
          containerRequests: [
            {
              quantity: 1,
              containerSpecId: 'spec-20gp',
              containerSpecCode: ' 20GP ',
              containerSpecName: '20尺普柜',
            },
            {
              quantity: 2,
              containerSpecId: 'spec-40hq',
              containerSpecCode: '40HQ',
              containerSpecName: '停用历史箱型',
            },
          ],
        },
      ],
      total: 1,
    });
    const result = await queryOrderList({}, seaExportDefinition, {
      organizationId: 'org-cold-start',
      ports: [],
      airports: [],
      customerMap: {},
    });
    expect(result.data[0].containerSummary).toBe('1×20GP, 2×40HQ');
    expect(masterOptionsMock).not.toHaveBeenCalled();
  });

  it('箱型代码未设置时使用同次响应中的真实名称', async () => {
    listOrdersMock.mockResolvedValue({
      success: true,
      data: [
        {
          id: 'order-1',
          containerRequests: [
            {
              quantity: 1,
              containerSpecId: 'named-spec',
              containerSpecName: ' 特种箱 ',
            },
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
    expect(result.data[0].containerSummary).toBe('1×特种箱');
    expect(masterOptionsMock).not.toHaveBeenCalled();
  });

  it.each([
    { quantity: 3, containerSpecId: 'missing-projection' },
    {
      quantity: 2,
      containerSpecId: 'blank-projection',
      containerSpecCode: ' ',
      containerSpecName: ' ',
    },
  ])('箱型投影缺失或空白时明确失败，不猜名称或另查字典', async (request) => {
    listOrdersMock.mockResolvedValue({
      success: true,
      data: [{ id: 'order-1', containerRequests: [request] }],
      total: 1,
    });
    await expect(
      queryOrderList({}, seaExportDefinition, {
        organizationId: 'org-1',
        ports: [],
        airports: [],
        customerMap: {},
      }),
    ).rejects.toThrow('订单箱计划缺少有效箱型投影');
    expect(masterOptionsMock).not.toHaveBeenCalled();
  });

  it.each([seaExportDefinition, airExportDefinition])(
    '$kind 无箱计划时不为摘要加载字典',
    async (definition) => {
      listOrdersMock.mockResolvedValue({
        success: true,
        data: [{ id: 'order-1' }],
        total: 1,
      });
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

  it('箱计划真实引用ID缺失时拒绝投影，不把展示名称用于身份判断', async () => {
    listOrdersMock.mockResolvedValue({
      success: true,
      data: [
        {
          id: 'order-1',
          containerRequests: [{ quantity: 1, containerSpecCode: '20GP' }],
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
    ).rejects.toThrow('订单箱计划缺少有效箱型投影');
    expect(masterOptionsMock).not.toHaveBeenCalled();
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
  it.each([
    [undefined, undefined, 'unknown', 0],
    [SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_UNSPECIFIED, 0, 'unknown', 0],
    [SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE, 2, 'house', 2],
    [SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT, 1, 'direct', 1],
  ] as const)(
    '海运单证映射遵循真实 wire 结构 %s / 票数 %s',
    async (structure, count, kind, memberCount) => {
      listOrdersMock.mockResolvedValue({
        success: true,
        data: [
          {
            id: 'order-1',
            orderNo: 'SE-001',
            seaMasterBill: {
              masterNo: 'MBL-001',
              ...(count === undefined ? {} : { memberCount: count }),
            },
            seaDocumentSummary:
              structure === undefined
                ? undefined
                : { documentStructure: structure, houseNo: 'HBL-001' },
            shippingDocuments: [{ houseNo: '旧单证不得使用' }],
          },
        ],
        total: 1,
      });
      const result = await queryOrderList({}, seaExportDefinition, {
        ports: [],
        airports: [],
        customerMap: {},
        organizationId: 'org-1',
      });
      expect(result.data[0].masterBlNo).toBe('MBL-001');
      expect(result.data[0].billRelation).toEqual({
        kind,
        memberCount,
        houseNo: structure === undefined ? undefined : 'HBL-001',
      });
    },
  );

  it('HOUSE 没有 HBL 号码时保留缺失，空运不启用海运关系模型', async () => {
    listOrdersMock.mockResolvedValue({
      success: true,
      data: [
        {
          id: 'order-1',
          seaDocumentSummary: {
            documentStructure:
              SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
          },
        },
      ],
      total: 1,
    });
    const ctx = {
      ports: [],
      airports: [],
      customerMap: {},
      organizationId: 'org-1',
    };
    const sea = await queryOrderList({}, seaExportDefinition, ctx);
    expect(sea.data[0].billRelation).toEqual({
      kind: 'house',
      memberCount: 0,
      houseNo: undefined,
    });
    const air = await queryOrderList({}, airExportDefinition, ctx);
    expect(air.data[0].billRelation).toBeUndefined();
  });
  it.each(['orderNo', 'createdAt', 'etd', 'eta'])(
    '向服务端传递 %s 排序，清除排序恢复默认',
    async (sorterField) => {
      listOrdersMock.mockResolvedValue({ success: true, data: [], total: 0 });
      const context = {
        ports: [],
        airports: [],
        customerMap: {},
        organizationId: 'org-1',
      };
      await queryOrderList(
        { sorterField, sorterOrder: 'ascend' },
        seaExportDefinition,
        context,
      );
      expect(listOrdersMock).toHaveBeenLastCalledWith(
        expect.objectContaining({ sortBy: sorterField, sortOrder: 'ascend' }),
      );
      await queryOrderList(
        { sorterField, sorterOrder: 'descend' },
        seaExportDefinition,
        context,
      );
      expect(listOrdersMock).toHaveBeenLastCalledWith(
        expect.objectContaining({ sortBy: sorterField, sortOrder: 'descend' }),
      );
      await queryOrderList({ sorterField }, seaExportDefinition, context);
      expect(listOrdersMock).toHaveBeenLastCalledWith(
        expect.objectContaining({ sortBy: undefined, sortOrder: undefined }),
      );
    },
  );

  it('映射真实列表投影，分别保留本票目的港、共同卸货港及业务备注', async () => {
    listOrdersMock.mockResolvedValue({
      success: true,
      data: [
        {
          id: 'order-1',
          orderNo: 'SE-001',
          destinationLocationName: '鹿特丹港',
          dischargeLocationName: '汉堡港',
          transitLocationName: '新加坡港',
          shippingLineName: '示例船公司',
          etd: '2026-11-11T03:00:00Z',
          eta: '2026-11-25T04:00:00Z',
          siCutoff: '2026-11-10T03:00:00Z',
          docCutoff: '2026-11-10T04:00:00Z',
          customsCutoff: '2026-11-10T05:00:00Z',
          vgmCutoff: '2026-11-10T06:00:00Z',
          declarationCutoffAt: '2026-11-10T07:00:00Z',
          bookingNo: 'SO-001',
          foreignAgentName: '海外代理',
          shippingAgentName: '船代',
          internalReferenceNo: 'IN-001',
          orderDate: '2026-11-11',
          receivedAt: '2026-11-01T03:00:00Z',
          cargoReadyAt: '2026-11-05T03:00:00Z',
          factoryName: '工厂甲',
          goodsDescription: '机器',
          cargoValue: '15362.50000000',
          cargoCurrency: 'USD',
          unNumber: '1203',
          hazardClass: '3',
          allocationNotes: '配舱备注甲',
          bookingNotes: '订舱备注乙',
          operationNotes: '操作备注丙',
          containerNos: ['CSQU3054383', 'MSCU1234567'],
          attachmentCount: 2,
          customerServiceName: '客服李',
        },
      ],
      total: 1,
    });
    const result = await queryOrderList({}, seaExportDefinition, {
      ports: [],
      airports: [],
      customerMap: {},
      organizationId: 'org-1',
    });
    expect(result.data[0]).toEqual(
      expect.objectContaining({
        destinationPortName: '鹿特丹港',
        dischargePortName: '汉堡港',
        transitPortName: '新加坡港',
        shippingLineName: '示例船公司',
        etd: '2026-11-11T03:00:00Z',
        eta: '2026-11-25T04:00:00Z',
        siCutoff: '2026-11-10T03:00:00Z',
        docCutoff: '2026-11-10T04:00:00Z',
        customsCutoff: '2026-11-10T05:00:00Z',
        vgmCutoff: '2026-11-10T06:00:00Z',
        declarationCutoffAt: '2026-11-10T07:00:00Z',
        bookingNo: 'SO-001',
        foreignAgentName: '海外代理',
        shippingAgentName: '船代',
        internalReferenceNo: 'IN-001',
        orderDate: '2026-11-11',
        receivedAt: '2026-11-01T03:00:00Z',
        cargoReadyAt: '2026-11-05T03:00:00Z',
        factoryName: '工厂甲',
        goodsDescription: '机器',
        cargoValue: '15362.50000000',
        cargoCurrency: 'USD',
        unNumber: '1203',
        hazardClass: '3',
        spaceNotes: '配舱备注甲',
        bookingNotes: '订舱备注乙',
        operationNotes: '操作备注丙',
        containerNos: ['CSQU3054383', 'MSCU1234567'],
        attachmentCount: 2,
        customerServiceName: '客服李',
      }),
    );
    expect(result.data[0]).not.toHaveProperty('finalDestination');
    expect(result.data[0]).not.toHaveProperty('commissionSummary');
  });

  it('可选附件数与货值保持存在性，缺省数组为空而显式零不丢失', async () => {
    listOrdersMock.mockResolvedValue({
      success: true,
      data: [
        { id: 'missing', orderNo: 'SE-MISSING' },
        {
          id: 'zero',
          orderNo: 'SE-ZERO',
          attachmentCount: 0,
          cargoValue: '0',
          totalPackages: 0,
          totalGrossWeightKg: 0,
          totalVolumeCbm: 0,
        },
      ],
      total: 2,
    });
    const result = await queryOrderList({}, seaExportDefinition, {
      ports: [],
      airports: [],
      customerMap: {},
      organizationId: 'org-1',
    });
    expect(result.data[0]).toEqual(
      expect.objectContaining({
        containerNos: [],
        attachmentCount: undefined,
        cargoValue: undefined,
        customerServiceName: undefined,
      }),
    );
    expect(result.data[1]).toEqual(
      expect.objectContaining({
        containerNos: [],
        attachmentCount: 0,
        cargoValue: '0',
        totalPackages: 0,
        grossWeightKg: 0,
        volumeCbm: 0,
      }),
    );
  });
});
