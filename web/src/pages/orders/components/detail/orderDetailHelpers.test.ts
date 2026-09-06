import { describe, expect, it } from 'vitest';
import { OrderPersonnelRole } from '@/enums.generated';
import { buildInitialValues, buildUpdatePayload } from './orderDetailHelpers';

describe('buildInitialValues', () => {
  it('按生成的人员角色枚举回填订单协作人员', () => {
    const result = buildInitialValues(
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
        creatorOrganizationId: 'org-creator',
        operatorUserId: 'operator',
        operatorOrganizationId: 'org-operator',
        commercialUserId: 'commercial',
        commercialOrganizationId: 'org-commercial',
      }),
    );
  });

  it('回填当前共享主单版本供单票身份更正做乐观锁校验', () => {
    const result = buildInitialValues(
      {
        id: 'order-1',
        carrierId: 'carrier-1',
        seaMasterBill: {
          masterBillId: 'mbl-1',
          masterNo: 'COSCO123456',
          issuerPartnerId: 'issuer-1',
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
        carrierId: 'carrier-1',
        seaMasterBillExpectedCandidateVersion: '7',
      }),
    );
    expect(result).not.toHaveProperty('seaMasterBillIssuerPartnerId');
  });

  it('更新海运订单时从船公司派生 MBL 签发方', () => {
    const result = buildUpdatePayload('order-1', '7', {
      customerId: 'customer-1',
      tradeTerm: 3,
      paymentTerm: 1,
      carrierId: 'carrier-2',
      seaMasterBillMasterNo: 'COSCO123457',
      seaDocumentStructure: 1,
    });

    expect(result.seaMasterBill).toEqual({
      masterNo: 'COSCO123457',
      issuerPartnerId: 'carrier-2',
    });
  });
});
