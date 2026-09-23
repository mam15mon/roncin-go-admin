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
  orderFeeServiceBatchAssignOrderFeeTags,
  orderFeeServiceBatchRemoveOrderFeeTags,
  orderFeeServiceBulkRemoveOrderFees,
  orderFeeServiceBulkUpdateOrderFees,
  orderFeeServiceListFees,
  orderFeeServiceListOrderFeeTagOptions,
  orderFeeServiceResolveFeeExchangeRate,
} from '@/services/roncin/orderFeeService';
import { FEE_BILLED, FEE_UNBILLED, RECEIVABLE } from './feeConstants';
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
const resolveRate = vi.mocked(orderFeeServiceResolveFeeExchangeRate);
const bulkUpdate = vi.mocked(orderFeeServiceBulkUpdateOrderFees);
const bulkRemove = vi.mocked(orderFeeServiceBulkRemoveOrderFees);
const listTagOptions = vi.mocked(orderFeeServiceListOrderFeeTagOptions);
const batchAssignTags = vi.mocked(orderFeeServiceBatchAssignOrderFeeTags);
const batchRemoveTags = vi.mocked(orderFeeServiceBatchRemoveOrderFeeTags);

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

/** 模拟在 Dropdown 菜单中点击批量操作项。 */
async function triggerBulkAction(name: string | RegExp, tableIndex = 0) {
  const bulkButtons = screen.getAllByRole('button', { name: /批量操作/ });
  await act(async () => {
    fireEvent.click(bulkButtons[tableIndex]);
  });
  const menuItem = await screen.findByText(name);
  await act(async () => {
    fireEvent.click(menuItem);
  });
}

function makeFee(partial: Partial<API.OrderFee>): API.OrderFee {
  return {
    direction: RECEIVABLE,
    status: FEE_UNBILLED,
    currency: 'CNY',
    quantity: '1',
    unitPrice: '100',
    expenseDate: '2026-09-20 10:00',
    feeName: '海运费',
    ...partial,
  } as API.OrderFee;
}

const unbilledFeeA = makeFee({
  id: 'fee-a',
  version: '3',
  feeName: '海运费',
});
const unbilledFeeB = makeFee({
  id: 'fee-b',
  version: '1',
  feeName: '拖车费',
});
const billedFeeC = makeFee({
  id: 'fee-c',
  version: '5',
  status: FEE_BILLED,
  feeName: '已建账港杂费',
});

function makeProps(orderId: string) {
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

async function renderTabs(props: ReturnType<typeof makeProps>['props']) {
  render(
    <App>
      <OrderFeeTableTabs {...props} />
    </App>,
  );
  await waitFor(() => expect(listFees).toHaveBeenCalled());
}

function getDialog() {
  return screen.getByRole('dialog');
}

describe('OrderFeeTableTabs 批量维护操作', () => {
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
  });

  it('全选只收集已保存有效行：未保存新行复选框禁用（A2 回归）', async () => {
    listFees.mockResolvedValue({
      data: [
        unbilledFeeA,
        makeFee({ id: 'new_1700000000000', feeName: '未保存新行' }),
      ],
    } as any);
    const { props } = makeProps('order-1');
    await renderTabs(props);
    await screen.findByText('海运费');

    const rowCheckbox = (rowText: string) =>
      within(
        screen.getByText(rowText).closest('tr') as HTMLTableRowElement,
      ).getByRole('checkbox');

    // 未保存行（new_ 前缀且无版本）不可勾选
    expect(rowCheckbox('未保存新行')).toBeDisabled();
    expect(rowCheckbox('海运费')).not.toBeDisabled();

    const table = screen
      .getByText('海运费')
      .closest('table') as HTMLTableElement;
    const selectAll = within(
      table.querySelector('thead') as HTMLTableSectionElement,
    ).getByRole('checkbox');
    await act(async () => {
      fireEvent.click(selectAll);
    });

    // rowSelection.onChange 首参为选中行 key 集合：全选只收集可勾选的已保存行
    expect(props.setSelectedReceivableFeeIds.mock.calls.at(-1)?.[0]).toEqual([
      'fee-a',
    ]);
  });

  it('批量改结算单位：提交版本快照 targets 与目标单位，成功后清空选择并刷新（A2/A3）', async () => {
    bulkUpdate.mockResolvedValueOnce({ updatedCount: 2 } as any);
    listFees.mockResolvedValue({
      data: [unbilledFeeA, unbilledFeeB],
    } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-b', 'fee-a'];
    await renderTabs(props);
    await screen.findByText('海运费');
    // 成功后两表 reload 会重新发起费用查询
    const listCallsBefore = listFees.mock.calls.length;

    // 应收入口可用；应付表未勾选时入口禁用
    const bulkButtons = screen.getAllByRole('button', {
      name: /批量操作/,
    });
    expect(bulkButtons[0]).not.toBeDisabled();
    expect(bulkButtons[1]).toBeDisabled();

    await act(async () => {
      fireEvent.click(bulkButtons[0]);
    });
    const partyMenuItem = await screen.findByText('批量改结算单位');
    await act(async () => {
      fireEvent.click(partyMenuItem);
    });
    await waitFor(() =>
      expect(
        screen.getByText('批量修改结算单位（已选 2 笔应收费用）'),
      ).toBeInTheDocument(),
    );

    const combobox = within(getDialog()).getByRole('combobox');
    await pickSelectOption(combobox, '新结算单位');

    await act(async () => {
      within(getDialog()).getByRole('button', { name: '确定修改' }).click();
    });

    await waitFor(() =>
      expect(bulkUpdate).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        {
          orderId: 'order-1',
          targets: [
            { feeId: 'fee-a', expectedVersion: '3' },
            { feeId: 'fee-b', expectedVersion: '1' },
          ],
          settlementPartyId: 'party-new',
        },
      ),
    );
    expect(await screen.findByText('已修改 2 笔费用的结算单位')).toBeVisible();
    expect(props.setSelectedReceivableFeeIds).toHaveBeenCalledWith([]);
    await waitFor(() =>
      // 应收/应付两表 reload 重新拉取费用列表
      expect(listFees.mock.calls.length).toBeGreaterThan(listCallsBefore),
    );
    expect(props.onFeeSaved).toHaveBeenCalled();
  });

  it('批量删除：二次确认展示笔数，删除原因随请求传递', async () => {
    bulkRemove.mockResolvedValueOnce({ removedCount: 1 } as any);
    listFees.mockResolvedValue({ data: [unbilledFeeA] } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a'];
    await renderTabs(props);
    await screen.findByText('海运费');

    await triggerBulkAction('批量删除');
    await waitFor(() =>
      expect(
        screen.getByText('批量删除费用（已选 1 笔应收费用）'),
      ).toBeInTheDocument(),
    );
    expect(screen.getByText('将删除 1 笔应收费用')).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText(/删除原因/), {
      target: { value: '客户取消订单' },
    });
    await act(async () => {
      within(getDialog())
        .getByRole('button', { name: /删\s*除/ })
        .click();
    });

    await waitFor(() =>
      expect(bulkRemove).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        {
          orderId: 'order-1',
          targets: [{ feeId: 'fee-a', expectedVersion: '3' }],
          reason: '客户取消订单',
        },
      ),
    );
    expect(await screen.findByText('已删除 1 笔费用')).toBeVisible();
    expect(props.setSelectedReceivableFeeIds).toHaveBeenCalledWith([]);
  });

  it('批量改费用时间：按分钟精度提交 YYYY-MM-DD HH:mm', async () => {
    listFees.mockResolvedValue({ data: [unbilledFeeA] } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a'];
    await renderTabs(props);
    await screen.findByText('海运费');

    await triggerBulkAction('批量改费用时间');
    await waitFor(() =>
      expect(
        screen.getByText('批量修改费用时间（已选 1 笔应收费用）'),
      ).toBeInTheDocument(),
    );

    const pickerInput = screen.getByPlaceholderText('请选择费用发生时间');
    fireEvent.change(pickerInput, {
      target: { value: '2026-09-23 14:30' },
    });
    fireEvent.keyDown(pickerInput, { key: 'Enter' });
    await act(async () => {});

    await act(async () => {
      within(getDialog()).getByRole('button', { name: '确定修改' }).click();
    });

    await waitFor(() =>
      expect(bulkUpdate).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        {
          orderId: 'order-1',
          targets: [{ feeId: 'fee-a', expectedVersion: '3' }],
          expenseDate: '2026-09-23 14:30',
        },
      ),
    );
  });

  it('版本冲突整批失败：展示后端原因，保留选中且不刷新（A3）', async () => {
    bulkUpdate.mockRejectedValueOnce(
      new Error('费用 fee-a：已被更新，请刷新后重试'),
    );
    listFees.mockResolvedValue({ data: [unbilledFeeA, unbilledFeeB] } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a', 'fee-b'];
    await renderTabs(props);
    await screen.findByText('海运费');
    const listCallsBefore = listFees.mock.calls.length;

    await triggerBulkAction('批量改结算单位');
    await waitFor(() =>
      expect(
        screen.getByText('批量修改结算单位（已选 2 笔应收费用）'),
      ).toBeInTheDocument(),
    );
    await pickSelectOption(
      within(getDialog()).getByRole('combobox'),
      '新结算单位',
    );
    await act(async () => {
      within(getDialog()).getByRole('button', { name: '确定修改' }).click();
    });

    expect(
      await screen.findByText(/费用 fee-a：已被更新，请刷新后重试/),
    ).toBeVisible();
    // 失败：选择保留供核对，不触发刷新链路
    expect(props.setSelectedReceivableFeeIds).not.toHaveBeenCalledWith([]);
    expect(listFees.mock.calls.length).toBe(listCallsBefore);
    expect(props.onFeeSaved).not.toHaveBeenCalled();
    // 弹窗保持打开，用户可取消或调整后重试
    expect(
      screen.getByText('批量修改结算单位（已选 2 笔应收费用）'),
    ).toBeInTheDocument();
  });

  it('生成账单：勾选含已建账行时显式拒绝并列出不合格笔数', async () => {
    listFees.mockResolvedValue({
      data: [unbilledFeeA, billedFeeC],
    } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a', 'fee-c'];
    await renderTabs(props);
    await screen.findByText('海运费');

    await act(async () => {
      screen.getByRole('button', { name: /生成账单（2）/ }).click();
    });

    expect(
      await screen.findByText(
        /选中费用中有 1 笔已建账，不能生成账单（已建账港杂费），请取消勾选后重试/,
      ),
    ).toBeVisible();
    expect(props.onOpenBillWorkbench).not.toHaveBeenCalled();
  });

  it('批量标签：添加/移除两个入口按选中行提交，已建账行可参与（A2）', async () => {
    listFees.mockResolvedValue({
      data: [unbilledFeeA, billedFeeC],
    } as any);
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a', 'fee-c'];
    await renderTabs(props);
    await screen.findByText('海运费');

    // 应收入口可用；应付表未勾选时入口禁用
    const bulkButtons = screen.getAllByRole('button', {
      name: /批量操作/,
    });
    expect(bulkButtons[0]).not.toBeDisabled();
    expect(bulkButtons[1]).toBeDisabled();

    await triggerBulkAction('添加标签');
    const dialog = getDialog();
    await waitFor(() =>
      expect(within(dialog).getByText('已选 2 个对象')).toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(listTagOptions).toHaveBeenCalledWith(
        expect.objectContaining({ orderId: 'order-1', page: 1, pageSize: 50 }),
      ),
    );

    await pickSelectOption(within(dialog).getByRole('combobox'), '重点单');
    await act(async () => {
      within(dialog)
        .getByRole('button', { name: /添加标签/ })
        .click();
    });

    await waitFor(() =>
      expect(batchAssignTags).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        {
          orderId: 'order-1',
          feeIds: ['fee-a', 'fee-c'],
          tagIds: ['tag-1'],
        },
      ),
    );
    expect(await screen.findByText('已为 2 笔费用添加标签')).toBeVisible();
    expect(props.setSelectedReceivableFeeIds).toHaveBeenCalledWith([]);

    // 「删除标签」入口以移除模式重新打开同一弹窗（重开可能重建内容节点，
    // 断言时需重查 dialog，避免持有失联的旧节点）
    await triggerBulkAction('删除标签');
    await waitFor(() => {
      const okButton = within(getDialog()).getByRole('button', {
        name: /移除标签/,
      }) as HTMLButtonElement;
      // 等上一次提交的 loading 态完全复位，避免点击被吞
      expect(okButton.className).not.toContain('ant-btn-loading');
    });
    expect(
      within(getDialog()).queryByRole('button', { name: /添加标签/ }),
    ).not.toBeInTheDocument();

    await pickSelectOption(within(getDialog()).getByRole('combobox'), '重点单');
    await act(async () => {
      within(getDialog())
        .getByRole('button', { name: /移除标签/ })
        .click();
    });

    await waitFor(() =>
      expect(batchRemoveTags).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        {
          orderId: 'order-1',
          feeIds: ['fee-a', 'fee-c'],
          tagIds: ['tag-1'],
        },
      ),
    );
  });
});
