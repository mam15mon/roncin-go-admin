import { describe, expect, it } from 'vitest';
import { OrderPersonnelRole } from '@/enums.generated';
import { buildInitialValues } from './orderDetailHelpers';

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
});
