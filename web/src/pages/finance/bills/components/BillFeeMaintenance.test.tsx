import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { FinanceBillFeeLockFact, FinanceBillStatus } from '@/enums.generated';
import {
  settlementServiceGetBill,
  settlementServiceListBillFeeCandidates,
  settlementServiceUpdateBillFees,
} from '@/services/roncin/settlementService';
import BillFeeMaintenance from './BillFeeMaintenance';

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceGetBill: vi.fn(),
  settlementServiceListBillFeeCandidates: vi.fn(),
  settlementServiceUpdateBillFees: vi.fn(),
}));
const getBill = vi.mocked(settlementServiceGetBill);
const listCandidates = vi.mocked(settlementServiceListBillFeeCandidates);
const updateFees = vi.mocked(settlementServiceUpdateBillFees);

type GetBillResult = Awaited<ReturnType<typeof settlementServiceGetBill>>;
type CandidatesResult = Awaited<
  ReturnType<typeof settlementServiceListBillFeeCandidates>
>;
type UpdateResult = Awaited<ReturnType<typeof settlementServiceUpdateBillFees>>;

const lineColumns = [
  { title: '费用名称', dataIndex: 'feeName' },
  { title: '订单编号', dataIndex: 'orderNo' },
];

/**
 * 真实 wire 形态：ProtoJSON 省略 false 与空数组——有效行带 active=true，
 * 历史停用行省略 active；未锁定账单省略 feeLockFacts 与 feesLocked；版本为 uint64 字符串。
 */
function wireBill(overrides: Partial<API.FinanceBill> = {}): API.FinanceBill {
  return {
    id: 'bill-1',
    billNo: 'BILL-1',
    status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
    direction: 'RECEIVABLE',
    currency: 'USD',
    settlementPartyName: '客户甲',
    version: '3',
    feeCount: 2,
    lines: [
      { id: 'line-a', orderFeeId: 'fee-a', feeName: '海运费', active: true },
      { id: 'line-b', orderFeeId: 'fee-b', feeName: '订舱费', active: true },
      { id: 'line-old', orderFeeId: 'fee-old', feeName: '已释放费' },
    ],
    ...overrides,
  };
}

function mount(bill: API.FinanceBill) {
  const onUpdated = vi.fn();
  const onRefresh = vi.fn();
  const view = renderWithApp(
    <BillFeeMaintenance
      bill={bill}
      lineColumns={lineColumns}
      onUpdated={onUpdated}
      onRefresh={onRefresh}
    />,
  );
  return { ...view, onUpdated, onRefresh };
}

function removeButtons() {
  return screen.queryAllByRole('button', { name: '移除' });
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  getBill.mockResolvedValue({
    success: true,
    data: { id: 'bill-1', version: '3' },
  } as GetBillResult);
});
afterEach(() => {
  vi.restoreAllMocks();
});

describe('账单费用维护', () => {
  it('只展示有效费用，隐藏省略 active 或显式 false 的已释放行', async () => {
    const bill = wireBill();
    bill.lines?.push({
      id: 'line-false',
      feeName: '显式释放费',
      active: false,
    });
    mount(bill);
    expect(screen.getByText('海运费')).toBeInTheDocument();
    expect(screen.getByText('订舱费')).toBeInTheDocument();
    expect(screen.queryByText('已释放费')).not.toBeInTheDocument();
    expect(screen.queryByText('显式释放费')).not.toBeInTheDocument();
    expect(removeButtons()).toHaveLength(2);
    expect(removeButtons()[0]).toBeEnabled();
    expect(screen.getByRole('button', { name: '添加费用' })).toBeEnabled();
    expect(screen.queryByText(/不能增删费用/)).not.toBeInTheDocument();
  });

  it('存在锁定事实时置灰添加与移除并提示事实类型', () => {
    mount(
      wireBill({
        feesLocked: true,
        feeLockFacts: [
          FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_INVOICE,
          FinanceBillFeeLockFact.FINANCE_BILL_FEE_LOCK_FACT_NETTING_DRAFT,
        ],
      }),
    );
    expect(
      screen.getByText('账单已有发票、草稿对冲，不能增删费用'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '添加费用' })).toBeDisabled();
    for (const button of removeButtons()) expect(button).toBeDisabled();
  });

  it('仅剩一条有效费用时禁止移除', () => {
    mount(
      wireBill({
        lines: [
          {
            id: 'line-a',
            orderFeeId: 'fee-a',
            feeName: '海运费',
            active: true,
          },
        ],
      }),
    );
    expect(removeButtons()).toHaveLength(1);
    expect(removeButtons()[0]).toBeDisabled();
  });

  it('已取消账单不提供费用维护', () => {
    mount(
      wireBill({ status: FinanceBillStatus.FINANCE_BILL_STATUS_CANCELLED }),
    );
    expect(removeButtons()).toHaveLength(0);
    expect(
      screen.queryByRole('button', { name: '添加费用' }),
    ).not.toBeInTheDocument();
  });

  it('移除提交当前版本，成功后回传新账单', async () => {
    const updated = wireBill({
      version: '4',
      feeCount: 1,
      lines: [
        { id: 'line-a', orderFeeId: 'fee-a', feeName: '海运费' },
        { id: 'line-b', orderFeeId: 'fee-b', feeName: '订舱费', active: true },
      ],
    });
    updateFees.mockResolvedValue({
      success: true,
      data: updated,
    } as UpdateResult);
    const { onUpdated, rerender } = mount(wireBill());
    fireEvent.click(removeButtons()[0]);
    fireEvent.click(await screen.findByRole('button', { name: /确\s*定|OK/ }));
    await waitFor(() => expect(onUpdated).toHaveBeenCalledWith(updated));
    rerender(
      <BillFeeMaintenance
        bill={updated}
        lineColumns={lineColumns}
        onUpdated={onUpdated}
        onRefresh={vi.fn()}
      />,
    );
    expect(screen.queryByText('海运费')).not.toBeInTheDocument();
    expect(screen.getByText('订舱费')).toBeInTheDocument();
    expect(removeButtons()).toHaveLength(1);
    expect(removeButtons()[0]).toBeDisabled();
    expect(updateFees).toHaveBeenCalledWith(
      { id: 'bill-1' },
      {
        id: 'bill-1',
        expectedVersion: '3',
        addFeeIds: [],
        removeFeeIds: ['fee-a'],
      },
    );
  });

  it('移除失败保留确认框，允许按当前版本重试', async () => {
    updateFees.mockResolvedValueOnce(undefined as unknown as UpdateResult);
    const { onUpdated } = mount(wireBill());
    fireEvent.click(removeButtons()[0]);
    fireEvent.click(await screen.findByRole('button', { name: /确\s*定|OK/ }));
    await waitFor(() => expect(updateFees).toHaveBeenCalledTimes(1));
    expect(onUpdated).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /确\s*定|OK/ })).toBeEnabled(),
    );
    expect(screen.getByText('确认从账单移除该费用？')).toBeInTheDocument();
  });

  it('切换单据卸载后，旧移除请求的迟到成功不再回填父详情', async () => {
    let resolveUpdate: ((response: UpdateResult) => void) | undefined;
    updateFees.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveUpdate = resolve;
        }),
    );
    const { onUpdated, unmount } = mount(wireBill());
    fireEvent.click(removeButtons()[0]);
    fireEvent.click(await screen.findByRole('button', { name: /确\s*定|OK/ }));
    await waitFor(() => expect(updateFees).toHaveBeenCalledTimes(1));
    unmount();
    await act(async () => {
      resolveUpdate?.({
        success: true,
        data: wireBill({ version: '4' }),
      } as UpdateResult);
    });
    expect(onUpdated).not.toHaveBeenCalled();
  });

  it('候选由服务端关键字过滤分页，多选后一次提交；失败保留选择', async () => {
    listCandidates.mockImplementation(
      async (params) =>
        ({
          success: true,
          // ProtoJSON：canBill=false 省略，total 为 int64 字符串。
          total: '2',
          data:
            params.keyword === '订舱'
              ? [{ id: 'fee-c', feeName: '订舱附加', canBill: true }]
              : [
                  { id: 'fee-c', feeName: '订舱附加', canBill: true },
                  {
                    id: 'fee-d',
                    feeName: '不可选',
                    blockedReasons: ['口径不符'],
                  },
                ],
        }) as CandidatesResult,
    );
    updateFees.mockResolvedValueOnce(undefined as unknown as UpdateResult);
    const { onUpdated } = mount(wireBill());
    fireEvent.click(screen.getByRole('button', { name: '添加费用' }));
    const dialog = await screen.findByRole('dialog');
    await within(dialog).findByText('订舱附加');
    expect(listCandidates).toHaveBeenLastCalledWith(
      { billId: 'bill-1', keyword: undefined, page: 1, pageSize: 20 },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    const checkboxes = within(dialog).getAllByRole('checkbox');
    // 表头全选 + 两行；不可加入的候选禁用。
    expect(checkboxes[2]).toBeDisabled();
    fireEvent.change(within(dialog).getByPlaceholderText(/费用代码/), {
      target: { value: ' 订舱 ' },
    });
    fireEvent.keyDown(within(dialog).getByPlaceholderText(/费用代码/), {
      key: 'Enter',
      code: 'Enter',
      keyCode: 13,
    });
    await waitFor(() =>
      expect(listCandidates).toHaveBeenLastCalledWith(
        { billId: 'bill-1', keyword: '订舱', page: 1, pageSize: 20 },
        expect.anything(),
      ),
    );
    await waitFor(() =>
      expect(within(dialog).getAllByRole('checkbox')).toHaveLength(2),
    );
    fireEvent.click(within(dialog).getAllByRole('checkbox')[1]);
    const okButton = within(dialog).getByRole('button', { name: /添加 1 笔/ });
    fireEvent.click(okButton);
    await waitFor(() => expect(updateFees).toHaveBeenCalledTimes(1));
    expect(updateFees).toHaveBeenLastCalledWith(
      { id: 'bill-1' },
      {
        id: 'bill-1',
        expectedVersion: '3',
        addFeeIds: ['fee-c'],
        removeFeeIds: [],
      },
    );
    // 请求失败（统一错误处理返回 undefined）：不回传、弹窗与勾选保留。
    expect(onUpdated).not.toHaveBeenCalled();
    expect(
      within(dialog).getByRole('button', { name: /添加 1 笔/ }),
    ).toBeInTheDocument();
  });

  it('探测到更新版本时提醒，无未提交勾选时直接刷新', async () => {
    getBill.mockResolvedValue({
      success: true,
      data: { id: 'bill-1', version: '5' },
    } as GetBillResult);
    const { onRefresh } = mount(wireBill());
    expect(
      await screen.findByText('这个账单已经发生变动，请重新刷新。'),
    ).toBeInTheDocument();
    expect(getBill).toHaveBeenCalledWith(
      { id: 'bill-1' },
      { signal: expect.any(AbortSignal), skipErrorHandler: true },
    );
    fireEvent.click(screen.getByRole('button', { name: /刷\s*新/ }));
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it('探测版本不新于当前详情时不提醒', async () => {
    mount(wireBill());
    await waitFor(() => expect(getBill).toHaveBeenCalled());
    expect(
      screen.queryByText('这个账单已经发生变动，请重新刷新。'),
    ).not.toBeInTheDocument();
  });
});
