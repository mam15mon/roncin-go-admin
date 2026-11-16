import type { ActionType } from '@ant-design/pro-components';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { expect, it, vi } from 'vitest';
import { orderFeeServiceListFees } from '@/services/roncin/orderFeeService';
import { PAYABLE } from './feeConstants';
import { integerQuantityMessage } from './feeQuantityRule';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

const emptySummary = {
  count: 0,
  gross: '0',
  net: '0',
  unconvertedCount: 0,
  currencies: [],
};

it('整数计费单位输入小数数量时立即显示错误，且不会被字段规则覆盖', async () => {
  vi.mocked(orderFeeServiceListFees).mockResolvedValue({
    data: [
      {
        id: 'pay-1',
        direction: PAYABLE,
        version: '1',
        canEdit: true,
        feeCode: 'CUSTOMS',
        currency: 'CNY',
        billingUnitId: 'unit-bl',
        billingUnit: '票',
        quantity: '1.0000',
        unitPrice: '123.0000',
        totalAmount: '123',
        baseCurrencyAmount: '123',
      },
    ],
  } as Awaited<ReturnType<typeof orderFeeServiceListFees>>);
  const ref = () =>
    ({ current: undefined }) as React.RefObject<ActionType | undefined>;

  const { container } = render(
    <App>
      <OrderFeeTableTabs
        orderId="order-1"
        receivableActionRef={ref()}
        payableActionRef={ref()}
        receivableSummary={emptySummary}
        payableSummary={emptySummary}
        selectedReceivableFeeIds={[]}
        setSelectedReceivableFeeIds={vi.fn()}
        selectedPayableFeeIds={[]}
        setSelectedPayableFeeIds={vi.fn()}
        setAllReceivableItems={vi.fn()}
        setAllPayableItems={vi.fn()}
        setReceivableSummary={vi.fn()}
        setPayableSummary={vi.fn()}
        canCreateFinanceBills
        canCreateFees
        canUpdateFees
        canDeleteFees
        feeWritesDisabled={false}
        onOpenBillWorkbench={vi.fn()}
        billingUnits={[
          { id: 'unit-bl', name: '票', quantityMustBeInteger: true },
          { id: 'unit-kg', name: '公斤', quantityMustBeInteger: false },
        ]}
      />
    </App>,
  );

  const quantityInput = await waitFor(() => {
    const input = Array.from(
      container.querySelectorAll<HTMLInputElement>(
        '[data-row-key="pay-1"] input:not([type=checkbox])',
      ),
    ).find((item) => item.value === '1');
    expect(input).toBeTruthy();
    return input as HTMLInputElement;
  });

  const hasError = () =>
    Boolean(quantityInput.closest('.ant-form-item-has-error'));

  fireEvent.change(quantityInput, { target: { value: '1.5' } });
  await waitFor(() => expect(hasError()).toBe(true));
  // 等待字段异步规则全部结束后，整数错误仍保留。
  await new Promise((resolve) => setTimeout(resolve, 200));
  expect(hasError()).toBe(true);
  // 行内错误放在气泡中，点击输入框可查看具体原因。
  fireEvent.click(quantityInput);
  expect(await screen.findByText(integerQuantityMessage)).toBeInTheDocument();

  fireEvent.change(quantityInput, { target: { value: '2' } });
  await waitFor(() => expect(hasError()).toBe(false));
});
