import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, expect, it, vi } from 'vitest';
import {
  confirmIfAnyTabDirty,
  getAnyDirtyTabKey,
} from '@/components/layout/tabCloseGuard';
import {
  orderFeeServiceListFees,
  orderFeeServiceResolveFeeExchangeRate,
  orderFeeServiceUpdateFee,
} from '@/services/roncin/orderFeeService';
import { PAYABLE, RECEIVABLE } from './feeConstants';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceAddFee: vi.fn(),
  orderFeeServiceUpdateFee: vi.fn(),
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
  orderFeeServiceBulkUpdateOrderFees: vi.fn(),
  orderFeeServiceBulkRemoveOrderFees: vi.fn(),
  orderFeeServiceListOrderFeeTagOptions: vi.fn(),
  orderFeeServiceBatchAssignOrderFeeTags: vi.fn(),
  orderFeeServiceBatchRemoveOrderFeeTags: vi.fn(),
}));
const listFees = vi.mocked(orderFeeServiceListFees);

function makeProps() {
  return {
    orderId: 'order-1',
    receivableActionRef: { current: undefined },
    payableActionRef: { current: undefined },
    receivableSummary: { totalAmount: 0, count: 0 },
    payableSummary: { totalAmount: 0, count: 0 },
    selectedReceivableFeeIds: [],
    selectedPayableFeeIds: [],
    setSelectedReceivableFeeIds: vi.fn(),
    setSelectedPayableFeeIds: vi.fn(),
    setAllReceivableItems: vi.fn(),
    setAllPayableItems: vi.fn(),
    setReceivableSummary: vi.fn(),
    setPayableSummary: vi.fn(),
    canCreateFinanceBills: false,
    canCreateFees: true,
    canDeleteFees: true,
    feeWritesDisabled: false,
    onOpenBillWorkbench: vi.fn(),
    feeSettings: [{ id: 'setting-of', feeCode: 'OF', nameZh: '海运费' }],
    settlementParties: [{ id: 'party-old', name: '原结算单位' }],
    currencies: [{ code: 'CNY', name: '人民币' }],
    billingUnits: [{ id: 'unit-piao', code: 'PIAO', name: '票' }],
  };
}

beforeEach(() => {
  listFees.mockReset();
  vi.mocked(orderFeeServiceResolveFeeExchangeRate).mockResolvedValue({
    exchangeRate: '1',
    exchangeRateSource: 'SYSTEM',
  });
  window.history.replaceState({}, '', '/orders/sea-export/order-1/fees');
  vi.mocked(orderFeeServiceUpdateFee).mockReset();
  listFees.mockResolvedValue({
    data: [RECEIVABLE, PAYABLE].map((direction) => ({
      id: `fee-${direction}`,
      canEdit: true,
      canBill: true,
      direction,
      version: '1',
      note: direction === RECEIVABLE ? '应收测试费' : '应付测试费',
      feeSettingId: 'setting-of',
      settlementPartyId: 'party-old',
      billingUnitId: 'unit-piao',
      currency: 'CNY',
      quantity: '1',
      unitPrice: '100',
      expenseDate: '2026-10-20',
    })),
  });
});

it('行内真实输入同步保护；单纯进入编辑不脏，取消一侧不清另一侧，失败保留并在成功保存后清理', async () => {
  const props = makeProps();
  render(
    <App>
      <OrderFeeTableTabs {...props} />
    </App>,
  );
  const receivableRow = (await screen.findByText('应收测试费')).closest('tr');
  const payableRow = (await screen.findByText('应付测试费')).closest('tr');
  if (!receivableRow || !payableRow) throw new Error('费用行不存在');
  fireEvent.click(within(receivableRow).getByRole('button', { name: /编辑/ }));
  fireEvent.click(within(payableRow).getByRole('button', { name: /编辑/ }));
  const receivableInput =
    await within(receivableRow).findByPlaceholderText('备注（可选）');
  const payableInput =
    await within(payableRow).findByPlaceholderText('备注（可选）');
  expect(getAnyDirtyTabKey()).toBeUndefined();
  fireEvent.change(receivableInput, { target: { value: '应收未保存' } });
  expect(getAnyDirtyTabKey()).toBe('/orders/sea-export');
  fireEvent.change(payableInput, { target: { value: '应付未保存' } });
  fireEvent.click(within(receivableRow).getByText('取消'));
  await waitFor(() =>
    expect(
      within(receivableRow).queryByPlaceholderText('备注（可选）'),
    ).not.toBeInTheDocument(),
  );
  expect(getAnyDirtyTabKey()).toBe('/orders/sea-export');
  const leave = vi.fn();
  const confirmation = vi.fn();
  expect(confirmIfAnyTabDirty(leave, confirmation)).toBe(false);
  confirmation.mock.calls[0][0].onCancel?.();
  expect(leave).not.toHaveBeenCalled();
  expect(payableInput).toHaveValue('应付未保存');
  vi.mocked(orderFeeServiceUpdateFee).mockResolvedValue(undefined as never);
  await act(async () => {
    fireEvent.click(within(payableRow).getByText('保存'));
  });
  await waitFor(() => expect(orderFeeServiceUpdateFee).toHaveBeenCalledOnce());
  expect(orderFeeServiceUpdateFee).toHaveBeenCalledWith(
    expect.anything(),
    expect.objectContaining({ expenseDate: '2026-10-20' }),
  );
  expect(payableInput).toHaveValue('应付未保存');
  expect(getAnyDirtyTabKey()).toBe('/orders/sea-export');
  vi.mocked(orderFeeServiceUpdateFee).mockResolvedValue({
    data: { canEdit: true, canBill: true, id: 'fee-2' },
  });
  await act(async () => {
    fireEvent.click(within(payableRow).getByText('保存'));
  });
  await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
});
