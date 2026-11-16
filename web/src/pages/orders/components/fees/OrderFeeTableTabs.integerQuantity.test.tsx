import type { ActionType } from '@ant-design/pro-components';
import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import { orderFeeServiceListFees } from '@/services/roncin/orderFeeService';
import { PAYABLE } from './feeConstants';
import { integerQuantityMessage } from './feeQuantityRule';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

const listFees = vi.mocked(orderFeeServiceListFees);

const emptySummary = {
  count: 0,
  gross: '0',
  net: '0',
  unconvertedCount: 0,
  currencies: [],
};

const billingUnits = [
  { id: 'unit-bl', name: '票', quantityMustBeInteger: true },
  { id: 'unit-kg', name: '公斤', quantityMustBeInteger: false },
];

function renderTabs(fee: Partial<API.OrderFee>) {
  listFees.mockResolvedValue({
    data: [
      {
        id: 'pay-1',
        direction: PAYABLE,
        version: '1',
        canEdit: true,
        feeCode: 'CUSTOMS',
        currency: 'CNY',
        unitPrice: '123.0000',
        totalAmount: '123',
        baseCurrencyAmount: '123',
        ...fee,
      },
    ],
  } as Awaited<ReturnType<typeof listFees>>);
  const ref = () =>
    ({ current: undefined }) as React.RefObject<ActionType | undefined>;
  return render(
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
        billingUnits={billingUnits}
      />
    </App>,
  );
}

/** 按当前值定位数量输入框（数量列是该行唯一取该值的文本框）。 */
async function findQuantityInput(container: HTMLElement, value: string) {
  return waitFor(() => {
    const input = Array.from(
      container.querySelectorAll<HTMLInputElement>(
        '[data-row-key="pay-1"] input:not([type=checkbox])',
      ),
    ).find((item) => item.value === value);
    expect(input).toBeTruthy();
    return input as HTMLInputElement;
  });
}

const hasError = (input: HTMLInputElement) =>
  Boolean(input.closest('.ant-form-item-has-error'));

beforeEach(() => {
  listFees.mockReset();
});

it('整数计费单位不接受小数点：键入的点不出现，粘贴的小数在点处截断', async () => {
  const { container } = renderTabs({
    billingUnitId: 'unit-bl',
    billingUnit: '票',
    quantity: '1.0000',
  });
  const quantityInput = await findQuantityInput(container, '1');

  fireEvent.change(quantityInput, { target: { value: '1.' } });
  await waitFor(() => expect(quantityInput.value).toBe('1'));
  fireEvent.change(quantityInput, { target: { value: '1.5' } });
  await waitFor(() => expect(quantityInput.value).toBe('1'));
  fireEvent.change(quantityInput, { target: { value: '2' } });
  await waitFor(() => expect(quantityInput.value).toBe('2'));
  expect(hasError(quantityInput)).toBe(false);
});

it('非整数计费单位照常接受小数', async () => {
  const { container } = renderTabs({
    billingUnitId: 'unit-kg',
    billingUnit: '公斤',
    quantity: '1.0000',
  });
  const quantityInput = await findQuantityInput(container, '1');

  fireEvent.change(quantityInput, { target: { value: '1.5' } });
  await waitFor(() => expect(quantityInput.value).toBe('1.5'));
  expect(hasError(quantityInput)).toBe(false);
});

it('先填小数再切到整数计费单位时，错误直接显示在数量输入框下方', async () => {
  const { container } = renderTabs({
    billingUnitId: 'unit-kg',
    billingUnit: '公斤',
    quantity: '1.5000',
  });
  const quantityInput = await findQuantityInput(container, '1.5');
  const unitSelect = Array.from(
    container.querySelectorAll<HTMLElement>(
      '[data-row-key="pay-1"] .ant-select',
    ),
  ).find((select) => select.textContent?.includes('公斤'));
  expect(unitSelect).toBeTruthy();

  fireEvent.mouseDown(
    unitSelect?.querySelector('.ant-select-selector') ??
      (unitSelect as HTMLElement),
  );
  const option = await waitFor(() => {
    const candidate = screen
      .getAllByText('票')
      .find((element) => element.closest('.ant-select-dropdown'));
    expect(candidate).toBeTruthy();
    return candidate as HTMLElement;
  });
  fireEvent.click(option);

  await waitFor(() => expect(hasError(quantityInput)).toBe(true));
  // 错误文字在输入框所在单元格内直接展示，无需点击弹出气泡。
  expect(quantityInput.closest('td')?.textContent).toContain(
    integerQuantityMessage,
  );
});
