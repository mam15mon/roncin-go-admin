import type { ActionType } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  type RenderOptions,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { expect, vi } from 'vitest';
import {
  orderFeeServiceBatchAssignOrderFeeTags,
  orderFeeServiceBatchRemoveOrderFeeTags,
  orderFeeServiceBulkRemoveOrderFees,
  orderFeeServiceBulkUpdateOrderFees,
  orderFeeServiceListFees,
  orderFeeServiceListOrderFeeTagOptions,
  orderFeeServiceResolveFeeExchangeRate,
} from '@/services/roncin/orderFeeService';
import { RECEIVABLE } from './feeConstants';
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

export const listFees = vi.mocked(orderFeeServiceListFees);
export const resolveRate = vi.mocked(orderFeeServiceResolveFeeExchangeRate);
export const bulkUpdate = vi.mocked(orderFeeServiceBulkUpdateOrderFees);
export const bulkRemove = vi.mocked(orderFeeServiceBulkRemoveOrderFees);
export const listTagOptions = vi.mocked(orderFeeServiceListOrderFeeTagOptions);
export const batchAssignTags = vi.mocked(
  orderFeeServiceBatchAssignOrderFeeTags,
);
export const batchRemoveTags = vi.mocked(
  orderFeeServiceBatchRemoveOrderFeeTags,
);

/** 模拟 antd Select：点开下拉并选择可见选项。 */
export async function pickSelectOption(combobox: Element, name: string) {
  fireEvent.mouseDown(combobox);
  const option = await waitFor(() => {
    const candidates = screen.getAllByText(name).filter((element) => {
      const dropdown = element.closest('.ant-select-dropdown');
      return (
        dropdown !== null &&
        !dropdown.className.includes('-hidden') &&
        !dropdown.className.includes('ant-slide-up-leave')
      );
    });
    if (candidates.length === 0) {
      throw new Error(`选项 ${name} 未渲染`);
    }
    return candidates[0];
  });
  fireEvent.click(option);
}

/** 模拟在 Dropdown 菜单中点击批量操作项。 */
export async function triggerBulkAction(name: string | RegExp, tableIndex = 0) {
  const bulkButtons = screen.getAllByRole('button', { name: /批量操作/ });
  await act(async () => {
    fireEvent.click(bulkButtons[tableIndex]);
  });
  const menuItem = await screen.findByText(name);
  await act(async () => {
    fireEvent.click(menuItem);
  });
}

export function makeFee(partial: Partial<API.OrderFee>): API.OrderFee {
  return {
    direction: RECEIVABLE,
    currency: 'CNY',
    quantity: '1',
    unitPrice: '100',
    expenseDate: '2026-09-20 10:00',
    feeName: '海运费',
    ...partial,
  } as API.OrderFee;
}

export const unbilledFeeA = makeFee({
  canEdit: true,
  canDelete: true,
  canBill: true,
  id: 'fee-a',
  version: '3',
  feeName: '海运费',
});
export const unbilledFeeB = makeFee({
  canEdit: true,
  canDelete: true,
  canBill: true,
  id: 'fee-b',
  version: '1',
  feeName: '拖车费',
});
export const billedFeeC = makeFee({
  canEdit: false,
  canDelete: false,
  canBill: false,
  id: 'fee-c',
  version: '5',
  hasActiveBill: true,
  feeName: '已建账港杂费',
});

export function makeProps(orderId: string) {
  return {
    props: {
      orderId,
      receivableActionRef: {
        current: undefined,
      } as React.RefObject<ActionType | undefined>,
      payableActionRef: {
        current: undefined,
      } as React.RefObject<ActionType | undefined>,
      receivableSummary: { totalAmount: 0, count: 0 },
      payableSummary: { totalAmount: 0, count: 0 },
      selectedReceivableFeeIds: [] as React.Key[],
      setSelectedReceivableFeeIds: vi.fn(),
      selectedPayableFeeIds: [] as React.Key[],
      setSelectedPayableFeeIds: vi.fn(),
      setAllReceivableItems: vi.fn(),
      setAllPayableItems: vi.fn(),
      setReceivableSummary: vi.fn(),
      setPayableSummary: vi.fn(),
      canCreateFinanceBills: true,
      canCreateFees: true,
      canDeleteFees: true,
      feeWritesDisabled: false,
      onOpenBillWorkbench: vi.fn(),
      onFeeSaved: vi.fn(),
      getTableColumns: undefined,
      feeSettings: [{ id: 'setting-of', feeCode: 'OF', nameZh: '海运费' }],
      settlementParties: [
        { id: 'party-old', name: '原结算单位', code: 'OLD' },
        { id: 'party-new', name: '新结算单位', code: 'NEW' },
      ],
      currencies: [{ code: 'CNY', name: '人民币' }],
      billingUnits: [{ id: 'unit-piao', code: 'PIAO', name: '票' }],
    },
  };
}

export async function renderTabs(
  props: ReturnType<typeof makeProps>['props'],
  options?: RenderOptions,
) {
  render(
    <App>
      <OrderFeeTableTabs {...props} />
    </App>,
    options,
  );
  await waitFor(() => expect(listFees).toHaveBeenCalled());
}

export function getDialog() {
  return screen.getByRole('dialog');
}

/** 每个拆分套件使用独立服务替身，场景之间清空请求和提交结果。 */
export function resetBulkActionMocks() {
  listFees.mockReset();
  listFees.mockResolvedValue({ items: [] } as Awaited<
    ReturnType<typeof listFees>
  >);
  resolveRate.mockReset();
  resolveRate.mockResolvedValue({
    exchangeRate: '1.0000',
    exchangeRateSource: 'SYSTEM',
  } as Awaited<ReturnType<typeof resolveRate>>);
  bulkUpdate.mockReset();
  bulkUpdate.mockResolvedValue({ updatedCount: 0 } as Awaited<
    ReturnType<typeof bulkUpdate>
  >);
  bulkRemove.mockReset();
  bulkRemove.mockResolvedValue({ removedCount: 0 } as Awaited<
    ReturnType<typeof bulkRemove>
  >);
  listTagOptions.mockReset();
  listTagOptions.mockResolvedValue({
    tags: [
      {
        id: 'tag-1',
        name: '重点单',
        groupName: '运营',
        enabled: true,
      },
    ],
  } as Awaited<ReturnType<typeof listTagOptions>>);
  batchAssignTags.mockReset();
  batchAssignTags.mockResolvedValue({ assignedCount: 0 } as Awaited<
    ReturnType<typeof batchAssignTags>
  >);
  batchRemoveTags.mockReset();
  batchRemoveTags.mockResolvedValue({ removedCount: 0 } as Awaited<
    ReturnType<typeof batchRemoveTags>
  >);
}
