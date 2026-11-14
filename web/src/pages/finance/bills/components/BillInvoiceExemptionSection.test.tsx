import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FinanceBillFeeLockFact, FinanceBillStatus } from '@/enums.generated';
import { formatDate } from '@/utils/format';
import BillInvoiceExemptionSection from './BillInvoiceExemptionSection';

const mocks = vi.hoisted(() => ({
  mark: vi.fn(),
  revoke: vi.fn(),
  probe: vi.fn(),
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceSetBillInvoiceExemption: mocks.mark,
  settlementServiceRevokeBillInvoiceExemption: mocks.revoke,
  settlementServiceGetBill: mocks.probe,
}));

const bill: API.FinanceBill = {
  id: 'bill-a',
  billNo: 'BILL-A',
  organizationId: 'company-a',
  version: '3',
  status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
};
const activeExemption: API.FinanceInvoiceExemption = {
  id: 'exemption-a',
  reason: '合同约定无需发票',
  markedBy: 'private-user-id',
  markedByName: '张三',
  createdAt: '2026-11-14T08:20:00Z',
  version: '1',
};
const updated = { ...bill, version: '4', invoiceExemptions: [activeExemption] };

function mount(
  detail = bill,
  actions = { onUpdated: vi.fn(), onRefresh: vi.fn() },
) {
  const view = renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <BillInvoiceExemptionSection bill={detail} actions={actions} />
    </ConfigProvider>,
  );
  return { ...view, actions };
}
async function open(mode = '标记无需发票') {
  fireEvent.click(screen.getByRole('button', { name: mode }));
  const dialog = await screen.findByRole('dialog', { name: mode });
  await waitFor(() => expect(dialog).toBeVisible());
  return within(dialog);
}
function fill(
  dialog: ReturnType<typeof within>,
  reason = ' 客户书面同意 ',
  label = '免票原因',
) {
  fireEvent.change(dialog.getByLabelText(label), { target: { value: reason } });
}
function save(dialog: ReturnType<typeof within>) {
  fireEvent.click(dialog.getByRole('button', { name: /保\s*存/ }));
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  mocks.probe.mockResolvedValue({
    success: true,
    data: { id: bill.id, version: bill.version },
  });
  mocks.mark.mockResolvedValue({ success: true, data: updated });
  mocks.revoke.mockResolvedValue({
    success: true,
    data: { ...bill, version: '5' },
  });
});

describe('账单无需发票决定', () => {
  it('无操作权限仍显示有效与已撤销历史；省略撤销时间表示有效，人员不暴露内部ID', async () => {
    renderWithApp(
      <BillInvoiceExemptionSection
        bill={{
          ...bill,
          invoiceExemptions: [
            activeExemption,
            {
              ...activeExemption,
              id: 'old',
              reason: '历史约定',
              markedByName: undefined,
              revokedAt: '2026-11-14T09:30:00Z',
              revokedByName: '李四',
              revocationReason: '改为实际开票',
            },
          ],
        }}
      />,
    );
    await act(async () => {});
    expect(
      screen.queryByRole('button', { name: /无需发票/ }),
    ).not.toBeInTheDocument();
    expect(screen.getByText('有效')).toBeInTheDocument();
    expect(screen.getByText('已撤销')).toBeInTheDocument();
    expect(screen.getByText('张三')).toBeInTheDocument();
    expect(screen.getByText('未命名人员')).toBeInTheDocument();
    expect(screen.getByText('李四')).toBeInTheDocument();
    expect(screen.queryByText('private-user-id')).not.toBeInTheDocument();
    expect(
      screen.getByText(formatDate('2026-11-14T09:30:00Z', 'datetime')),
    ).toBeInTheDocument();
  });

  it('标记必须填写原因，保存提交当前账单版本，成功回传账单并关闭', async () => {
    const { actions } = mount();
    const dialog = await open();
    save(dialog);
    expect(await dialog.findByText('请填写原因')).toBeInTheDocument();
    expect(mocks.mark).not.toHaveBeenCalled();
    fill(dialog);
    save(dialog);
    await waitFor(() =>
      expect(actions.onUpdated).toHaveBeenCalledWith(updated),
    );
    expect(mocks.mark).toHaveBeenCalledWith(
      { id: 'bill-a' },
      { id: 'bill-a', expectedVersion: '3', reason: '客户书面同意' },
      expect.objectContaining({
        signal: expect.any(AbortSignal),
        skipErrorHandler: true,
      }),
    );
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
  });

  it('有效免票提供撤销并要求原因，撤销保留历史而非清空原记录', async () => {
    const { actions } = mount(updated);
    const dialog = await open('撤销无需发票');
    fill(dialog, '  改为实际开票  ', '撤销原因');
    save(dialog);
    await waitFor(() => expect(actions.onUpdated).toHaveBeenCalled());
    expect(mocks.revoke).toHaveBeenCalledWith(
      { id: 'bill-a' },
      { id: 'bill-a', expectedVersion: '4', reason: '改为实际开票' },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    expect(screen.getByText('合同约定无需发票')).toBeInTheDocument();
    expect(mocks.mark).not.toHaveBeenCalled();
  });

  it('已有有效发票或账单已取消时不允许标记', async () => {
    const view = mount({
      ...bill,
      feeLockFacts: [FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_INVOICE],
    });
    expect(screen.getByRole('button', { name: '标记无需发票' })).toBeDisabled();
    view.rerender(
      <BillInvoiceExemptionSection
        bill={{
          ...bill,
          status: FinanceBillStatus.FINANCE_BILL_STATUS_CANCELLED,
        }}
        actions={view.actions}
      />,
    );
    expect(screen.getByRole('button', { name: '标记无需发票' })).toBeDisabled();
    await act(async () => {});
  });

  it('提交失败保留原因与打开时版本，父级新数据不能偷换提交基线', async () => {
    mocks.mark.mockResolvedValue(undefined);
    const view = mount();
    const dialog = await open();
    fill(dialog);
    save(dialog);
    await waitFor(() => expect(mocks.mark).toHaveBeenCalledTimes(1));
    expect(dialog.getByLabelText('免票原因')).toHaveValue(' 客户书面同意 ');
    expect(view.actions.onUpdated).not.toHaveBeenCalled();
    view.rerender(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <BillInvoiceExemptionSection
          bill={{ ...bill, version: '8' }}
          actions={view.actions}
        />
      </ConfigProvider>,
    );
    await waitFor(() =>
      expect(dialog.getByRole('button', { name: /保\s*存/ })).toBeEnabled(),
    );
    save(dialog);
    await waitFor(() => expect(mocks.mark).toHaveBeenCalledTimes(2));
    expect(mocks.mark.mock.lastCall?.[1].expectedVersion).toBe('3');
  });

  it('外部变动提示保留原因，刷新需确认后才清除输入和重读', async () => {
    mocks.probe.mockResolvedValue({
      success: true,
      data: { id: bill.id, version: '4' },
    });
    const view = mount();
    const dialog = await open();
    fill(dialog);
    expect(
      await dialog.findByText('这个账单已经发生变动，请重新刷新。'),
    ).toBeInTheDocument();
    expect(dialog.getByLabelText('免票原因')).toHaveValue(' 客户书面同意 ');
    fireEvent.click(dialog.getByRole('button', { name: /刷\s*新/ }));
    expect(
      await screen.findByText('刷新会清除尚未保存的原因，确认继续？'),
    ).toBeInTheDocument();
    expect(view.actions.onRefresh).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: /确\s*定/ }));
    await waitFor(() =>
      expect(view.actions.onRefresh).toHaveBeenCalledTimes(1),
    );
    expect(mocks.mark).not.toHaveBeenCalled();
  });

  it.each([{ version: '4' }, { id: 'bill-a' }])(
    '成功报文缺少账单身份或版本时明确失败并保留输入：%j',
    async (data) => {
      mocks.mark.mockResolvedValue({ success: true, data });
      const view = mount();
      const dialog = await open();
      fill(dialog);
      save(dialog);
      expect(
        await screen.findByText('无需发票操作未返回有效账单，请刷新后核对'),
      ).toBeInTheDocument();
      expect(dialog.getByLabelText('免票原因')).toHaveValue(' 客户书面同意 ');
      expect(view.actions.onUpdated).not.toHaveBeenCalled();
    },
  );

  it('关闭权限或切换账单后，不接收旧提交的迟到响应', async () => {
    let resolve!: (response: API.UpdateBillResponse) => void;
    mocks.mark.mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done;
        }),
    );
    const view = mount();
    const dialog = await open();
    fill(dialog);
    save(dialog);
    await waitFor(() => expect(mocks.mark).toHaveBeenCalledTimes(1));
    view.rerender(
      <BillInvoiceExemptionSection bill={{ ...bill, id: 'bill-b' }} />,
    );
    await act(async () => {
      resolve({ success: true, data: updated });
    });
    expect(view.actions.onUpdated).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
