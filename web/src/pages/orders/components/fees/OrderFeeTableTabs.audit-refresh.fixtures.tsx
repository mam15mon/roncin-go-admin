import { render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import { expect } from 'vitest';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';
import { PAYABLE, RECEIVABLE } from './feeConstants';
import {
  listFees,
  makeFee,
  makeProps,
  resetBulkActionMocks,
  saveFees,
} from './OrderFeeTableTabs.bulkActions.fixtures';

const rowOf = (id: string) =>
  document.querySelector(`tr[data-row-key="${id}"]`) as HTMLElement;
export const auditButton = (label: string) => {
  const button = screen.getByText(label, { exact: true }).closest('button');
  expect(button).not.toBeNull();
  return button as HTMLButtonElement;
};
export const noteOf = (id: string) =>
  within(rowOf(id)).getByPlaceholderText('备注（可选）');

export async function mountForAuditRefresh() {
  const { default: OrderFeeTableTabs } = await import('./OrderFeeTableTabs');
  listFees.mockResolvedValue({
    data: [RECEIVABLE, PAYABLE].map((direction) =>
      makeFee({
        id: `fee-${direction}`,
        direction,
        version: '2',
        canEdit: true,
        canDelete: true,
        canBill: true,
        feeSettingId: 'setting-of',
        settlementPartyId: 'party-old',
        billingUnitId: 'unit-piao',
        note: direction === RECEIVABLE ? '原应收' : '原应付',
      }),
    ),
  } as never);
  saveFees.mockResolvedValue({
    success: true,
    data: [],
    feeContentVersion: '7',
  } as never);
  const { props } = makeProps('order-1');
  const view = (refreshing: boolean) => (
    <App>
      <OrderFeeTableTabs {...props} feeWritesDisabled={refreshing} />
    </App>
  );
  const { rerender } = render(view(false));
  await waitFor(() => expect(rowOf(`fee-${PAYABLE}`)).not.toBeNull());
  await waitFor(() => expect(noteOf(`fee-${RECEIVABLE}`)).toBeVisible());
  return {
    props,
    synchronize: async () => {
      rerender(view(true));
      await waitFor(() => expect(auditButton('保存应付费用')).toBeDisabled());
      expect(getAnyDirtyTabKey()).toBe('/orders/sea-export/order-1/fees');
      rerender(view(false));
      await waitFor(() => expect(auditButton('保存应付费用')).toBeEnabled());
    },
  };
}

export function resetAuditRefresh() {
  resetBulkActionMocks();
  window.history.replaceState({}, '', '/orders/sea-export/order-1/fees');
}
