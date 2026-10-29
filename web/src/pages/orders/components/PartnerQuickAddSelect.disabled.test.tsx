import { act, fireEvent, render, screen } from '@testing-library/react';
import { App, Form } from 'antd';
import { expect, it, vi } from 'vitest';
import { PartnerRoleType } from '@/enums.generated';
import PartnerQuickAddSelect from './PartnerQuickAddSelect';

vi.mock('@/app/access', () => ({
  useAccess: () => ({ canCreatePartners: true }),
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: { currentUser: { currentOrganization: { id: 'org' } } },
  }),
}));

it('父表单禁用时客商选择继承禁用，不能在订单提交中继续更换客户', async () => {
  render(
    <App>
      <Form disabled>
        <PartnerQuickAddSelect
          name="customerId"
          displayName="委托单位"
          role={PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER}
          createRoute="/partners/customers/create"
          searchPartners={async () => []}
        />
      </Form>
    </App>,
  );
  await act(async () => {});
  expect(screen.getByRole('combobox')).toBeDisabled();
  fireEvent.mouseDown(screen.getByRole('combobox'));
  expect(
    screen.queryByRole('button', { name: '新增 委托单位' }),
  ).not.toBeInTheDocument();
});
