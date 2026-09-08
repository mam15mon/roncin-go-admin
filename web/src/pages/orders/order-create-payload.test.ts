import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import { ORDER_KIND_CONFIGS } from './common';
import { buildCreateOrderPayload } from './order-create-payload';

describe('buildCreateOrderPayload', () => {
  it('规范化订单字段并装配完整岗位人员', () => {
    const result = buildCreateOrderPayload(
      {
        customerId: 'customer-1',
        customerReferenceNo: '  CUST-001  ',
        bookingNo: '  BOOKING-001  ',
        internalReferenceNo: '  INTERNAL-001  ',
        tradeTerm: 3,
        paymentTerm: 1,
        cargoReadyAt: dayjs('2026-08-29T01:00:00.000Z'),
        orderDate: '2026-08-29T02:00:00.000Z',
        operatorUserId: 'operator-1',
        operatorOrganizationId: 'org-1',
        salesUserId: 'sales-1',
        salesOrganizationId: 'org-2',
        customerServiceUserId: 'service-1',
        customerServiceOrganizationId: 'org-3',
        associateUserId: 'associate-1',
        associateOrganizationId: 'org-4',
        documentUserId: 'document-1',
        documentOrganizationId: 'org-5',
        commercialUserId: 'commercial-1',
        commercialOrganizationId: 'org-6',
        associate2UserId: 'associate-2',
        associate2OrganizationId: 'org-7',
        seaDocumentStructure: 3,
        seaHouseBill: { houseNo: '  HBL-001  ', issuerSource: 1 },
      },
      ORDER_KIND_CONFIGS['sea-export'],
    );

    expect(result).toMatchObject({
      customerId: 'customer-1',
      customerReferenceNo: 'CUST-001',
      bookingNo: 'BOOKING-001',
      internalReferenceNo: 'INTERNAL-001',
      businessType: 1,
      tradeDirection: 1,
      tradeTerm: 3,
      paymentTerm: 1,
      cargoReadyAt: '2026-08-29T01:00:00.000Z',
      orderDate: '2026-08-29T02:00:00.000Z',
      personnelAssignments: [
        { role: 2, userId: 'operator-1', organizationId: 'org-1' },
        { role: 3, userId: 'sales-1', organizationId: 'org-2' },
        { role: 4, userId: 'service-1', organizationId: 'org-3' },
        { role: 7, userId: 'associate-1', organizationId: 'org-4' },
        { role: 5, userId: 'document-1', organizationId: 'org-5' },
        { role: 6, userId: 'commercial-1', organizationId: 'org-6' },
        { role: 8, userId: 'associate-2', organizationId: 'org-7' },
      ],
      shippingDocuments: undefined,
      seaDocument: {
        documentStructure: 3,
        houseBill: { houseNo: '  HBL-001  ', issuerSource: 1 },
      },
    });
  });

  it('不静默丢弃用户填写不完整的海运分单', () => {
    const result = buildCreateOrderPayload(
      {
        customerId: 'customer-1',
        tradeTerm: 3,
        paymentTerm: 1,
        seaHouseBill: { houseNo: '   ', issuerSource: 1 },
      },
      ORDER_KIND_CONFIGS['sea-export'],
    );

    expect(result.seaDocument?.houseBill).toEqual({
      houseNo: '   ',
      issuerSource: 1,
    });
  });

  it('忽略空白可选字段和不完整的岗位人员', () => {
    const result = buildCreateOrderPayload(
      {
        customerId: 'customer-1',
        customerReferenceNo: '   ',
        tradeTerm: 3,
        paymentTerm: 1,
        operatorUserId: 'operator-1',
      },
      ORDER_KIND_CONFIGS['sea-export'],
    );

    expect(result.customerReferenceNo).toBeUndefined();
    expect(result.personnelAssignments).toEqual([]);
    expect(result.shippingDocuments).toBeUndefined();
  });

  it('非海运订单装配旧提单 shippingDocuments', () => {
    const nonSeaConfig = {
      kind: 'air-export' as any,
      businessType: 2,
      tradeDirection: 1,
      title: '空运出口订单',
      navigationTitle: '空运出口',
      category: 'air' as const,
    };
    const result = buildCreateOrderPayload(
      {
        customerId: 'customer-1',
        tradeTerm: 3,
        paymentTerm: 1,
        shippingDocuments: [{ houseNo: '  AWB-001  ' }, { houseNo: '  ' }],
      },
      nonSeaConfig,
    );

    expect(result.shippingDocuments).toEqual([{ houseNo: 'AWB-001' }]);
    expect(result.seaDocument).toBeUndefined();
  });

  it('组装海运出口 MBL 主单与候选确认参数', () => {
    const result = buildCreateOrderPayload(
      {
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
      },
      ORDER_KIND_CONFIGS['sea-export'],
    );

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
    const result = buildCreateOrderPayload(
      {
        customerId: 'customer-1',
        tradeTerm: 3,
        paymentTerm: 1,
        shippingLineId: 'carrier-authoritative',
        seaMasterBill: {
          masterNo: 'COSCO999902',
        },
      },
      ORDER_KIND_CONFIGS['sea-export'],
    );

    expect(result.seaMasterBill).toEqual({
      masterNo: 'COSCO999902',
    });
  });

  it('未提供 tradeTerm 时 payload 中 tradeTerm 为 undefined', () => {
    const result = buildCreateOrderPayload(
      {
        customerId: 'customer-1',
        paymentTerm: 1,
      },
      ORDER_KIND_CONFIGS['sea-export'],
    );

    expect(result.tradeTerm).toBeUndefined();
  });
});
