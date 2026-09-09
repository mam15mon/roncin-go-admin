import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App, Form } from 'antd';
import React, { useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const serviceMocks = vi.hoisted(() => ({
  listFinanceOrganizationOptions: vi.fn(),
  listInvoiceCreationBills: vi.fn(),
  listInvoiceProfilesForBill: vi.fn(),
}));

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListFinanceOrganizationOptions:
    serviceMocks.listFinanceOrganizationOptions,
  settlementServiceListInvoiceCreationBills:
    serviceMocks.listInvoiceCreationBills,
  settlementServiceListInvoiceProfilesForBill:
    serviceMocks.listInvoiceProfilesForBill,
}));

import InvoiceCreateModal from './InvoiceCreateModal';

function InvoiceCreateModalHarness({ open = true }: { open?: boolean }) {
  const [form] = Form.useForm();
  const [selectedIDs, setSelectedIDs] = useState<React.Key[]>([]);
  const [selectedBills, setSelectedBills] = useState<API.FinanceBill[]>([]);
  return (
    <InvoiceCreateModal
      open={open}
      onCancel={vi.fn()}
      submitting={false}
      createForm={form}
      selectedBills={selectedBills}
      selectedIDs={selectedIDs}
      setSelectedIDs={setSelectedIDs}
      setSelectedBills={setSelectedBills}
      onOk={async () => undefined}
    />
  );
}

describe('InvoiceCreateModal 发票创建组织候选', () => {
  beforeEach(() => {
    serviceMocks.listFinanceOrganizationOptions.mockReset();
    serviceMocks.listInvoiceCreationBills.mockReset();
    serviceMocks.listInvoiceProfilesForBill.mockReset();
    serviceMocks.listFinanceOrganizationOptions.mockResolvedValue({
      data: [
        { id: 'organization-a', code: 'A', name: '公司 A' },
        { id: 'organization-b', code: 'B', name: '公司 B' },
      ],
    });
    serviceMocks.listInvoiceCreationBills.mockImplementation(
      ({ organizationId }) =>
        Promise.resolve({
          data: [
            {
              id: organizationId === 'organization-a' ? 'bill-a' : 'bill-b',
              billNo: organizationId === 'organization-a' ? 'BILL-A' : 'BILL-B',
              organizationId,
              direction: 'RECEIVABLE',
              settlementPartyId: `party-${organizationId}`,
              settlementPartyName: `结算单位 ${organizationId}`,
              currency: 'CNY',
              totalAmount: '100.00000000',
              taxAmount: '6.00000000',
            },
          ],
          total: 1,
        }),
    );
  });

  const selectOrganization = async (name: string) => {
    fireEvent.mouseDown(screen.getByLabelText('所属公司'));
    fireEvent.click(await screen.findByText(name));
  };

  it('未选公司不请求账单，选择可写公司后仅透传该 organizationId', async () => {
    render(
      <App>
        <InvoiceCreateModalHarness />
      </App>,
    );
    await waitFor(() =>
      expect(serviceMocks.listFinanceOrganizationOptions).toHaveBeenCalledWith({
        purpose: 10,
      }),
    );
    expect(serviceMocks.listInvoiceCreationBills).not.toHaveBeenCalled();

    await selectOrganization('公司 A');
    await waitFor(() =>
      expect(serviceMocks.listInvoiceCreationBills).toHaveBeenCalledWith(
        expect.objectContaining({ organizationId: 'organization-a' }),
      ),
    );
    expect(await screen.findByText('BILL-A')).toBeInTheDocument();
  });

  it('切换公司清空账单和资料，并忽略迟到的旧账单资料响应', async () => {
    let resolveOldProfiles: ((value: unknown) => void) | undefined;
    serviceMocks.listInvoiceProfilesForBill.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveOldProfiles = resolve;
        }),
    );
    render(
      <App>
        <InvoiceCreateModalHarness />
      </App>,
    );
    await selectOrganization('公司 A');
    const billRow = (await screen.findByText('BILL-A')).closest('tr');
    if (!billRow) throw new Error('账单候选行不存在');
    fireEvent.click(within(billRow).getByRole('checkbox'));
    await waitFor(() =>
      expect(serviceMocks.listInvoiceProfilesForBill).toHaveBeenCalledWith(
        { billId: 'bill-a' },
        { skipErrorHandler: true },
      ),
    );

    await selectOrganization('公司 B');
    await waitFor(() => {
      expect(screen.queryByText('BILL-A')).not.toBeInTheDocument();
      expect(screen.getByText('BILL-B')).toBeInTheDocument();
      expect(screen.queryByText('已选开票抬头')).not.toBeInTheDocument();
    });
    resolveOldProfiles?.({
      data: {
        data: [
          {
            id: 'profile-a',
            invoiceTitle: 'A 公司旧抬头',
            taxpayerIdentificationNo: '91310000TEST',
            defaultInvoiceType: 'NORMAL',
            isDefault: true,
          },
        ],
      },
    });
    await waitFor(() =>
      expect(screen.queryByText('A 公司旧抬头')).not.toBeInTheDocument(),
    );
  });

  it('关闭弹窗后不会让在途旧账单资料回填', async () => {
    let resolveProfiles: ((value: unknown) => void) | undefined;
    serviceMocks.listInvoiceProfilesForBill.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveProfiles = resolve;
        }),
    );
    const view = render(
      <App>
        <InvoiceCreateModalHarness />
      </App>,
    );
    await selectOrganization('公司 A');
    const billRow = (await screen.findByText('BILL-A')).closest('tr');
    if (!billRow) throw new Error('账单候选行不存在');
    fireEvent.click(within(billRow).getByRole('checkbox'));
    await waitFor(() =>
      expect(serviceMocks.listInvoiceProfilesForBill).toHaveBeenCalledWith(
        { billId: 'bill-a' },
        { skipErrorHandler: true },
      ),
    );
    view.rerender(
      <App>
        <InvoiceCreateModalHarness open={false} />
      </App>,
    );
    resolveProfiles?.({
      data: {
        data: [
          { id: 'profile-a', invoiceTitle: '关闭后旧抬头', isDefault: true },
        ],
      },
    });
    view.rerender(
      <App>
        <InvoiceCreateModalHarness />
      </App>,
    );
    await waitFor(() =>
      expect(screen.queryByText('关闭后旧抬头')).not.toBeInTheDocument(),
    );
  });
});
