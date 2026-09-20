import { ProForm } from '@ant-design/pro-components';
import { act, render } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PartnerRoleType } from '@/enums.generated';
import { buildSeaBaseInfoSection } from './SeaBasicInfoSection';

const quickAddCalls = vi.hoisted(() => [] as Array<Record<string, unknown>>);

vi.mock('../../../components/PartnerQuickAddSelect', () => ({
  default: (props: Record<string, unknown>) => {
    quickAddCalls.push(props);
    return <div data-testid={`partner-field-${String(props.name)}`} />;
  },
}));

describe('SeaBasicInfoSection 伙伴快捷新增映射', () => {
  beforeEach(() => {
    quickAddCalls.length = 0;
  });

  it('委托单位、订舱代理和国外代理使用各自角色与完整档案路由', async () => {
    const section = buildSeaBaseInfoSection({
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
      locationOptions: [],
      searchLocations: vi.fn().mockResolvedValue([]),
      currencyOptions: [],
      containerSpecOptions: [],
      searchCustomers: vi.fn().mockResolvedValue([]),
      searchShippingLines: vi.fn().mockResolvedValue([]),
      searchBookingAgents: vi.fn().mockResolvedValue([]),
      searchForeignAgents: vi.fn().mockResolvedValue([]),
      searchShippingAgents: vi.fn().mockResolvedValue([]),
      setCustomerCode: vi.fn(),
      checkCustomerReferenceNo: vi.fn().mockResolvedValue(undefined),
      checkInternalReferenceNo: vi.fn().mockResolvedValue(undefined),
      personnelOptions: [],
    });

    render(<ProForm submitter={false}>{section.content}</ProForm>);

    expect(
      quickAddCalls.map(({ name, role, createRoute }) => ({
        name,
        role,
        createRoute,
      })),
    ).toEqual([
      {
        name: 'customerId',
        role: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
        createRoute: '/partners/customers/create',
      },
      {
        name: 'bookingAgentId',
        role: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
        createRoute: '/partners/suppliers/create',
      },
      {
        name: 'foreignAgentId',
        role: PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT,
        createRoute: '/partners/foreign-agents/create',
      },
    ]);
    // 船公司/船代等 request 型下拉的挂载期查询在 act 内落地，避免迟到更新
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });
  });
});
