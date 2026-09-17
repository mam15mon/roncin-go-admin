import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import {
  OrderBusinessType,
  OrderPersonnelRole,
  ShipmentMode,
  ShipmentType,
  TradeDirection,
  TradeTerm,
} from '@/enums.generated';
import {
  buildSeaExportCreateDefaults,
  buildSeaExportCreatePayload,
  buildSeaExportDetailInitialValues,
  buildSeaExportUpdatePayload,
  seaExportFormAdapter,
} from './form-adapter';

const traditionalForwardingServiceOptions = [
  'BOOKING',
  'TRUCKING',
  'CUSTOMS_EXPORT',
  'STUFFING',
  'CUSTOMS_IMPORT',
  'OVERSEA_SEGMENT',
  'INSURANCE',
  'WAREHOUSING',
].map((code) => ({ label: code, value: `st-${code}`, code }));

describe('buildSeaExportCreateDefaults', () => {
  const creator = {
    userId: 'user-1',
    displayName: '张三',
  };

  it('按传统货代口径生成默认运输、贸易条款与推荐服务', () => {
    const defaults = buildSeaExportCreateDefaults({
      creator,
      serviceTypeOptions: traditionalForwardingServiceOptions,
      cargoCategoryOptions: [
        { label: '普通货物', value: 'cat-gen', code: 'GENERAL' },
      ],
    });

    expect(dayjs.isDayjs(defaults.orderDate)).toBe(true);
    expect(defaults.shipmentMode).toBe(
      ShipmentMode.SHIPMENT_MODE_TRADITIONAL_FORWARDING,
    );
    expect(defaults.shipmentType).toBe(ShipmentType.SHIPMENT_TYPE_FCL);
    expect(defaults.tradeTerm).toBe(TradeTerm.TRADE_TERM_CIF);
    expect(defaults.serviceTypeIds).toEqual(['st-BOOKING']);
  });

  it('优先按 GENERAL 业务码识别默认货类，再回退 label=普货，均无则 undefined', () => {
    const generalFirst = buildSeaExportCreateDefaults({
      creator,
      serviceTypeOptions: [],
      cargoCategoryOptions: [
        { label: '普货', value: 'cat-pu-wrong', code: 'OTHER' },
        { label: '集装箱普通货', value: 'cat-gen-correct', code: 'GENERAL' },
      ],
    });
    expect(generalFirst.cargoCategoryIds).toEqual(['cat-gen-correct']);

    const labelFallback = buildSeaExportCreateDefaults({
      creator,
      serviceTypeOptions: [],
      cargoCategoryOptions: [
        { label: '危险品', value: 'cat-dg', code: 'DG' },
        { label: '普货', value: 'cat-pu-fallback' },
      ],
    });
    expect(labelFallback.cargoCategoryIds).toEqual(['cat-pu-fallback']);

    const none = buildSeaExportCreateDefaults({
      creator,
      serviceTypeOptions: [],
      cargoCategoryOptions: [{ label: '危险品', value: 'cat-dg', code: 'DG' }],
    });
    expect(none.cargoCategoryIds).toBeUndefined();
  });

  it('回填当前创建人；缺少创建人时保持 undefined', () => {
    const withCreator = buildSeaExportCreateDefaults({
      creator,
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
    });
    expect(withCreator.creatorUserId).toBe('user-1');

    const withoutCreator = buildSeaExportCreateDefaults({
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
    });
    expect(withoutCreator.creatorUserId).toBeUndefined();
  });
});

describe('buildSeaExportCreatePayload', () => {
  it('规范化订单字段并装配完整岗位人员', () => {
    const result = buildSeaExportCreatePayload({
      customerId: 'customer-1',
      customerReferenceNo: '  CUST-001  ',
      bookingNo: '  BOOKING-001  ',
      internalReferenceNo: '  INTERNAL-001  ',
      tradeTerm: 3,
      paymentTerm: 1,
      cargoReadyAt: dayjs('2026-08-29T01:00:00.000Z'),
      orderDate: '2026-08-29T02:00:00.000Z',
      operatorUserId: 'operator-1',
      salesUserId: 'sales-1',
      customerServiceUserId: 'service-1',
      associateUserId: 'associate-1',
      documentUserId: 'document-1',
      commercialUserId: 'commercial-1',
      associate2UserId: 'associate-2',
      seaDocumentStructure: 3,
      seaHouseBill: { houseNo: '  HBL-001  ', issuerSource: 1 },
    });

    expect(result).toMatchObject({
      customerId: 'customer-1',
      customerReferenceNo: 'CUST-001',
      bookingNo: 'BOOKING-001',
      internalReferenceNo: 'INTERNAL-001',
      businessType: OrderBusinessType.BUSINESS_TYPE_SE,
      tradeDirection: TradeDirection.TRADE_DIRECTION_EXPORT,
      tradeTerm: 3,
      paymentTerm: 1,
      cargoReadyAt: '2026-08-29T01:00:00.000Z',
      orderDate: '2026-08-29T02:00:00.000Z',
      personnelAssignments: [
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR,
          userId: 'operator-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_SALES,
          userId: 'sales-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE,
          userId: 'service-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE,
          userId: 'associate-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_DOCUMENT,
          userId: 'document-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_COMMERCIAL,
          userId: 'commercial-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE2,
          userId: 'associate-2',
        },
      ],
      shippingDocuments: undefined,
      seaDocument: {
        documentStructure: 3,
        houseBill: { houseNo: '  HBL-001  ', issuerSource: 1 },
      },
    });
  });

  it('不静默丢弃用户填写不完整的海运分单', () => {
    const result = buildSeaExportCreatePayload({
      customerId: 'customer-1',
      tradeTerm: 3,
      paymentTerm: 1,
      seaHouseBill: { houseNo: '   ', issuerSource: 1 },
    });

    expect(result.seaDocument?.houseBill).toEqual({
      houseNo: '   ',
      issuerSource: 1,
    });
  });

  it('忽略空白可选字段并按人员装配岗位', () => {
    const result = buildSeaExportCreatePayload({
      customerId: 'customer-1',
      customerReferenceNo: '   ',
      tradeTerm: 3,
      paymentTerm: 1,
      operatorUserId: 'operator-1',
    });

    expect(result.customerReferenceNo).toBeUndefined();
    expect(result.personnelAssignments).toEqual([
      {
        role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR,
        userId: 'operator-1',
      },
    ]);
    expect(result.shippingDocuments).toBeUndefined();
  });

  it('即使填写了通用 shippingDocuments，SE 也不提交该字段', () => {
    const result = buildSeaExportCreatePayload({
      customerId: 'customer-1',
      tradeTerm: 3,
      paymentTerm: 1,
      shippingDocuments: [{ houseNo: 'AWB-001' }],
    });

    expect(result.shippingDocuments).toBeUndefined();
    expect(result.seaDocument).toBeDefined();
  });

  it('组装海运出口 MBL 主单与候选确认参数', () => {
    const result = buildSeaExportCreatePayload({
      customerId: 'customer-1',
      tradeTerm: 3,
      paymentTerm: 1,
      shippingLineId: 'carrier-1',
      seaMasterBillMasterNo: 'COSCO999901',
      seaMasterBillCandidateId: 'candidate-mbl-1',
      seaMasterBillExpectedCandidateVersion: 3,
      seaMasterBillCandidateTeId: 'te-1',
      seaMasterBillExpectedCandidateTeVersion: 5,
      seaMasterBillCorrectionReason: '更正主单号',
    });

    expect(result.seaMasterBill).toEqual({
      masterNo: 'COSCO999901',
      candidateId: 'candidate-mbl-1',
      expectedCandidateVersion: '3',
      candidateTeId: 'te-1',
      expectedCandidateTeVersion: '5',
      correctionReason: '更正主单号',
    });
  });

  it('MBL 只提交主单字段，船公司由订单 shippingLineId 统一表达', () => {
    const result = buildSeaExportCreatePayload({
      customerId: 'customer-1',
      tradeTerm: 3,
      paymentTerm: 1,
      shippingLineId: 'carrier-authoritative',
      seaMasterBill: {
        masterNo: 'COSCO999902',
      },
    });

    expect(result.seaMasterBill).toEqual({
      masterNo: 'COSCO999902',
    });
  });

  it('主单号为空或空白时绝不组装 seaMasterBill 对象，避免服务端报 missing required field', () => {
    const resultWithShippingLine = buildSeaExportCreatePayload({
      customerId: 'customer-1',
      tradeTerm: 3,
      paymentTerm: 1,
      shippingLineId: 'carrier-1',
      seaMasterBillMasterNo: '   ',
      seaMasterBill: {} as any,
    });

    expect(resultWithShippingLine.seaMasterBill).toBeUndefined();

    const resultCompletelyEmpty = buildSeaExportCreatePayload({
      customerId: 'customer-1',
      tradeTerm: 3,
      paymentTerm: 1,
      shippingLineId: 'carrier-1',
    });

    expect(resultCompletelyEmpty.seaMasterBill).toBeUndefined();
  });

  it('未提供 tradeTerm 时 payload 中 tradeTerm 为 undefined', () => {
    const result = buildSeaExportCreatePayload({
      customerId: 'customer-1',
      paymentTerm: 1,
    });

    expect(result.tradeTerm).toBeUndefined();
  });
});

describe('buildSeaExportDetailInitialValues', () => {
  it('按生成的人员角色枚举回填订单协作人员', () => {
    const result = buildSeaExportDetailInitialValues(
      { id: 'order-1' },
      [],
      [
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CREATOR,
          userId: 'creator',
          organizationId: 'org-creator',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR,
          userId: 'operator',
          organizationId: 'org-operator',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_COMMERCIAL,
          userId: 'commercial',
          organizationId: 'org-commercial',
        },
      ],
    );

    expect(result).toEqual(
      expect.objectContaining({
        creatorUserId: 'creator',
        operatorUserId: 'operator',
        commercialUserId: 'commercial',
      }),
    );
  });

  it('回填当前共享主单版本供单票身份更正做乐观锁校验', () => {
    const result = buildSeaExportDetailInitialValues(
      {
        id: 'order-1',
        shippingLineId: 'carrier-1',
        seaMasterBill: {
          masterBillId: 'mbl-1',
          masterNo: 'COSCO123456',
          shippingLineId: 'carrier-1',
          transportExecutionId: 'transport-1',
          vesselName: '',
          voyageNo: '',
          status: 'DRAFT',
          version: '7',
          memberCount: 1,
        },
      },
      [],
      [],
    );

    expect(result).toEqual(
      expect.objectContaining({
        seaMasterBillMasterNo: 'COSCO123456',
        shippingLineId: 'carrier-1',
        seaMasterBillExpectedCandidateVersion: '7',
      }),
    );
    expect(result).not.toHaveProperty('seaMasterBillIssuerPartnerId');
  });

  it('订单缺失时返回空对象，日期字段恢复为 Dayjs', () => {
    expect(buildSeaExportDetailInitialValues(undefined, [], [])).toEqual({});

    const result = buildSeaExportDetailInitialValues({
      id: 'order-2',
      etd: '2026-08-01T00:00:00.000Z',
      orderDate: '2026-08-01T00:00:00.000Z',
      createdAt: '2026-07-01T00:00:00.000Z',
    });
    expect(dayjs.isDayjs(result.etd)).toBe(true);
    expect(dayjs.isDayjs(result.orderDate)).toBe(true);
    expect(dayjs(result.orderDate).toISOString()).toBe(
      '2026-08-01T00:00:00.000Z',
    );
  });
});

describe('buildSeaExportUpdatePayload', () => {
  it('更新海运订单时由订单 shippingLineId 统一表达 MBL 船公司', () => {
    const result = buildSeaExportUpdatePayload('order-1', '7', {
      customerId: 'customer-1',
      tradeTerm: 3,
      paymentTerm: 1,
      shippingLineId: 'carrier-2',
      seaMasterBillMasterNo: 'COSCO123457',
      seaDocumentStructure: 1,
    });

    expect(result.id).toBe('order-1');
    expect(result.expectedVersion).toBe('7');
    expect(result.seaMasterBill).toEqual({
      masterNo: 'COSCO123457',
      candidateId: undefined,
      expectedCandidateVersion: undefined,
      correctionReason: undefined,
    });
  });

  it('不再通过字段存在性猜测运输方式：seaDocument 只来自显式表单值', () => {
    const result = buildSeaExportUpdatePayload('order-1', '0', {
      customerId: 'customer-1',
      paymentTerm: 1,
    });

    expect(result.seaDocument).toBeUndefined();
    expect(result.shippingDocuments).toBeUndefined();
    expect(result.seaMasterBill).toBeUndefined();
    expect(result.expectedVersion).toBe('0');
  });

  it('箱量请求过滤缺少箱型或数量的行并保留既有 ID', () => {
    const result = buildSeaExportUpdatePayload('order-1', '3', {
      customerId: 'customer-1',
      paymentTerm: 1,
      containerRequests: [
        { id: 'req-1', containerSpecId: 'spec-1', quantity: 2 },
        { containerSpecId: '', quantity: 1 },
        { containerSpecId: 'spec-2' },
      ],
    });

    expect(result.containerRequests).toEqual([
      { id: 'req-1', containerSpecId: 'spec-1', quantity: 2 },
    ]);
  });

  it('日期字段转换为 ISO 字符串，空值保持 undefined', () => {
    const result = buildSeaExportUpdatePayload('order-1', '1', {
      customerId: 'customer-1',
      paymentTerm: 1,
      etd: dayjs('2026-08-29T01:00:00.000Z'),
      eta: undefined,
    });

    expect(result.etd).toBe('2026-08-29T01:00:00.000Z');
    expect(result.eta).toBeUndefined();
  });
});

describe('SE 适配器完整固定夹具等价', () => {
  it('完整创建表单逐字段映射为完整创建请求', () => {
    const result = buildSeaExportCreatePayload({
      customerId: 'customer-1',
      customerReferenceNo: '  CUST-001  ',
      bookingNo: 'BOOKING-001',
      internalReferenceNo: 'INTERNAL-001',
      tradeTerm: 3,
      paymentTerm: 1,
      shippingLineId: 'carrier-1',
      bookingAgentId: 'agent-1',
      foreignAgentId: 'foreign-1',
      shippingAgentId: 'shipping-agent-1',
      contractNo: '  CONTRACT-001  ',
      cargoValue: ' 12000.50 ',
      cargoCurrency: 'USD',
      insurancePremium: ' 300 ',
      insuranceCurrency: 'CNY',
      unNumber: ' UN1263 ',
      hazardClass: ' 3 ',
      factoryName: ' 上海工厂 ',
      cargoReadyAt: '2026-08-20T01:00:00.000Z',
      declarationCutoffAt: dayjs('2026-08-21T02:00:00.000Z'),
      receivedAt: '2026-08-22T03:00:00.000Z',
      shipmentType: 1,
      containerOwnership: 1,
      shipmentMode: 1,
      serviceTypeIds: ['st-booking', 'st-trucking'],
      cargoCategoryIds: ['cat-gen'],
      originLocationId: 'loc-origin',
      destinationLocationId: 'loc-dest',
      dischargeLocationId: 'loc-discharge',
      transitLocationId: 'loc-transit',
      vesselVoyage: '  COSCO STAR / 024W  ',
      etd: dayjs('2026-09-01T00:00:00.000Z'),
      eta: '2026-09-15T00:00:00.000Z',
      siCutoff: dayjs('2026-08-28T04:00:00.000Z'),
      docCutoff: dayjs('2026-08-29T05:00:00.000Z'),
      customsCutoff: dayjs('2026-08-30T06:00:00.000Z'),
      vgmCutoff: dayjs('2026-08-31T07:00:00.000Z'),
      goodsDescription: ' 普通家具 ',
      totalPackages: 100,
      totalGrossWeightKg: 18500.5,
      totalVolumeCbm: 68.2,
      totalPackageUnit: ' CTNS ',
      specialRequirements: ' 易碎品 ',
      orderDate: dayjs('2026-08-19T08:00:00.000Z'),
      notes: ' 内部备注 ',
      bookingNotes: ' 订舱备注 ',
      allocationNotes: ' 配舱备注 ',
      operationNotes: ' 操作备注 ',
      shippingDocuments: [{ houseNo: 'SHOULD-NOT-SEND' }],
      containerRequests: [
        { id: 'req-1', containerSpecId: 'spec-1', quantity: 2 },
      ],
      seaMasterBillMasterNo: 'COSCO999901',
      seaMasterBillCandidateId: 'candidate-1',
      seaMasterBillExpectedCandidateVersion: 3,
      seaMasterBillCandidateTeId: 'te-1',
      seaMasterBillExpectedCandidateTeVersion: 5,
      seaMasterBillCorrectionReason: ' 更正主单号 ',
      operatorUserId: 'operator-1',
      salesUserId: 'sales-1',
      customerServiceUserId: 'cs-1',
      associateUserId: 'assoc-1',
      documentUserId: 'doc-1',
      commercialUserId: 'comm-1',
      associate2UserId: 'assoc2-1',
      creatorUserId: 'creator-1',
      seaDocumentStructure: 3,
      seaMasterBillContent: { remark: 'MBL 备注' } as API.SeaBillContent,
      seaHouseBill: {
        houseNo: ' HBL-001 ',
        issuerSource: 1,
        note: ' note ',
        content: { remark: 'HBL 备注' } as never,
      },
    });

    expect(result).toEqual({
      customerId: 'customer-1',
      customerReferenceNo: 'CUST-001',
      bookingNo: 'BOOKING-001',
      internalReferenceNo: 'INTERNAL-001',
      businessType: OrderBusinessType.BUSINESS_TYPE_SE,
      tradeDirection: TradeDirection.TRADE_DIRECTION_EXPORT,
      tradeTerm: 3,
      paymentTerm: 1,
      shippingLineId: 'carrier-1',
      bookingAgentId: 'agent-1',
      foreignAgentId: 'foreign-1',
      shippingAgentId: 'shipping-agent-1',
      contractNo: 'CONTRACT-001',
      cargoValue: '12000.50',
      cargoCurrency: 'USD',
      insurancePremium: '300',
      insuranceCurrency: 'CNY',
      unNumber: 'UN1263',
      hazardClass: '3',
      factoryName: '上海工厂',
      cargoReadyAt: '2026-08-20T01:00:00.000Z',
      declarationCutoffAt: '2026-08-21T02:00:00.000Z',
      receivedAt: '2026-08-22T03:00:00.000Z',
      shipmentType: 1,
      containerOwnership: 1,
      shipmentMode: 1,
      serviceTypeIds: ['st-booking', 'st-trucking'],
      cargoCategoryIds: ['cat-gen'],
      originLocationId: 'loc-origin',
      destinationLocationId: 'loc-dest',
      dischargeLocationId: 'loc-discharge',
      transitLocationId: 'loc-transit',
      vesselVoyage: 'COSCO STAR / 024W',
      etd: '2026-09-01T00:00:00.000Z',
      eta: '2026-09-15T00:00:00.000Z',
      siCutoff: '2026-08-28T04:00:00.000Z',
      docCutoff: '2026-08-29T05:00:00.000Z',
      customsCutoff: '2026-08-30T06:00:00.000Z',
      vgmCutoff: '2026-08-31T07:00:00.000Z',
      goodsDescription: '普通家具',
      totalPackages: 100,
      totalGrossWeightKg: 18500.5,
      totalVolumeCbm: 68.2,
      totalPackageUnit: 'CTNS',
      specialRequirements: '易碎品',
      orderDate: '2026-08-19T08:00:00.000Z',
      notes: '内部备注',
      bookingNotes: '订舱备注',
      allocationNotes: '配舱备注',
      operationNotes: '操作备注',
      personnelAssignments: [
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR,
          userId: 'operator-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_SALES,
          userId: 'sales-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CUSTOMER_SERVICE,
          userId: 'cs-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE,
          userId: 'assoc-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_DOCUMENT,
          userId: 'doc-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_COMMERCIAL,
          userId: 'comm-1',
        },
        {
          role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_ASSOCIATE2,
          userId: 'assoc2-1',
        },
      ],
      shippingDocuments: undefined,
      containerRequests: [
        { id: 'req-1', containerSpecId: 'spec-1', quantity: 2 },
      ],
      seaMasterBill: {
        masterNo: 'COSCO999901',
        candidateId: 'candidate-1',
        expectedCandidateVersion: '3',
        candidateTeId: 'te-1',
        expectedCandidateTeVersion: '5',
        correctionReason: '更正主单号',
      },
      seaDocument: {
        documentStructure: 3,
        masterBillContent: { remark: 'MBL 备注' },
        houseBill: {
          houseNo: ' HBL-001 ',
          id: undefined,
          issuerPartnerId: undefined,
          issuerSource: 1,
          note: 'note',
          content: { remark: 'HBL 备注' },
        },
      },
    });
  });

  it('完整订单聚合与人员逐字段映射为完整详情初始值', () => {
    const result = buildSeaExportDetailInitialValues(
      {
        id: 'order-1',
        orderNo: 'SE-001',
        customerId: 'customer-1',
        customerReferenceNo: 'CUST-001',
        bookingNo: 'BOOKING-001',
        internalReferenceNo: 'INTERNAL-001',
        tradeTerm: 3,
        paymentTerm: 1,
        shippingLineId: 'carrier-1',
        bookingAgentId: 'agent-1',
        foreignAgentId: 'foreign-1',
        shippingAgentId: 'shipping-agent-1',
        contractNo: 'CONTRACT-001',
        cargoValue: '12000.50',
        cargoCurrency: 'USD',
        insurancePremium: '300',
        insuranceCurrency: 'CNY',
        unNumber: 'UN1263',
        hazardClass: '3',
        factoryName: '上海工厂',
        cargoReadyAt: '2026-08-20T01:00:00.000Z',
        declarationCutoffAt: '2026-08-21T02:00:00.000Z',
        receivedAt: '2026-08-22T03:00:00.000Z',
        shipmentType: 1,
        containerOwnership: 1,
        shipmentMode: 1,
        serviceTypeIds: ['st-booking'],
        cargoCategoryIds: ['cat-gen'],
        originLocationId: 'loc-origin',
        destinationLocationId: 'loc-dest',
        dischargeLocationId: 'loc-discharge',
        transitLocationId: 'loc-transit',
        vesselVoyage: 'COSCO STAR / 024W',
        etd: '2026-09-01T00:00:00.000Z',
        eta: '2026-09-15T00:00:00.000Z',
        siCutoff: '2026-08-28T04:00:00.000Z',
        docCutoff: '2026-08-29T05:00:00.000Z',
        customsCutoff: '2026-08-30T06:00:00.000Z',
        vgmCutoff: '2026-08-31T07:00:00.000Z',
        goodsDescription: '普通家具',
        specialRequirements: '易碎品',
        totalPackages: 100,
        totalGrossWeightKg: 18500.5,
        totalVolumeCbm: 68.2,
        totalPackageUnit: 'CTNS',
        orderDate: '2026-08-19T08:00:00.000Z',
        createdAt: '2026-08-18T00:00:00.000Z',
        notes: '内部备注',
        bookingNotes: '订舱备注',
        allocationNotes: '配舱备注',
        operationNotes: '操作备注',
        containerRequests: [
          { id: 'req-1', containerSpecId: 'spec-1', quantity: 2 },
        ],
        seaMasterBill: {
          masterNo: 'COSCO999901',
          version: '7',
          shippingLineName: '中远海运',
        } as API.SeaMasterBillSummary,
        seaDocumentStructure: 3,
        seaDocumentLinkVersion: '11',
        seaDocumentSummary: { houseCount: 1 } as API.SeaOrderDocumentSummary,
      } as API.Order,
      [{ id: 'doc-1', houseNo: 'HBL-001' } as API.OrderShippingDocument],
      [
        { role: 1, userId: 'creator-1', organizationId: 'org-8' },
        { role: 2, userId: 'operator-1', organizationId: 'org-1' },
        { role: 3, userId: 'sales-1', organizationId: 'org-2' },
        { role: 4, userId: 'cs-1', organizationId: 'org-3' },
        { role: 5, userId: 'doc-1', organizationId: 'org-5' },
        { role: 6, userId: 'comm-1', organizationId: 'org-6' },
        { role: 7, userId: 'assoc-1', organizationId: 'org-4' },
        { role: 8, userId: 'assoc2-1', organizationId: 'org-7' },
      ] as API.OrderPersonnel[],
    );

    expect(result).toEqual({
      orderNo: 'SE-001',
      customerId: 'customer-1',
      customerReferenceNo: 'CUST-001',
      bookingNo: 'BOOKING-001',
      internalReferenceNo: 'INTERNAL-001',
      tradeTerm: 3,
      paymentTerm: 1,
      shippingLineId: 'carrier-1',
      bookingAgentId: 'agent-1',
      foreignAgentId: 'foreign-1',
      shippingAgentId: 'shipping-agent-1',
      contractNo: 'CONTRACT-001',
      cargoValue: '12000.50',
      cargoCurrency: 'USD',
      insurancePremium: '300',
      insuranceCurrency: 'CNY',
      unNumber: 'UN1263',
      hazardClass: '3',
      factoryName: '上海工厂',
      cargoReadyAt: dayjs('2026-08-20T01:00:00.000Z'),
      declarationCutoffAt: dayjs('2026-08-21T02:00:00.000Z'),
      receivedAt: dayjs('2026-08-22T03:00:00.000Z'),
      shipmentType: 1,
      containerOwnership: 1,
      shipmentMode: 1,
      serviceTypeIds: ['st-booking'],
      cargoCategoryIds: ['cat-gen'],
      originLocationId: 'loc-origin',
      destinationLocationId: 'loc-dest',
      dischargeLocationId: 'loc-discharge',
      transitLocationId: 'loc-transit',
      vesselVoyage: 'COSCO STAR / 024W',
      etd: dayjs('2026-09-01T00:00:00.000Z'),
      eta: dayjs('2026-09-15T00:00:00.000Z'),
      siCutoff: dayjs('2026-08-28T04:00:00.000Z'),
      docCutoff: dayjs('2026-08-29T05:00:00.000Z'),
      customsCutoff: dayjs('2026-08-30T06:00:00.000Z'),
      vgmCutoff: dayjs('2026-08-31T07:00:00.000Z'),
      goodsDescription: '普通家具',
      specialRequirements: '易碎品',
      totalPackages: 100,
      totalGrossWeightKg: 18500.5,
      totalVolumeCbm: 68.2,
      totalPackageUnit: 'CTNS',
      orderDate: dayjs('2026-08-19T08:00:00.000Z'),
      notes: '内部备注',
      bookingNotes: '订舱备注',
      allocationNotes: '配舱备注',
      operationNotes: '操作备注',
      shippingDocuments: [{ id: 'doc-1', houseNo: 'HBL-001' }],
      containerRequests: [
        { id: 'req-1', containerSpecId: 'spec-1', quantity: 2 },
      ],
      creatorUserId: 'creator-1',
      operatorUserId: 'operator-1',
      salesUserId: 'sales-1',
      customerServiceUserId: 'cs-1',
      documentUserId: 'doc-1',
      commercialUserId: 'comm-1',
      associateUserId: 'assoc-1',
      associate2UserId: 'assoc2-1',
      seaMasterBillMasterNo: 'COSCO999901',
      seaMasterBillCandidateId: undefined,
      seaMasterBillExpectedCandidateVersion: '7',
      seaMasterBillCorrectionReason: undefined,
      seaMasterBill: {
        masterNo: 'COSCO999901',
        version: '7',
        shippingLineName: '中远海运',
      },
      seaDocumentStructure: 3,
      seaDocumentLinkVersion: '11',
      seaDocumentSummary: { houseCount: 1 },
    });
  });

  it('更新请求对空 receivedAt 显式输出 undefined，不误传默认日期', () => {
    const result = buildSeaExportUpdatePayload('order-1', '1', {
      customerId: 'customer-1',
      paymentTerm: 1,
      cargoReadyAt: dayjs('2026-08-20T01:00:00.000Z'),
      receivedAt: undefined,
    });

    expect(result).toHaveProperty('receivedAt', undefined);
    expect(result.cargoReadyAt).toBe('2026-08-20T01:00:00.000Z');
  });

  it('完整详情表单逐字段映射为完整更新请求', () => {
    const result = buildSeaExportUpdatePayload('order-1', '9', {
      customerId: 'customer-1',
      customerReferenceNo: '  CUST-001  ',
      bookingNo: 'BOOKING-001',
      internalReferenceNo: 'INTERNAL-001',
      tradeTerm: 3,
      paymentTerm: 1,
      shippingLineId: 'carrier-1',
      bookingAgentId: 'agent-1',
      foreignAgentId: 'foreign-1',
      shippingAgentId: 'shipping-agent-1',
      contractNo: '  CONTRACT-001  ',
      cargoValue: ' 12000.50 ',
      cargoCurrency: 'USD',
      insurancePremium: ' 300 ',
      insuranceCurrency: 'CNY',
      unNumber: ' UN1263 ',
      hazardClass: ' 3 ',
      factoryName: ' 上海工厂 ',
      cargoReadyAt: dayjs('2026-08-20T01:00:00.000Z'),
      declarationCutoffAt: '2026-08-21T02:00:00.000Z',
      receivedAt: dayjs('2026-08-22T03:00:00.000Z'),
      shipmentType: 1,
      containerOwnership: 1,
      shipmentMode: 1,
      serviceTypeIds: ['st-booking', 'st-trucking'],
      cargoCategoryIds: ['cat-gen'],
      originLocationId: 'loc-origin',
      destinationLocationId: 'loc-dest',
      dischargeLocationId: 'loc-discharge',
      transitLocationId: 'loc-transit',
      vesselVoyage: '  COSCO STAR / 024W  ',
      etd: dayjs('2026-09-01T00:00:00.000Z'),
      eta: '2026-09-15T00:00:00.000Z',
      siCutoff: dayjs('2026-08-28T04:00:00.000Z'),
      docCutoff: dayjs('2026-08-29T05:00:00.000Z'),
      customsCutoff: dayjs('2026-08-30T06:00:00.000Z'),
      vgmCutoff: dayjs('2026-08-31T07:00:00.000Z'),
      goodsDescription: ' 普通家具 ',
      specialRequirements: ' 易碎品 ',
      totalPackages: 100,
      totalGrossWeightKg: 18500.5,
      totalVolumeCbm: 68.2,
      totalPackageUnit: ' CTNS ',
      notes: ' 内部备注 ',
      bookingNotes: ' 订舱备注 ',
      allocationNotes: ' 配舱备注 ',
      operationNotes: ' 操作备注 ',
      containerRequests: [
        { id: 'req-1', containerSpecId: 'spec-1', quantity: 2 },
        { containerSpecId: '', quantity: 1 },
      ],
      seaMasterBillMasterNo: 'COSCO999902',
      seaMasterBillCandidateId: 'candidate-2',
      seaMasterBillExpectedCandidateVersion: 4,
      seaMasterBillCorrectionReason: ' 更正主单号 ',
      seaDocument: {
        documentStructure: 3,
        houseBill: { houseNo: 'HBL-002' },
      } as API.SeaOrderDocumentInput,
    });

    expect(result).toEqual({
      id: 'order-1',
      expectedVersion: '9',
      customerId: 'customer-1',
      customerReferenceNo: 'CUST-001',
      bookingNo: 'BOOKING-001',
      internalReferenceNo: 'INTERNAL-001',
      tradeTerm: 3,
      paymentTerm: 1,
      shippingLineId: 'carrier-1',
      bookingAgentId: 'agent-1',
      foreignAgentId: 'foreign-1',
      shippingAgentId: 'shipping-agent-1',
      contractNo: 'CONTRACT-001',
      cargoValue: '12000.50',
      cargoCurrency: 'USD',
      insurancePremium: '300',
      insuranceCurrency: 'CNY',
      unNumber: 'UN1263',
      hazardClass: '3',
      factoryName: '上海工厂',
      cargoReadyAt: '2026-08-20T01:00:00.000Z',
      declarationCutoffAt: '2026-08-21T02:00:00.000Z',
      receivedAt: '2026-08-22T03:00:00.000Z',
      shipmentType: 1,
      containerOwnership: 1,
      shipmentMode: 1,
      serviceTypeIds: ['st-booking', 'st-trucking'],
      cargoCategoryIds: ['cat-gen'],
      originLocationId: 'loc-origin',
      destinationLocationId: 'loc-dest',
      dischargeLocationId: 'loc-discharge',
      transitLocationId: 'loc-transit',
      vesselVoyage: 'COSCO STAR / 024W',
      etd: '2026-09-01T00:00:00.000Z',
      eta: '2026-09-15T00:00:00.000Z',
      siCutoff: '2026-08-28T04:00:00.000Z',
      docCutoff: '2026-08-29T05:00:00.000Z',
      customsCutoff: '2026-08-30T06:00:00.000Z',
      vgmCutoff: '2026-08-31T07:00:00.000Z',
      goodsDescription: '普通家具',
      specialRequirements: '易碎品',
      totalPackages: 100,
      totalGrossWeightKg: 18500.5,
      totalVolumeCbm: 68.2,
      totalPackageUnit: 'CTNS',
      notes: '内部备注',
      bookingNotes: '订舱备注',
      allocationNotes: '配舱备注',
      operationNotes: '操作备注',
      shippingDocuments: undefined,
      containerRequests: [
        { id: 'req-1', containerSpecId: 'spec-1', quantity: 2 },
      ],
      seaMasterBill: {
        masterNo: 'COSCO999902',
        candidateId: 'candidate-2',
        expectedCandidateVersion: '4',
        correctionReason: '更正主单号',
      },
      seaDocument: {
        documentStructure: 3,
        houseBill: { houseNo: 'HBL-002' },
      },
    });
  });
});

describe('seaExportFormAdapter', () => {
  it('适配器五个入口均绑定 SE 专属实现', () => {
    expect(seaExportFormAdapter.buildSections).toBeDefined();
    expect(seaExportFormAdapter.buildCreateDefaults).toBe(
      buildSeaExportCreateDefaults,
    );
    expect(seaExportFormAdapter.buildCreatePayload).toBe(
      buildSeaExportCreatePayload,
    );
    expect(seaExportFormAdapter.buildDetailInitialValues).toBe(
      buildSeaExportDetailInitialValues,
    );
    expect(seaExportFormAdapter.buildUpdatePayload).toBe(
      buildSeaExportUpdatePayload,
    );
  });
});
