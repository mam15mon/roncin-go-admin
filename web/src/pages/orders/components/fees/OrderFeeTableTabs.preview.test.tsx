import type { ActionType } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  orderFeeServiceListFees,
  orderFeeServiceResolveFeeExchangeRate,
} from '@/services/roncin/orderFeeService';
import { RECEIVABLE } from './feeConstants';
import OrderFeeTableTabs from './OrderFeeTableTabs';

vi.mock('@/services/roncin/orderFeeService', () => ({
  orderFeeServiceListFees: vi.fn(),
  orderFeeServiceResolveFeeExchangeRate: vi.fn(),
}));

const listFees = vi.mocked(orderFeeServiceListFees);
const resolveRate = vi.mocked(orderFeeServiceResolveFeeExchangeRate);

/** 模拟 antd Select：点开下拉并选择可见选项。 */
async function pickSelectOption(combobox: Element, name: string) {
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

function makeProps(orderId: string) {
  return {
    orderId,
    receivableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    payableActionRef: {
      current: undefined,
    } as React.RefObject<ActionType | undefined>,
    receivableSummary: { totalAmount: 100, count: 1 },
    payableSummary: { totalAmount: 0, count: 0 },
    selectedReceivableFeeIds: ['fee-1'],
    setSelectedReceivableFeeIds: vi.fn(),
    selectedPayableFeeIds: [],
    setSelectedPayableFeeIds: vi.fn(),
    setAllReceivableItems: vi.fn(),
    setAllPayableItems: vi.fn(),
    setReceivableSummary: vi.fn(),
    setPayableSummary: vi.fn(),
    canCreateFinanceBills: true,
    feeWritesDisabled: true,
    onOpenBillWorkbench: vi.fn(),
    onOpenFeeModal: vi.fn(),
    getTableColumns: () => [{ dataIndex: 'id', title: '费用 ID' }],
  };
}

describe('OrderFeeTableTabs 计算列实时预览', () => {
  beforeEach(() => {
    listFees.mockReset();
    listFees.mockResolvedValue({ items: [] } as Awaited<
      ReturnType<typeof listFees>
    >);
    resolveRate.mockReset();
    resolveRate.mockResolvedValue({
      exchangeRate: '1.0000',
      exchangeRateSource: 'SYSTEM',
    } as Awaited<ReturnType<typeof resolveRate>>);
  });

  it('行内编辑输入单价与数量后，总金额列实时预览计算结果', async () => {
    const today = '2026-09-21';
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-unbilled-3',
          direction: RECEIVABLE,
          currency: 'CNY',
          quantity: '1',
          expenseDate: today,
          version: '2',
          feeSettingId: 'setting-of',
          settlementPartyId: 'customer-1',
          billingUnitId: 'unit-piao',
        } as API.OrderFee,
      ],
    } as any);

    const props = {
      ...makeProps('order-1'),
      selectedReceivableFeeIds: [],
      feeWritesDisabled: false,
      getTableColumns: undefined,
      billingUnits: [{ id: 'unit-piao', code: 'PIAO', name: '票' }],
      feeSettings: [{ id: 'setting-of', feeCode: 'OF', nameZh: '海运费' }],
      settlementParties: [{ id: 'customer-1', name: '测试客户' }],
      currencies: [{ code: 'CNY', name: '人民币' }],
    };

    render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );

    await screen.findByText(today);
    await act(async () => {
      screen.getByRole('button', { name: /编\s*辑/ }).click();
    });

    const unitPriceInput = await waitFor(() => {
      const input = screen.getByPlaceholderText('0.00');
      expect(input).toBeInTheDocument();
      return input;
    });
    await act(async () => {
      fireEvent.change(unitPriceInput, { target: { value: '25.5' } });
    });

    await waitFor(() => {
      expect(screen.getByText('25.50 CNY')).toBeInTheDocument();
    });

    const quantityInput = screen.getByDisplayValue('1');
    await act(async () => {
      fireEvent.change(quantityInput, { target: { value: '3' } });
    });

    await waitFor(() => {
      expect(screen.getByText('76.50 CNY')).toBeInTheDocument();
    });
  });

  it('已有费用修改单价、数量或费用项目后实时重算，清空输入不保留旧金额', async () => {
    const today = '2026-09-21';
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-unbilled-4',
          direction: RECEIVABLE,
          currency: 'CNY',
          quantity: '1',
          unitPrice: '123',
          totalAmount: '123',
          taxInclusive: true,
          taxRate: '0',
          taxAmount: '0',
          netAmount: '123',
          baseCurrencyAmount: '123',
          exchangeRate: '1',
          expenseDate: today,
          version: '2',
          feeSettingId: 'setting-of',
          settlementPartyId: 'customer-1',
          billingUnitId: 'unit-piao',
        } as API.OrderFee,
      ],
    } as any);

    const props = {
      ...makeProps('order-1'),
      selectedReceivableFeeIds: [],
      feeWritesDisabled: false,
      getTableColumns: undefined,
      billingUnits: [{ id: 'unit-piao', code: 'PIAO', name: '票' }],
      feeSettings: [
        { id: 'setting-of', feeCode: 'OF', nameZh: '海运费', taxRate: '0.00' },
        {
          id: 'setting-thc',
          feeCode: 'THC',
          nameZh: '码头操作费',
          taxRate: '6.00',
        },
      ],
      settlementParties: [{ id: 'customer-1', name: '测试客户' }],
      currencies: [{ code: 'CNY', name: '人民币' }],
    };

    render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );

    await screen.findByText(today);
    await act(async () => {
      screen.getByRole('button', { name: /编\s*辑/ }).click();
    });

    // 不重新选择费用项目，直接改价也应读取当前费用项目税率。
    const priceInput = screen.getByPlaceholderText('0.00');
    await act(async () => {
      fireEvent.change(priceInput, { target: { value: '106' } });
    });
    await waitFor(() => {
      expect(screen.getAllByText('106.00').length).toBe(3);
      expect(screen.queryByText('123.00')).not.toBeInTheDocument();
    });

    const comboboxes = await waitFor(() => {
      const boxes = screen.getAllByRole('combobox');
      expect(boxes.length).toBeGreaterThanOrEqual(4);
      return boxes;
    });
    await pickSelectOption(comboboxes[0], '码头操作费 (THC)');

    await waitFor(() => {
      expect(screen.getAllByText('THC').length).toBeGreaterThan(0);
      expect(screen.getAllByText('6%').length).toBeGreaterThan(0);
    });

    // 含税 106、税率 6%：不含税总额与不含税单价均为 100.00，税金 6.00
    await waitFor(() => {
      expect(screen.getAllByText('100.00').length).toBe(2);
      expect(screen.getByText('6.00')).toBeInTheDocument();
    });

    await act(async () => {
      fireEvent.change(screen.getByDisplayValue('1'), {
        target: { value: '2' },
      });
    });
    await waitFor(() => {
      expect(screen.getByText('200.00')).toBeInTheDocument();
      expect(screen.getByText('100.00')).toBeInTheDocument();
      expect(screen.getByText('12.00')).toBeInTheDocument();
    });

    await act(async () => {
      fireEvent.change(priceInput, { target: { value: '' } });
    });
    await waitFor(() => {
      for (const value of [
        '100.00',
        '200.00',
        '12.00',
        '123.00',
        '212.00 CNY',
      ]) {
        expect(screen.queryByText(value)).not.toBeInTheDocument();
      }
    });
    expect(screen.queryByText('待保存')).not.toBeInTheDocument();
  });

  it.each(['应收', '应付'])(
    '行内新增%s费用：输入即折算，缺少输入显示占位且无需保存',
    async (direction) => {
      listFees.mockResolvedValue({ data: [] } as any);

      const props = {
        ...makeProps('order-1'),
        selectedReceivableFeeIds: [],
        feeWritesDisabled: false,
        getTableColumns: undefined,
        billingUnits: [{ id: 'unit-piao', code: 'PIAO', name: '票' }],
        feeSettings: [
          {
            id: 'setting-thc',
            feeCode: 'THC',
            nameZh: '码头操作费',
            taxRate: '6.00',
          },
        ],
        settlementParties: [{ id: 'customer-1', name: '测试客户' }],
        currencies: [{ code: 'CNY', name: '人民币' }],
      };

      render(
        <App>
          <OrderFeeTableTabs {...props} />
        </App>,
      );

      await act(async () => {
        screen
          .getByRole('button', { name: new RegExp(`新增${direction}费用`) })
          .click();
      });
      await waitFor(() => expect(screen.getByText('保存')).toBeInTheDocument());

      // 未输入时显示空值占位，计算结果不依赖保存。
      expect(screen.queryByText('待保存')).not.toBeInTheDocument();

      const comboboxes = await waitFor(() => {
        const boxes = screen.getAllByRole('combobox');
        expect(boxes.length).toBeGreaterThanOrEqual(4);
        return boxes;
      });
      await pickSelectOption(comboboxes[0], '码头操作费 (THC)');

      // 仅选择费用项目即显示税率，金额继续等待单价输入。
      await waitFor(() => {
        expect(screen.getAllByText('6%').length).toBeGreaterThan(0);
      });
      expect(screen.queryByText('待保存')).not.toBeInTheDocument();

      const unitPriceInput = screen.getByPlaceholderText('0.00');
      await act(async () => {
        fireEvent.change(unitPriceInput, { target: { value: '106' } });
      });

      // 含税单价 106、税率 6%：不含税单价 100.00，与不含税总额预览同值
      await waitFor(() => {
        expect(screen.getAllByText('100.00').length).toBe(2);
        expect(screen.getByText('6.00')).toBeInTheDocument();
      });
      expect(screen.queryByText('待保存')).not.toBeInTheDocument();
      expect(screen.getByText('保存')).toBeInTheDocument();
      await act(async () => {
        fireEvent.change(unitPriceInput, { target: { value: '' } });
      });
      await waitFor(() =>
        expect(screen.queryByText('100.00')).not.toBeInTheDocument(),
      );
      await act(async () => {
        fireEvent.change(unitPriceInput, { target: { value: '212' } });
      });
      await waitFor(() => {
        expect(screen.getAllByText('200.00').length).toBe(2);
        expect(screen.getByText('12.00')).toBeInTheDocument();
      });
    },
  );
});
