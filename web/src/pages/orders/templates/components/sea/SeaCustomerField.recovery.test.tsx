import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { Form, Input } from 'antd';
import { expect, it, vi } from 'vitest';
import { PartnerAssignmentRole } from '@/enums.generated';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import { SeaCustomerField } from './SeaBasicInfoSection';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceGetPartner: vi.fn(),
}));
vi.mock('../../../components/PartnerQuickAddSelect', () => ({
  default: () => <div>委托单位</div>,
}));

it('草稿恢复时客户人员未加载完成或失败，可原地重试且保留手动人员', async () => {
  const getPartner = vi.mocked(partnerServiceGetPartner);
  getPartner.mockRejectedValueOnce(new Error('网络失败'));
  getPartner.mockResolvedValueOnce({
    data: {
      assignments: [
        {
          role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_OPERATOR,
          userId: 'default-operator',
        },
        {
          role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
          userId: 'sales',
        },
        {
          role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE,
          userId: 'service',
        },
        {
          role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_DOCUMENT,
          userId: 'default-document',
        },
      ],
    },
  });
  renderWithClient(
    <Form
      initialValues={{
        customerId: 'restored-customer',
        operatorUserId: 'manual-operator',
        salesUserId: null,
        customerServiceUserId: null,
      }}
    >
      <SeaCustomerField
        searchCustomers={async () => []}
        setCustomerCode={() => {}}
      />
      <Form.Item name="operatorUserId" label="操作员">
        <Input />
      </Form.Item>
      <Form.Item name="salesUserId" label="业务">
        <Input />
      </Form.Item>
      <Form.Item name="customerServiceUserId" label="客服">
        <Input />
      </Form.Item>
      <Form.Item name="documentUserId" label="单证">
        <Input />
      </Form.Item>
    </Form>,
  );
  expect(getPartner).not.toHaveBeenCalled();
  fireEvent.click(
    await screen.findByRole('button', { name: '重新加载客户人员' }),
  );
  await screen.findByText('客户人员加载失败，请重试');
  fireEvent.change(screen.getByLabelText('单证'), {
    target: { value: 'manual-document' },
  });
  fireEvent.click(screen.getByRole('button', { name: '重试人员加载' }));
  await waitFor(() =>
    expect(screen.getByLabelText('业务')).toHaveValue('sales'),
  );
  expect(screen.getByLabelText('客服')).toHaveValue('service');
  expect(screen.getByLabelText('操作员')).toHaveValue('manual-operator');
  expect(screen.getByLabelText('单证')).toHaveValue('manual-document');
});
