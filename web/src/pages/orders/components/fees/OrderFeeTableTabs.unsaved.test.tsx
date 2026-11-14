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
  orderFeeServiceSaveOrderFees,
} from '@/services/roncin/orderFeeService';
import { PAYABLE, RECEIVABLE } from './feeConstants';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceSaveOrderFees: vi.fn(),
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
    receivableSummary: {
      count: 0,
      gross: '0',
      net: '0',
      unconvertedCount: 0,
      currencies: [],
    },
    payableSummary: {
      count: 0,
      gross: '0',
      net: '0',
      unconvertedCount: 0,
      currencies: [],
    },
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
    canUpdateFees: true,
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
  vi.mocked(orderFeeServiceSaveOrderFees).mockReset();
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
      createdAt: '2026-10-20',
    })),
  });
});

it('常开编辑：单纯进入不脏，真实输入同步标脏；保存一侧只清该侧，另一侧草稿继续受保护', async () => {
  const props = makeProps();
  const saveFees = vi.mocked(orderFeeServiceSaveOrderFees);
  render(
    <App>
      <OrderFeeTableTabs {...props} />
    </App>,
  );
  const receivableRow = document.querySelector(
    `tr[data-row-key="fee-${RECEIVABLE}"]`,
  ) as HTMLElement | null;
  const payableRow = (await waitFor(() => {
    const row = document.querySelector(`tr[data-row-key="fee-${PAYABLE}"]`);
    expect(row).not.toBeNull();
    return row;
  })) as HTMLElement;
  const receivableRowEl =
    receivableRow ??
    (document.querySelector(
      `tr[data-row-key="fee-${RECEIVABLE}"]`,
    ) as HTMLElement);
  const receivableInput =
    await within(receivableRowEl).findByPlaceholderText('备注（可选）');
  const payableInput =
    await within(payableRow).findByPlaceholderText('备注（可选）');
  // 所有可编辑行进入输入态本身不算修改。
  expect(getAnyDirtyTabKey()).toBeUndefined();
  fireEvent.change(receivableInput, { target: { value: '应收未保存' } });
  expect(getAnyDirtyTabKey()).toBe('/orders/sea-export/order-1/fees');
  fireEvent.change(payableInput, { target: { value: '应付未保存' } });

  const leave = vi.fn();
  const confirmation = vi.fn();
  expect(confirmIfAnyTabDirty(leave, confirmation)).toBe(false);
  confirmation.mock.calls[0][0].onCancel?.();
  expect(leave).not.toHaveBeenCalled();

  // 保存失败：输入保留，仍受保护。
  saveFees.mockRejectedValueOnce(
    Object.assign(new Error('Request failed'), {
      response: {
        status: 400,
        data: { code: 400, reason: 'ORDER_FEE_INVALID', message: '费用不合法' },
      },
    }),
  );
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name: /保存应付费用/ }));
  });
  await waitFor(() => expect(saveFees).toHaveBeenCalledOnce());
  expect(payableInput).toHaveValue('应付未保存');
  expect(getAnyDirtyTabKey()).toBe('/orders/sea-export/order-1/fees');

  // 保存成功：只清理应付侧，应收未保存输入仍在并继续受保护。
  saveFees.mockResolvedValueOnce({
    success: true,
    data: [],
    feeContentVersion: '3',
  } as never);
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name: /保存应付费用/ }));
  });
  await waitFor(() => expect(saveFees).toHaveBeenCalledTimes(2));
  await waitFor(() =>
    expect(listFees.mock.calls.length).toBeGreaterThanOrEqual(3),
  );
  expect(receivableInput).toHaveValue('应收未保存');
  expect(getAnyDirtyTabKey()).toBe('/orders/sea-export/order-1/fees');
  expect(saveFees.mock.calls[1][1]).toMatchObject({
    direction: PAYABLE,
    rows: [expect.objectContaining({ feeId: `fee-${PAYABLE}` })],
  });

  // 再保存应收侧后，整页不再有未保存内容。
  saveFees.mockResolvedValueOnce({
    success: true,
    data: [],
    feeContentVersion: '4',
  } as never);
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name: /保存应收费用/ }));
  });
  await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
});
