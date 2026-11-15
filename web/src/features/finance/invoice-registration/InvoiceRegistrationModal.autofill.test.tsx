import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import InvoiceRegistrationModal from './InvoiceRegistrationModal';
import {
  invoiceBill,
  invoiceUser,
  NoMotion,
  submitRegistration,
} from './InvoiceRegistrationModal.fixtures';

const mocks = vi.hoisted(() => ({
  profiles: vi.fn(),
  candidates: vi.fn(),
  register: vi.fn(),
  impact: vi.fn(),
  revise: vi.fn(),
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListInvoiceProfilesForParty: mocks.profiles,
  settlementServiceListInvoiceCreationBills: mocks.candidates,
  settlementServiceRegisterInvoices: mocks.register,
  settlementServiceGetInvoiceRevisionImpact: mocks.impact,
  settlementServiceReviseInvoiceRegistrations: mocks.revise,
}));
const taxedBill: API.FinanceBill = {
  ...invoiceBill,
  invoiceableAmount: '2000',
  lines: [
    {
      ...invoiceBill.lines?.[0],
      totalAmount: '2000',
      invoiceableAmount: '2000',
      taxRate: '6',
    },
  ],
};
const twoFeeBill: API.FinanceBill = {
  ...invoiceBill,
  id: 'bill-two',
  billNo: 'BI-002',
  lines: [
    {
      id: 'l1',
      orderFeeId: 'f1',
      active: true,
      feeCode: 'OF',
      feeName: '海运费',
      invoiceableAmount: '100',
      taxRate: '6',
    },
    {
      id: 'l2',
      orderFeeId: 'f2',
      active: true,
      feeCode: 'DOC',
      feeName: '文件费',
      invoiceableAmount: '50',
      taxRate: '9',
    },
  ],
};
beforeEach(() => {
  vi.clearAllMocks();
  mocks.profiles.mockResolvedValue({
    success: true,
    data: {
      organizationId: 'company',
      settlementPartyId: 'party',
      data: [
        {
          id: 'profile',
          invoiceTitle: '测试抬头',
          taxpayerIdentificationNo: 'TAX',
        },
      ],
    },
  });
  mocks.candidates.mockImplementation(async ({ billId }) => ({
    success: true,
    data: [{ id: billId, version: invoiceBill.version }],
    total: '1',
  }));
  mocks.register.mockResolvedValue({
    success: true,
    data: [{ id: 'invoice', organizationId: 'company' }],
  });
});
function open(bills: API.FinanceBill[]) {
  renderWithApp(
    <NoMotion>
      <InvoiceRegistrationModal
        open
        bills={bills}
        onOpenChange={vi.fn()}
        onSuccess={vi.fn()}
      />
    </NoMotion>,
    { user: invoiceUser },
  );
  return within(screen.getByRole('dialog'));
}
async function fillHeader(dialog: ReturnType<typeof within>, total: string) {
  fireEvent.mouseDown(await dialog.findByLabelText('开票抬头'));
  fireEvent.click(await screen.findByText('测试抬头 · TAX'));
  fireEvent.change(dialog.getByLabelText('已有发票号码'), {
    target: { value: 'NO-1' },
  });
  fireEvent.change(dialog.getByLabelText('发票含税总额'), {
    target: { value: total },
  });
}

describe('登记自动带入', () => {
  it('一账单一费用：建议税额预填可改，手改后再改总额不覆盖税额', async () => {
    const dialog = open([taxedBill]);
    await fillHeader(dialog, '1060');
    const tax = dialog.getByLabelText('本次登记税额');
    await waitFor(() => expect(tax).toHaveValue('60.00'));
    expect(
      dialog.getByText('建议值，请以实际发票税额为准'),
    ).toBeInTheDocument();
    fireEvent.change(tax, { target: { value: '59.99' } });
    fireEvent.change(dialog.getByLabelText('发票含税总额'), {
      target: { value: '1000' },
    });
    await waitFor(() =>
      expect(dialog.getByLabelText('本次含税金额')).toHaveValue('1000'),
    );
    expect(tax).toHaveValue('59.99');
    expect(
      dialog.queryByText('建议值，请以实际发票税额为准'),
    ).not.toBeInTheDocument();
    submitRegistration();
    await waitFor(() => expect(mocks.register).toHaveBeenCalled());
    expect(mocks.register.mock.calls[0][0].records[0]).toMatchObject({
      totalAmount: '1000.00000000',
      allocations: [
        {
          amount: '1000.00000000',
          fees: [{ amount: '1000.00000000', taxAmount: '59.99000000' }],
        },
      ],
    });
  });
  it('超过可登记余额时行内提示并不提交', async () => {
    const dialog = open([invoiceBill]);
    await fillHeader(dialog, '150');
    fireEvent.change(dialog.getByLabelText('本次登记税额'), {
      target: { value: '0' },
    });
    submitRegistration();
    await waitFor(() =>
      expect(dialog.getByText('超过可登记余额 100.00')).toBeInTheDocument(),
    );
    expect(mocks.register).not.toHaveBeenCalled();
  });
  it('多费用按可登记余额分配：逐费建议税额，余额不足显示差额并阻止提交', async () => {
    const dialog = open([twoFeeBill]);
    await fillHeader(dialog, '200');
    fireEvent.click(dialog.getByText('按可登记余额分配'));
    fireEvent.click(await screen.findByText('确 定'));
    await waitFor(() =>
      expect(dialog.getAllByLabelText('本次含税金额')).toHaveLength(2),
    );
    expect(
      dialog
        .getAllByLabelText('本次含税金额')
        .map((input) => input.getAttribute('value')),
    ).toEqual(['100', '50']);
    expect(
      dialog
        .getAllByLabelText('本次登记税额')
        .map((input) => input.getAttribute('value')),
    ).toEqual(['5.66', '4.13']);
    expect(dialog.getByLabelText('本账单含税分配（CNY）')).toHaveValue('150');
    expect(dialog.getByText('待分配 50.00')).toBeInTheDocument();
    submitRegistration();
    await waitFor(() =>
      expect(
        screen.getByText('第1张发票：账单分配合计须等于发票总额'),
      ).toBeInTheDocument(),
    );
    expect(mocks.register).not.toHaveBeenCalled();
  });
  it('批量勾选账单默认带出余额合计并分配平，改总额显示差额，继续添加默认剩余', async () => {
    const second = { ...twoFeeBill, version: invoiceBill.version };
    mocks.candidates.mockImplementation(async ({ billId }) =>
      billId
        ? {
            success: true,
            data: [{ id: billId, version: invoiceBill.version }],
            total: '1',
          }
        : { success: true, data: [invoiceBill, second], total: '2' },
    );
    renderWithApp(
      <NoMotion>
        <InvoiceRegistrationModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </NoMotion>,
      { user: invoiceUser },
    );
    const dialog = within(await screen.findByRole('dialog'));
    await dialog.findByText('BI-002');
    for (const name of ['BI-001', 'BI-002'])
      fireEvent.click(
        within(dialog.getByText(name).closest('tr') as HTMLElement).getByRole(
          'checkbox',
        ),
      );
    await waitFor(() =>
      expect(dialog.getByLabelText('发票含税总额')).toHaveValue('250'),
    );
    expect(
      dialog.getByText('默认为所选账单未登记余额，请按实际发票金额修改'),
    ).toBeInTheDocument();
    expect(dialog.getByText('待分配 0.00')).toBeInTheDocument();
    expect(
      dialog
        .getAllByLabelText('本次含税金额')
        .map((input) => input.getAttribute('value')),
    ).toEqual(['100', '100', '50']);
    fireEvent.change(dialog.getByLabelText('发票含税总额'), {
      target: { value: '240' },
    });
    await waitFor(() =>
      expect(dialog.getByText('待分配 -10.00')).toBeInTheDocument(),
    );
    expect(
      dialog.queryByText('默认为所选账单未登记余额，请按实际发票金额修改'),
    ).not.toBeInTheDocument();
    fireEvent.click(dialog.getByText('继续添加发票'));
    await waitFor(() =>
      expect(dialog.getAllByLabelText('发票含税总额')).toHaveLength(2),
    );
    expect(dialog.getAllByLabelText('发票含税总额')[1]).toHaveValue('');
  });
});
