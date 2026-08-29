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
        shippingDocuments: [
          { masterNo: '  MBL-001  ', houseNo: '  ' },
          { masterNo: '  ', houseNo: '  HBL-001  ' },
          { masterNo: '  ', houseNo: '' },
        ],
      },
      ORDER_KIND_CONFIGS['sea-export'],
    );

    expect(result).toMatchObject({
      customerId: 'customer-1',
      customerReferenceNo: 'CUST-001',
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
      shippingDocuments: [
        { masterNo: 'MBL-001', houseNo: '' },
        { masterNo: '', houseNo: 'HBL-001' },
      ],
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
        shippingDocuments: [],
      },
      ORDER_KIND_CONFIGS['sea-export'],
    );

    expect(result.customerReferenceNo).toBeUndefined();
    expect(result.personnelAssignments).toEqual([]);
    expect(result.shippingDocuments).toEqual([]);
  });
});
