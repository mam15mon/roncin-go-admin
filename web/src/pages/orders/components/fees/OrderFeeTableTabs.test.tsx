import type { ActionType } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import type React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  orderFeeServiceListFees,
  orderFeeServiceResolveFeeExchangeRate,
} from '@/services/roncin/orderFeeService';
import { FEE_UNBILLED, PAYABLE, RECEIVABLE } from './feeConstants';
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

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
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

describe('OrderFeeTableTabs 业务锁策略', () => {
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

  it('业务费用只读时禁用新增，但保留费用勾选的账单创建入口', async () => {
    const props = makeProps('order-1');

    render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );

    await waitFor(() => expect(listFees).toHaveBeenCalled());
    expect(
      screen.getByRole('button', { name: /生成账单（1）/ }),
    ).not.toBeDisabled();
    expect(screen.getByRole('button', { name: /新增应收费用/ })).toBeDisabled();
    expect(screen.getByRole('button', { name: /新增应付费用/ })).toBeDisabled();
  });

  it('订单 A 的费用响应晚于订单 B 时，不得覆盖 B 的费用与汇总', async () => {
    const responseA = deferred<any>();
    const responseB = deferred<any>();
    listFees.mockImplementation(({ orderId }) => {
      return orderId === 'order-A' ? responseA.promise : responseB.promise;
    });

    const props = makeProps('order-A');
    const { rerender } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );

    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-A' }),
    );

    const propsB = {
      ...props,
      orderId: 'order-B',
      selectedReceivableFeeIds: [],
    };
    rerender(
      <App>
        <OrderFeeTableTabs {...propsB} />
      </App>,
    );

    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-B' }),
    );

    const feeB = {
      id: 'fee-B',
      direction: RECEIVABLE,
      status: FEE_UNBILLED,
      baseCurrencyAmount: '200',
    } as API.OrderFee;
    responseB.resolve({ data: [feeB] });

    await waitFor(() =>
      expect(props.setAllReceivableItems).toHaveBeenLastCalledWith([feeB]),
    );
    expect(props.setReceivableSummary).toHaveBeenLastCalledWith({
      totalAmount: 200,
      count: 1,
    });

    const feeA = {
      id: 'fee-A',
      direction: RECEIVABLE,
      status: FEE_UNBILLED,
      baseCurrencyAmount: '999',
    } as API.OrderFee;
    await act(async () => {
      responseA.resolve({ data: [feeA] });
    });

    expect(props.setAllReceivableItems).toHaveBeenLastCalledWith([feeB]);
    expect(props.setReceivableSummary).toHaveBeenLastCalledWith({
      totalAmount: 200,
      count: 1,
    });
    expect(props.setAllReceivableItems).not.toHaveBeenCalledWith([feeA]);
  });

  it('订单 A 已展示后切换到 B 时，立即移除 A 的表格行且 B 失败也不恢复', async () => {
    const responseB = deferred<any>();
    const feeA = {
      id: 'visible-fee-A',
      direction: RECEIVABLE,
      status: FEE_UNBILLED,
      baseCurrencyAmount: '100',
    } as API.OrderFee;
    listFees
      .mockResolvedValueOnce({ data: [feeA] } as any)
      .mockImplementationOnce(() => responseB.promise);

    const props = makeProps('order-A');
    const { rerender } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );

    await waitFor(() =>
      expect(screen.getByText('visible-fee-A')).toBeInTheDocument(),
    );

    rerender(
      <App>
        <OrderFeeTableTabs {...props} orderId="order-B" />
      </App>,
    );

    expect(screen.queryByText('visible-fee-A')).not.toBeInTheDocument();
    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-B' }),
    );

    await act(async () => {
      responseB.reject(new Error('B 费用加载失败'));
    });

    expect(screen.queryByText('visible-fee-A')).not.toBeInTheDocument();
  });

  it('订单 A 的迟到异常不得破坏已经展示的订单 B 数据', async () => {
    const responseA = deferred<any>();
    const responseB = deferred<any>();
    listFees.mockImplementation(({ orderId }) =>
      orderId === 'order-A' ? responseA.promise : responseB.promise,
    );

    const props = makeProps('order-A');
    const { rerender } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-A' }),
    );

    rerender(
      <App>
        <OrderFeeTableTabs {...props} orderId="order-B" />
      </App>,
    );
    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-B' }),
    );

    const feeB = {
      id: 'visible-fee-B',
      direction: RECEIVABLE,
      status: FEE_UNBILLED,
      baseCurrencyAmount: '200',
    } as API.OrderFee;
    await act(async () => {
      responseB.resolve({ data: [feeB] });
    });
    await waitFor(() =>
      expect(screen.getByText('visible-fee-B')).toBeInTheDocument(),
    );

    await act(async () => {
      responseA.reject(new Error('A 迟到失败'));
    });

    expect(screen.getByText('visible-fee-B')).toBeInTheDocument();
    expect(props.setAllReceivableItems).toHaveBeenLastCalledWith([feeB]);
  });

  it('组件卸载后迟到响应不得回写父组件状态', async () => {
    const response = deferred<any>();
    listFees.mockReturnValue(response.promise);
    const props = makeProps('order-A');
    const { unmount } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );
    await waitFor(() => expect(listFees).toHaveBeenCalledTimes(2));
    unmount();

    await act(async () => {
      response.resolve({
        data: [
          {
            id: 'fee-after-unmount',
            direction: RECEIVABLE,
            status: FEE_UNBILLED,
          },
        ],
      });
    });

    expect(props.setAllReceivableItems).not.toHaveBeenCalled();
    expect(props.setReceivableSummary).not.toHaveBeenCalled();
  });
  it('订单 A 的应付响应晚于订单 B 时，不得覆盖 B 的应付费用与汇总', async () => {
    const responseA = deferred<any>();
    const responseB = deferred<any>();
    listFees.mockImplementation(({ orderId }) => {
      return orderId === 'order-A' ? responseA.promise : responseB.promise;
    });

    const props = makeProps('order-A');
    const { rerender } = render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );

    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-A' }),
    );

    const propsB = {
      ...props,
      orderId: 'order-B',
      selectedReceivableFeeIds: [],
    };
    rerender(
      <App>
        <OrderFeeTableTabs {...propsB} />
      </App>,
    );

    await waitFor(() =>
      expect(listFees).toHaveBeenCalledWith({ orderId: 'order-B' }),
    );

    const feeB = {
      id: 'payable-B',
      direction: PAYABLE,
      status: FEE_UNBILLED,
      baseCurrencyAmount: '80',
    } as API.OrderFee;
    responseB.resolve({ data: [feeB] });

    await waitFor(() =>
      expect(props.setAllPayableItems).toHaveBeenLastCalledWith([feeB]),
    );
    expect(props.setPayableSummary).toHaveBeenLastCalledWith({
      totalAmount: 80,
      count: 1,
    });

    const feeA = {
      id: 'payable-A',
      direction: PAYABLE,
      status: FEE_UNBILLED,
      baseCurrencyAmount: '666',
    } as API.OrderFee;
    await act(async () => {
      responseA.resolve({ data: [feeA] });
    });

    expect(props.setAllPayableItems).toHaveBeenLastCalledWith([feeB]);
    expect(props.setPayableSummary).toHaveBeenLastCalledWith({
      totalAmount: 80,
      count: 1,
    });
    expect(props.setAllPayableItems).not.toHaveBeenCalledWith([feeA]);
  });

  it('点击新增费用时在表格顶部插入可编辑新行，并默认填入 CNY、数量 1 及默认计费单位“票”', async () => {
    const props = {
      ...makeProps('order-1'),
      feeWritesDisabled: false,
      getTableColumns: undefined, // 使用内部完整可编辑列
      billingUnits: [
        { id: 'unit-box', code: 'BOX', name: '箱' },
        { id: 'unit-piao', code: 'PIAO', name: '票' },
      ],
      feeSettings: [{ id: 'setting-of', feeCode: 'OF', nameZh: '海运费' }],
      settlementParties: [
        { id: 'customer-1', name: '测试客户' },
        { id: 'agent-1', name: '测试订舱代理' },
      ],
      currencies: [{ code: 'CNY', name: '人民币' }],
      order: {
        id: 'order-1',
        customerId: 'customer-1',
        bookingAgentId: 'agent-1',
      } as API.Order,
      customerName: '测试客户',
    };

    render(
      <App>
        <OrderFeeTableTabs {...props} />
      </App>,
    );

    await waitFor(() => expect(listFees).toHaveBeenCalled());

    const addReceivableBtn = screen.getByRole('button', {
      name: /新增应收费用/,
    });
    expect(addReceivableBtn).not.toBeDisabled();

    await act(async () => {
      addReceivableBtn.click();
    });

    await waitFor(() => {
      expect(screen.getAllByText('CNY').length).toBeGreaterThan(0);
      expect(screen.getByDisplayValue('1')).toBeInTheDocument();
      expect(screen.getAllByText('票').length).toBeGreaterThan(0);
      expect(screen.getAllByText('测试客户').length).toBeGreaterThan(0);
      expect(screen.getByText('保存')).toBeInTheDocument();
      expect(screen.getByText('取消')).toBeInTheDocument();
    });

    // 新增行默认 CNY + 当天发生日期，应立即解析参考汇率
    await waitFor(() =>
      expect(resolveRate).toHaveBeenCalledWith(
        expect.objectContaining({
          orderId: 'order-1',
          direction: RECEIVABLE,
          currency: 'CNY',
        }),
        expect.anything(),
      ),
    );
    await waitFor(() => expect(screen.getByText('预览')).toBeInTheDocument());
  });

  it('行内编辑输入单价与数量后，总金额列实时预览计算结果', async () => {
    const today = '2026-09-21';
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-unbilled-3',
          direction: RECEIVABLE,
          status: FEE_UNBILLED,
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

  it('选择费用项目并输入单价后，税率税金与不含税总额/不含税单价实时预览', async () => {
    const today = '2026-09-21';
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-unbilled-4',
          direction: RECEIVABLE,
          status: FEE_UNBILLED,
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

    const unitPriceInput = screen.getByPlaceholderText('0.00');
    await act(async () => {
      fireEvent.change(unitPriceInput, { target: { value: '106' } });
    });

    // 含税 106、税率 6%：不含税总额与不含税单价均为 100.00，税金 6.00
    await waitFor(() => {
      expect(screen.getAllByText('100.00').length).toBe(2);
      expect(screen.getByText('6.00')).toBeInTheDocument();
    });
  });

  it('行内新增行：缺少单价输入时不含税单价列显示待保存，输入后实时折算（A3）', async () => {
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
      screen.getByRole('button', { name: /新增应收费用/ }).click();
    });
    await waitFor(() => expect(screen.getByText('保存')).toBeInTheDocument());

    // 新行缺少单价与税率输入：税率、税金、不含税总额、折本币金额、
    // 不含税单价五个只读列均显示待保存
    await waitFor(() => expect(screen.getAllByText('待保存').length).toBe(5));

    const comboboxes = await waitFor(() => {
      const boxes = screen.getAllByRole('combobox');
      expect(boxes.length).toBeGreaterThanOrEqual(4);
      return boxes;
    });
    await pickSelectOption(comboboxes[0], '码头操作费 (THC)');

    // 仅选择费用项目（含默认税率）补齐税率预览，其余四列仍缺单价输入
    await waitFor(() => {
      expect(screen.getAllByText('6%').length).toBeGreaterThan(0);
    });
    expect(screen.getAllByText('待保存').length).toBe(4);

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
  });

  it('费用项目默认币种联动时立即解析并预览该币种参考汇率', async () => {
    const today = '2026-09-21';
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-unbilled-1',
          direction: RECEIVABLE,
          status: FEE_UNBILLED,
          currency: 'CNY',
          expenseDate: today,
          version: '3',
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
        { id: 'setting-of', feeCode: 'OF', nameZh: '海运费' },
        {
          id: 'setting-thc',
          feeCode: 'THC',
          nameZh: '码头操作费',
          defaultCurrency: 'USD',
        },
      ],
      settlementParties: [{ id: 'customer-1', name: '测试客户' }],
      currencies: [
        { code: 'CNY', name: '人民币' },
        { code: 'USD', name: '美元' },
      ],
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

    const comboboxes = await waitFor(() => {
      const boxes = screen.getAllByRole('combobox');
      expect(boxes.length).toBeGreaterThanOrEqual(4);
      return boxes;
    });
    resolveRate.mockResolvedValueOnce({
      exchangeRate: '7.1234',
      exchangeRateSource: 'WEEKLY',
    } as any);
    await pickSelectOption(comboboxes[0], '码头操作费 (THC)');

    await waitFor(() =>
      expect(resolveRate).toHaveBeenCalledWith(
        expect.objectContaining({
          orderId: 'order-1',
          direction: RECEIVABLE,
          currency: 'USD',
          expenseDate: today,
        }),
        expect.anything(),
      ),
    );
    await waitFor(() => {
      expect(screen.getByText('7.1234')).toBeInTheDocument();
      expect(screen.getByText('预览')).toBeInTheDocument();
      expect(screen.queryByText('沿用上周')).not.toBeInTheDocument();
    });
  });

  it('行内编辑的币种下拉支持按代码搜索过滤选项', async () => {
    const today = '2026-09-21';
    listFees.mockResolvedValue({
      data: [
        {
          id: 'fee-unbilled-2',
          direction: RECEIVABLE,
          status: FEE_UNBILLED,
          currency: 'CNY',
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
      currencies: [
        { code: 'CNY', name: '人民币' },
        { code: 'USD', name: '美元' },
        { code: 'EUR', name: '欧元' },
      ],
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

    const comboboxes = await waitFor(() => {
      const boxes = screen.getAllByRole('combobox');
      expect(boxes.length).toBeGreaterThanOrEqual(4);
      return boxes;
    });
    const currencyBox = comboboxes[2];
    expect(currencyBox).toHaveAttribute('aria-expanded', 'false');

    fireEvent.mouseDown(currencyBox);
    await waitFor(() => {
      expect(currencyBox).toHaveAttribute('aria-expanded', 'true');
    });
    fireEvent.change(currencyBox, { target: { value: 'US' } });

    const dropdown = await waitFor(() => {
      const found = screen
        .getAllByText('USD')
        .map((element) => element.closest('.ant-select-dropdown'))
        .filter(
          (dropdownElement) =>
            dropdownElement !== null &&
            !dropdownElement.className.includes('-hidden'),
        );
      expect(found.length).toBeGreaterThan(0);
      return found[0] as HTMLElement;
    });
    expect(within(dropdown).queryByText('EUR')).not.toBeInTheDocument();
    expect(within(dropdown).queryByText('CNY')).not.toBeInTheDocument();
  });
});
