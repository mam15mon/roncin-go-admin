import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import InvoiceRegistrationModal from './InvoiceRegistrationModal';
import {
  fillRegistration,
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
  mocks.candidates.mockResolvedValue({
    success: true,
    data: [invoiceBill],
    total: '1',
  });
});
describe('登记输入会话', () => {
  it('在途继续编辑使旧成功失去关闭资格，保留新输入', async () => {
    let resolve: ((response: unknown) => void) | undefined;
    mocks.register.mockImplementation(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    const closed = vi.fn();
    renderWithApp(
      <NoMotion>
        <InvoiceRegistrationModal
          open
          bills={[invoiceBill]}
          onOpenChange={closed}
          onSuccess={vi.fn()}
        />
      </NoMotion>,
      { user: invoiceUser },
    );
    const dialog = await fillRegistration();
    submitRegistration();
    await waitFor(() => expect(mocks.register).toHaveBeenCalled());
    fireEvent.change(dialog.getByLabelText('已有发票号码'), {
      target: { value: '新输入' },
    });
    await act(async () => {
      resolve?.({
        success: true,
        data: [{ id: 'old', organizationId: 'company' }],
      });
    });
    expect(
      await screen.findByText(
        '原提交已保存，当前新增输入已保留。请核对登记结果后重新打开。',
      ),
    ).toBeInTheDocument();
    expect(dialog.getByLabelText('已有发票号码')).toHaveValue('新输入');
    expect(closed).not.toHaveBeenCalledWith(false);
  });
  it('关闭重开后迟到成功不关闭新表单，原意图重试仍用原键', async () => {
    let resolve: ((response: unknown) => void) | undefined;
    mocks.register
      .mockImplementationOnce(
        () =>
          new Promise((r) => {
            resolve = r;
          }),
      )
      .mockResolvedValueOnce(undefined);
    const closed = vi.fn();
    const view = renderWithApp(
      <NoMotion>
        <InvoiceRegistrationModal
          open
          bills={[invoiceBill]}
          onOpenChange={closed}
          onSuccess={vi.fn()}
        />
      </NoMotion>,
      { user: invoiceUser },
    );
    await fillRegistration();
    submitRegistration();
    await waitFor(() => expect(mocks.register).toHaveBeenCalled());
    const key = mocks.register.mock.calls[0][0].idempotencyKey;
    view.rerender(
      <NoMotion>
        <InvoiceRegistrationModal
          open={false}
          bills={[invoiceBill]}
          onOpenChange={closed}
          onSuccess={vi.fn()}
        />
      </NoMotion>,
    );
    view.rerender(
      <NoMotion>
        <InvoiceRegistrationModal
          open
          bills={[invoiceBill]}
          onOpenChange={closed}
          onSuccess={vi.fn()}
        />
      </NoMotion>,
    );
    const next = await fillRegistration();
    await act(async () => {
      resolve?.({
        success: true,
        data: [{ id: 'old', organizationId: 'company' }],
      });
    });
    expect(next.getByLabelText('已有发票号码')).toHaveValue('NO-1');
    expect(closed).not.toHaveBeenCalledWith(false);
    submitRegistration();
    await waitFor(() => expect(mocks.register).toHaveBeenCalledTimes(2));
    expect(mocks.register.mock.calls[1][0].idempotencyKey).toBe(key);
    await screen.findByText('登记结果未确认，请保留输入并重试');
  });
});
