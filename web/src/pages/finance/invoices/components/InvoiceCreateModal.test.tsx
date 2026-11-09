import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { App, Form } from 'antd';
import React, { useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const serviceMocks = vi.hoisted(() => ({
  listFinanceOrganizationOptions: vi.fn(),
  organizationId: 'organization-a' as string | undefined,
  canOperate: true,
  onOk: vi.fn(),
  listInvoiceCreationBills: vi.fn(),
  listInvoiceProfilesForBill: vi.fn(),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user',
        currentOrganization: { id: serviceMocks.organizationId },
      },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canCreateFinanceInvoices: serviceMocks.canOperate,
    canOperateOrganization: (id?: string) =>
      serviceMocks.canOperate &&
      Boolean(id) &&
      id === serviceMocks.organizationId,
  }),
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

function InvoiceCreateModalHarness({
  open = true,
  initialBills = [],
}: {
  open?: boolean;
  initialBills?: API.FinanceBill[];
}) {
  const [form] = Form.useForm();
  const [selectedIDs, setSelectedIDs] = useState<React.Key[]>([]);
  const [selectedBills, setSelectedBills] =
    useState<API.FinanceBill[]>(initialBills);
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
      onOk={serviceMocks.onOk}
    />
  );
}

describe('InvoiceCreateModal 发票创建组织候选', () => {
  beforeEach(() => {
    serviceMocks.organizationId = 'organization-a';
    serviceMocks.canOperate = true;
    serviceMocks.onOk.mockReset();
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
              totalAmount: '1200.00000000',
              invoiceableAmount: '100.00000000',
              invoiceableTaxAmount: '6.00000000',
              taxAmount: '6.00000000',
            },
          ],
          total: 1,
        }),
    );
  });

  it('无需选择公司即请求当前公司账单，保留开票金额展示', async () => {
    renderWithClient(
      <App>
        <InvoiceCreateModalHarness />
      </App>,
    );
    await waitFor(() =>
      expect(serviceMocks.listInvoiceCreationBills).toHaveBeenCalledWith(
        expect.objectContaining({ organizationId: 'organization-a' }),
      ),
    );
    expect(serviceMocks.listFinanceOrganizationOptions).not.toHaveBeenCalled();
    expect(screen.queryByLabelText('所属公司')).not.toBeInTheDocument();
    expect(await screen.findByText('BILL-A')).toBeInTheDocument();
    expect(screen.getByText('100.00 CNY')).toBeInTheDocument();
    expect(screen.queryByText('1,200.00 CNY')).not.toBeInTheDocument();
  });

  it.each([undefined, 'system'])(
    '身份%s不在公司工作区时不请求、不提交',
    async (id) => {
      serviceMocks.organizationId = id;
      serviceMocks.canOperate = false;
      renderWithClient(
        <App>
          <InvoiceCreateModalHarness />
        </App>,
      );
      fireEvent.click(screen.getByRole('button', { name: /OK|确\s*定/ }));
      expect(serviceMocks.listInvoiceCreationBills).not.toHaveBeenCalled();
      expect(serviceMocks.listInvoiceProfilesForBill).not.toHaveBeenCalled();
      expect(serviceMocks.onOk).not.toHaveBeenCalled();
    },
  );

  it('注入外公司账单不能读取抬头或提交', async () => {
    renderWithClient(
      <App>
        <InvoiceCreateModalHarness
          initialBills={[
            { id: 'foreign-bill', organizationId: 'organization-b' },
          ]}
        />
      </App>,
    );
    await waitFor(() =>
      expect(serviceMocks.listInvoiceCreationBills).toHaveBeenCalled(),
    );
    fireEvent.click(screen.getByRole('button', { name: /OK|确\s*定/ }));
    expect(serviceMocks.listInvoiceProfilesForBill).not.toHaveBeenCalled();
    expect(serviceMocks.onOk).not.toHaveBeenCalled();
  });

  it('切换公司清空账单和资料，并忽略迟到的旧账单资料响应', async () => {
    let resolveOldProfiles: ((value: unknown) => void) | undefined;
    serviceMocks.listInvoiceProfilesForBill.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveOldProfiles = resolve;
        }),
    );
    const view = renderWithClient(
      <App>
        <InvoiceCreateModalHarness />
      </App>,
    );
    const billRow = (await screen.findByText('BILL-A')).closest('tr');
    if (!billRow) throw new Error('账单候选行不存在');
    fireEvent.click(within(billRow).getByRole('checkbox'));
    await waitFor(() =>
      expect(serviceMocks.listInvoiceProfilesForBill).toHaveBeenCalledWith(
        { billId: 'bill-a' },
        { skipErrorHandler: true },
      ),
    );

    serviceMocks.organizationId = 'organization-b';
    view.rerender(
      <QueryClientProvider client={view.queryClient}>
        <App>
          <InvoiceCreateModalHarness key="B" />
        </App>
      </QueryClientProvider>,
    );
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
    const view = renderWithClient(
      <App>
        <InvoiceCreateModalHarness />
      </App>,
    );
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
      <QueryClientProvider client={view.queryClient}>
        <App>
          <InvoiceCreateModalHarness open={false} />
        </App>
      </QueryClientProvider>,
    );
    resolveProfiles?.({
      data: {
        data: [
          { id: 'profile-a', invoiceTitle: '关闭后旧抬头', isDefault: true },
        ],
      },
    });
    view.rerender(
      <QueryClientProvider client={view.queryClient}>
        <App>
          <InvoiceCreateModalHarness />
        </App>
      </QueryClientProvider>,
    );
    await waitFor(() =>
      expect(screen.queryByText('关闭后旧抬头')).not.toBeInTheDocument(),
    );
  });
});
