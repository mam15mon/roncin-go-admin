import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import FeeCorrectionCreate from './FeeCorrectionCreate';

const api = vi.hoisted(() => ({ create: vi.fn() }));
vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceCreateOrderFeeCorrection: api.create,
}));
vi.mock('./FeeSupplementModal', () => ({
  default: ({ initialRequest, initialDirection, onSubmit }: any) => (
    <button
      type="button"
      onClick={() =>
        onSubmit({
          ...initialRequest,
          direction: initialDirection,
          unitPrice: '800',
          reason: '客户优惠',
        })
      }
    >
      提交目标费用
    </button>
  ),
}));
const source: API.OrderFee = {
  id: 'fee-1',
  version: '7',
  direction: 1,
  feeSettingId: 'setting-1',
  feeName: '约定运费',
  settlementPartyId: 'party-1',
  settlementPartyName: '客户甲',
  billingUnitId: 'unit-1',
  currency: 'CNY',
  quantity: '1',
  unitPrice: '1000',
  totalAmount: '1000',
  taxInclusive: true,
  taxRate: '6',
  expenseDate: '2026-10-10',
};
const props = {
  orderId: 'o1',
  fees: [source],
  canCreate: true,
  canUpdate: true,
  disabled: false,
  onCreated: vi.fn(),
  feeSettings: [],
  settlementParties: [],
  currencies: [],
  billingUnits: [],
};
beforeEach(() => {
  vi.clearAllMocks();
  api.create.mockResolvedValue({ success: true });
});
describe('费用更正申请入口', () => {
  it('替代申请绑定原费用版本并保留原费名税率', async () => {
    renderWithApp(<FeeCorrectionCreate {...props} canCreate={false} />);
    fireEvent.click(screen.getByRole('button', { name: '申请费用更正' }));
    fireEvent.mouseDown(await screen.findByLabelText('原费用'));
    fireEvent.click(await screen.findByText(/应收 约定运费/));
    fireEvent.click(screen.getByRole('button', { name: /确.*定/ }));
    fireEvent.click(
      await screen.findByRole('button', { name: '提交目标费用' }),
    );
    await waitFor(() => expect(api.create).toHaveBeenCalledOnce());
    expect(api.create.mock.calls[0]).toEqual([
      { orderId: 'o1' },
      expect.objectContaining({
        operation: 2,
        sourceFeeId: 'fee-1',
        sourceFeeExpectedVersion: '7',
        reason: '客户优惠',
        idempotencyKey: expect.any(String),
        targetFee: expect.objectContaining({
          direction: 1,
          unitPrice: '800',
          feeNameOverride: '约定运费',
          taxRateOverride: '6',
          taxInclusive: true,
        }),
      }),
    ]);
    await waitFor(() => expect(props.onCreated).toHaveBeenCalledOnce());
    await act(async () => {});
  });
  it('作废仅提交原费用与原因，不伪造零金额目标', async () => {
    renderWithApp(<FeeCorrectionCreate {...props} canCreate={false} />);
    fireEvent.click(screen.getByRole('button', { name: '申请费用更正' }));
    fireEvent.mouseDown(await screen.findByLabelText('更正方式'));
    fireEvent.click(await screen.findByText('整笔作废'));
    fireEvent.mouseDown(screen.getByLabelText('原费用'));
    fireEvent.click(await screen.findByText(/应收 约定运费/));
    fireEvent.change(screen.getByLabelText('作废原因'), {
      target: { value: '重复费用' },
    });
    fireEvent.click(screen.getByRole('button', { name: /确.*定/ }));
    await waitFor(() => expect(api.create).toHaveBeenCalledOnce());
    expect(api.create.mock.calls[0][1]).toMatchObject({
      operation: 3,
      sourceFeeId: 'fee-1',
      reason: '重复费用',
    });
    expect(api.create.mock.calls[0][1].targetFee).toBeUndefined();
    await act(async () => {});
  });
  it('待审批期间禁止另建更正，没有录入权限隐藏入口', () => {
    const view = renderWithApp(<FeeCorrectionCreate {...props} disabled />);
    expect(screen.getByRole('button', { name: '申请费用更正' })).toBeDisabled();
    view.rerender(
      <FeeCorrectionCreate {...props} canCreate={false} canUpdate={false} />,
    );
    expect(
      screen.queryByRole('button', { name: '申请费用更正' }),
    ).not.toBeInTheDocument();
  });
});
