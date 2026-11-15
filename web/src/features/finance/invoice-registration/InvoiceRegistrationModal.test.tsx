import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
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
  mocks.register.mockResolvedValue({
    success: true,
    data: [{ id: 'invoice', organizationId: 'company' }],
  });
});
describe('登记弹窗', () => {
  it('当前公司查询及部分可登记余额保留，不提供公司选择', async () => {
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
    await waitFor(() =>
      expect(mocks.candidates).toHaveBeenCalledWith(
        expect.objectContaining({ organizationId: 'company' }),
      ),
    );
    expect(await screen.findByText('BI-001')).toBeInTheDocument();
    expect(screen.getByText('100.00 CNY')).toBeInTheDocument();
    expect(screen.queryByLabelText('所属公司')).not.toBeInTheDocument();
  });
  it('无公司身份不发经营请求也不能提交', async () => {
    renderWithApp(
      <NoMotion>
        <InvoiceRegistrationModal
          open
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </NoMotion>,
    );
    expect(screen.getByText('当前公司或权限不允许登记')).toBeInTheDocument();
    submitRegistration();
    expect(mocks.candidates).not.toHaveBeenCalled();
    expect(mocks.profiles).not.toHaveBeenCalled();
    expect(mocks.register).not.toHaveBeenCalled();
    await act(async () => {});
  });
  it('外公司来源不加载抬头、不允许提交', async () => {
    renderWithApp(
      <NoMotion>
        <InvoiceRegistrationModal
          open
          bills={[{ ...invoiceBill, organizationId: 'other' }]}
          onOpenChange={vi.fn()}
          onSuccess={vi.fn()}
        />
      </NoMotion>,
      { user: invoiceUser },
    );
    expect(screen.queryByText('继续添加发票')).not.toBeInTheDocument();
    submitRegistration();
    expect(mocks.profiles).not.toHaveBeenCalled();
    expect(mocks.register).not.toHaveBeenCalled();
    await act(async () => {});
  });
  it('直登记发送三层明确分配与手填税额，采纳成功后关闭并刷新', async () => {
    const closed = vi.fn(),
      saved = vi.fn();
    renderWithApp(
      <NoMotion>
        <InvoiceRegistrationModal
          open
          bills={[invoiceBill]}
          onOpenChange={closed}
          onSuccess={saved}
        />
      </NoMotion>,
      { user: invoiceUser },
    );
    await fillRegistration();
    submitRegistration();
    await waitFor(() => expect(mocks.register).toHaveBeenCalled());
    expect(mocks.register.mock.calls[0][0]).toMatchObject({
      records: [
        {
          invoiceProfileId: 'profile',
          taxInvoiceNo: 'NO-1',
          totalAmount: '30.00000000',
          allocations: [
            {
              billId: 'bill',
              expectedVersion: '2',
              amount: '30.00000000',
              fees: [
                {
                  orderFeeId: 'fee',
                  amount: '30.00000000',
                  taxAmount: '3.00000001',
                },
              ],
            },
          ],
        },
      ],
    });
    expect(mocks.register.mock.calls[0][0]).not.toHaveProperty(
      'organizationId',
    );
    await waitFor(() => expect(closed).toHaveBeenCalledWith(false));
    expect(saved).toHaveBeenCalledOnce();
  });
  it('未确认结果保留输入；A→B→A复用原幂等意图并不丢原税额', async () => {
    mocks.register.mockResolvedValue(undefined);
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
    await waitFor(() => expect(mocks.register).toHaveBeenCalledTimes(1));
    await screen.findByText('登记结果未确认，请保留输入并重试');
    const first = mocks.register.mock.calls[0][0].idempotencyKey;
    expect(dialog.getByLabelText('已有发票号码')).toHaveValue('NO-1');
    fireEvent.change(dialog.getByLabelText('已有发票号码'), {
      target: { value: 'NO-2' },
    });
    submitRegistration();
    await waitFor(() => expect(mocks.register).toHaveBeenCalledTimes(2));
    expect(mocks.register.mock.calls[1][0].idempotencyKey).not.toBe(first);
    fireEvent.change(dialog.getByLabelText('已有发票号码'), {
      target: { value: ' NO-1 ' },
    });
    submitRegistration();
    await waitFor(() => expect(mocks.register).toHaveBeenCalledTimes(3));
    expect(mocks.register.mock.calls[2][0].idempotencyKey).toBe(first);
    expect(closed).not.toHaveBeenCalledWith(false);
  });
  it('候选查询不嵌套表单，回车只查询候选不提交登记；唯一抬头自动选中', async () => {
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
    expect(await screen.findByText('BI-001')).toBeInTheDocument();
    expect(document.querySelectorAll('form form')).toHaveLength(0);
    const keyword = screen.getByPlaceholderText('账单号、结算单位、拼音');
    fireEvent.change(keyword, { target: { value: 'BI-001' } });
    fireEvent.keyDown(keyword, { key: 'Enter', code: 'Enter', keyCode: 13 });
    await waitFor(() =>
      expect(mocks.candidates).toHaveBeenCalledWith(
        expect.objectContaining({ keyword: 'BI-001' }),
      ),
    );
    expect(mocks.register).not.toHaveBeenCalled();
    fireEvent.click(
      within(screen.getByText('BI-001').closest('tr') as HTMLElement).getByRole(
        'checkbox',
      ),
    );
    await waitFor(() =>
      expect(screen.getByText('测试抬头 · TAX')).toBeInTheDocument(),
    );
  });
});
