import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FinanceInvoiceStatus } from '@/enums.generated';
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
const invoice: API.FinanceInvoice = {
  id: 'invoice',
  version: '1',
  organizationId: 'company',
  invoiceProfileId: 'profile',
  invoiceTitle: '原快照抬头',
  taxpayerIdentificationNo: 'OLD-TAX',
  status: FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED,
  invoiceType: 'NORMAL',
  taxInvoiceNo: 'OLD-1',
  invoiceDate: '2026-11-14',
  totalAmount: '30',
  billLinks: [
    {
      id: 'link',
      billId: 'bill',
      billNo: 'BI-001',
      active: true,
      amount: '30',
      fees: [
        { id: 'allocation', orderFeeId: 'fee', amount: '30', taxAmount: '0' },
      ],
    },
  ],
};
beforeEach(() => {
  vi.clearAllMocks();
  mocks.impact.mockResolvedValue({
    success: true,
    data: { invoice, bills: [invoiceBill], canRevise: true, dependencies: [] },
  });
  mocks.revise.mockResolvedValue({
    success: true,
    data: [{ ...invoice, version: '2' }],
  });
});
describe('发票修订与影响', () => {
  it('只有更新权限可修原票且保留原抬头，不能新增；提交包含全部账单与原票基线', async () => {
    const user = {
      ...invoiceUser,
      permissionCapabilities: invoiceUser.permissionCapabilities?.filter(
        (permission) => permission.key !== 'system.finance.invoice.create',
      ),
    };
    renderWithApp(
      <NoMotion>
        <InvoiceRegistrationModal
          open
          reviseInvoice={invoice}
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </NoMotion>,
      { user },
    );
    await waitFor(() =>
      expect(screen.getByLabelText('已有发票号码')).toHaveValue('OLD-1'),
    );
    expect(screen.queryByText('继续添加发票')).not.toBeInTheDocument();
    expect(mocks.profiles).not.toHaveBeenCalled();
    expect(mocks.candidates).not.toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText('修订原因'), {
      target: { value: '修正原号码' },
    });
    fireEvent.change(screen.getByLabelText('已有发票号码'), {
      target: { value: 'NEW-1' },
    });
    submitRegistration();
    await waitFor(() => expect(mocks.revise).toHaveBeenCalled());
    expect(mocks.revise.mock.calls[0][0]).toMatchObject({
      reason: '修正原号码',
      billVersions: [{ billId: 'bill', expectedVersion: '2' }],
      entries: [
        {
          id: 'invoice',
          expectedVersion: '1',
          registration: { taxInvoiceNo: 'NEW-1', invoiceProfileId: 'profile' },
        },
      ],
    });
  });
  it('wire省略canRevise即按false处理，展示全部跨账单结算影响并禁止提交', async () => {
    mocks.impact.mockResolvedValue({
      success: true,
      data: {
        invoice,
        bills: [
          invoiceBill,
          { ...invoiceBill, id: 'bill-two', billNo: 'BI-002' },
        ],
        dependencies: [
          {
            id: 'verification',
            billId: 'bill-two',
            billNo: 'BI-002',
            kind: 'VERIFICATION',
            documentNo: '核销001',
            currency: 'CNY',
            amount: '10',
          },
        ],
      },
    });
    renderWithApp(
      <NoMotion>
        <InvoiceRegistrationModal
          open
          reviseInvoice={invoice}
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </NoMotion>,
      { user: invoiceUser },
    );
    await waitFor(() =>
      expect(screen.getByText(/本次修订影响全部 2 张账单/)).toBeInTheDocument(),
    );
    expect(
      screen.getByText(/核销001 · 账单 BI-002 · 10.00 CNY/),
    ).toBeInTheDocument();
    const button = screen.getByText('保存登记').closest('button');
    expect(button).toBeDisabled();
    submitRegistration();
    expect(mocks.revise).not.toHaveBeenCalled();
    await act(async () => {});
  });
});
